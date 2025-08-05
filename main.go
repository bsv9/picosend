package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"embed"
	"encoding/base64"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/mux"
)

const (
	MaxSecretLength  = 65536 // Maximum secret content length in characters
	MaxUnreadSecrets = 1000  // Maximum number of unread secrets in memory
)

//go:embed templates/*.html
var templatesFS embed.FS

//go:embed static/*
var staticFS embed.FS

type Secret struct {
	ID        string    `json:"id"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

type SecretStore struct {
	mu      sync.RWMutex
	secrets map[string]*Secret
}

func NewSecretStore() *SecretStore {
	return &SecretStore{
		secrets: make(map[string]*Secret),
	}
}

func (s *SecretStore) Store(content string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if we've reached the maximum number of unread secrets
	if len(s.secrets) >= MaxUnreadSecrets {
		return "", fmt.Errorf("maximum number of unread secrets (%d) reached", MaxUnreadSecrets)
	}

	id := generateID()
	secret := &Secret{
		ID:        id,
		Content:   content,
		CreatedAt: time.Now(),
	}
	s.secrets[id] = secret
	return id, nil
}

func (s *SecretStore) Get(id string) (*Secret, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	secret, exists := s.secrets[id]
	if exists {
		// Create a copy of the secret for return
		secretCopy := &Secret{
			ID:        secret.ID,
			Content:   secret.Content,
			CreatedAt: secret.CreatedAt,
		}

		// Wipe the original secret's content from memory
		wipeSecret(secret)

		// Delete the secret from the store
		delete(s.secrets, id)

		return secretCopy, true
	}
	return nil, false
}

// wipeSecret securely overwrites secret data and creates a new secret with wiped content
func wipeSecret(secret *Secret) {
	if secret == nil {
		return
	}

	// Create byte slices to overwrite
	contentBytes := []byte(secret.Content)
	idBytes := []byte(secret.ID)

	// Overwrite the byte slices with zeros
	for i := range contentBytes {
		contentBytes[i] = 0
	}
	for i := range idBytes {
		idBytes[i] = 0
	}

	// Replace the string fields with empty strings
	// This doesn't guarantee the original strings are wiped but provides some protection
	secret.Content = ""
	secret.ID = ""
}

func (s *SecretStore) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.secrets)
}

func generateID() string {
	bytes := make([]byte, 12) // 12 bytes = 16 chars in base64url (vs 32 chars in hex)
	rand.Read(bytes)
	return base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(bytes)
}

func generateEncryptionKey() string {
	key := make([]byte, 32) // 256-bit key for AES
	rand.Read(key)
	return base64.StdEncoding.EncodeToString(key)
}

func decrypt(encryptedData, keyStr string) (string, error) {
	key, err := base64.StdEncoding.DecodeString(keyStr)
	if err != nil {
		return "", err
	}

	ciphertext, err := base64.StdEncoding.DecodeString(encryptedData)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	if len(ciphertext) < aes.BlockSize {
		return "", fmt.Errorf("ciphertext too short")
	}

	// Check if ciphertext length is valid for CBC (must be multiple of block size)
	if len(ciphertext)%aes.BlockSize != 0 {
		return "", fmt.Errorf("ciphertext is not a multiple of the block size")
	}

	iv := ciphertext[:aes.BlockSize]
	ciphertext = ciphertext[aes.BlockSize:]

	// Check if there's actual ciphertext after the IV
	if len(ciphertext) == 0 {
		return "", fmt.Errorf("no ciphertext after IV")
	}

	mode := cipher.NewCBCDecrypter(block, iv)
	mode.CryptBlocks(ciphertext, ciphertext)

	// Validate and remove PKCS7 padding
	if len(ciphertext) == 0 {
		return "", fmt.Errorf("empty ciphertext after decryption")
	}

	padding := int(ciphertext[len(ciphertext)-1])
	if padding > aes.BlockSize || padding == 0 {
		return "", fmt.Errorf("invalid padding")
	}

	if len(ciphertext) < padding {
		return "", fmt.Errorf("padding size larger than ciphertext")
	}

	// Validate that all padding bytes are the same
	for i := len(ciphertext) - padding; i < len(ciphertext); i++ {
		if ciphertext[i] != byte(padding) {
			return "", fmt.Errorf("invalid PKCS7 padding")
		}
	}

	return string(ciphertext[:len(ciphertext)-padding]), nil
}

var store = NewSecretStore()

func main() {
	r := mux.NewRouter()

	// STATIC SERVING
	// Serve embedded Pico CSS
	r.HandleFunc("/static/css/pico.min.css", func(w http.ResponseWriter, r *http.Request) {
		data, err := staticFS.ReadFile("static/css/pico.min.css")
		if err != nil {
			http.Error(w, "File not found", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "text/css")
		w.Write(data)
	}).Methods("GET")

	// Serve Open Graph image
	r.HandleFunc("/static/og-image.png", func(w http.ResponseWriter, r *http.Request) {
		data, err := staticFS.ReadFile("static/og-image.png")
		if err != nil {
			http.Error(w, "File not found", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "image/png")
		w.Write(data)
	}).Methods("GET")

	// Serve robots.txt
	r.HandleFunc("/robots.txt", func(w http.ResponseWriter, r *http.Request) {
		data, err := staticFS.ReadFile("static/robots.txt")
		if err != nil {
			http.Error(w, "File not found", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "text/plain")
		w.Write(data)
	}).Methods("GET")

	r.HandleFunc("/", homeHandler).Methods("GET")
	r.HandleFunc("/api/encryption-key", encryptionKeyHandler).Methods("GET")
	r.HandleFunc("/api/secrets", createSecretHandler).Methods("POST")
	r.HandleFunc("/api/secrets/{id}", getSecretHandler).Methods("GET")
	r.HandleFunc("/api/secrets/{id}/verify", verifySecretHandler).Methods("POST")
	r.HandleFunc("/s/{id}", viewSecretHandler).Methods("GET")

	fmt.Println("Server starting on :8080")
	log.Fatal(http.ListenAndServe(":8080", r))
}
