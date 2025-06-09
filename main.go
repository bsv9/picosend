package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"embed"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/mux"
)

//go:embed static/css/pico.min.css
var picoCSS embed.FS

//go:embed templates/*.html
var templatesFS embed.FS

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

func (s *SecretStore) Store(content string) string {
	s.mu.Lock()
	defer s.mu.Unlock()

	id := generateID()
	secret := &Secret{
		ID:        id,
		Content:   content,
		CreatedAt: time.Now(),
	}
	s.secrets[id] = secret
	return id
}

func (s *SecretStore) Get(id string) (*Secret, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	secret, exists := s.secrets[id]
	if exists {
		delete(s.secrets, id)
	}
	return secret, exists
}

func generateID() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
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
	
	return string(ciphertext[:len(ciphertext)-padding]), nil
}

var store = NewSecretStore()

func picoCSSHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/css")
	data, err := picoCSS.ReadFile("static/css/pico.min.css")
	if err != nil {
		http.Error(w, "CSS file not found", http.StatusNotFound)
		return
	}
	w.Write(data)
}

func main() {
	r := mux.NewRouter()

	// Serve embedded Pico CSS
	r.HandleFunc("/static/css/pico.min.css", picoCSSHandler).Methods("GET")

	r.HandleFunc("/", homeHandler).Methods("GET")
	r.HandleFunc("/api/encryption-key", encryptionKeyHandler).Methods("GET")
	r.HandleFunc("/api/secrets", createSecretHandler).Methods("POST")
	r.HandleFunc("/api/secrets/{id}", getSecretHandler).Methods("GET")
	r.HandleFunc("/api/secrets/{id}/verify", verifySecretHandler).Methods("POST")
	r.HandleFunc("/s/{id}", viewSecretHandler).Methods("GET")

	fmt.Println("Server starting on :8080")
	log.Fatal(http.ListenAndServe(":8080", r))
}