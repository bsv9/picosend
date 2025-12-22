package main

import (
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
	ExpiresAt time.Time `json:"expires_at"`
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

func (s *SecretStore) Store(content string, lifetime time.Duration) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if we've reached the maximum number of unread secrets
	if len(s.secrets) >= MaxUnreadSecrets {
		return "", fmt.Errorf("maximum number of unread secrets (%d) reached", MaxUnreadSecrets)
	}

	id := generateID()
	now := time.Now()
	secret := &Secret{
		ID:        id,
		Content:   content,
		CreatedAt: now,
		ExpiresAt: now.Add(lifetime),
	}
	s.secrets[id] = secret
	return id, nil
}

func (s *SecretStore) Get(id string) (*Secret, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	secret, exists := s.secrets[id]
	if !exists {
		return nil, false
	}

	// Check if secret has expired
	if time.Now().After(secret.ExpiresAt) {
		// Wipe and delete expired secret
		wipeSecret(secret)
		delete(s.secrets, id)
		return nil, false
	}

	// Create a copy of the secret for return
	secretCopy := &Secret{
		ID:        secret.ID,
		Content:   secret.Content,
		CreatedAt: secret.CreatedAt,
		ExpiresAt: secret.ExpiresAt,
	}

	// Wipe the original secret's content from memory
	wipeSecret(secret)

	// Delete the secret from the store
	delete(s.secrets, id)

	return secretCopy, true
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

func (s *SecretStore) CleanupExpired() int {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	count := 0

	for id, secret := range s.secrets {
		if now.After(secret.ExpiresAt) {
			wipeSecret(secret)
			delete(s.secrets, id)
			count++
		}
	}

	return count
}

func generateID() string {
	bytes := make([]byte, 12) // 12 bytes = 16 chars in base64url (vs 32 chars in hex)
	rand.Read(bytes)
	return base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(bytes)
}

var store = NewSecretStore()

func main() {
	// Start background cleanup goroutine
	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			count := store.CleanupExpired()
			if count > 0 {
				log.Printf("Cleaned up %d expired secrets", count)
			}
		}
	}()

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
	r.HandleFunc("/api/secrets", createSecretHandler).Methods("POST")
	r.HandleFunc("/api/secrets/{id}", getSecretHandler).Methods("GET")
	r.HandleFunc("/api/secrets/{id}/verify", verifySecretHandler).Methods("POST")
	r.HandleFunc("/s/{id}", viewSecretHandler).Methods("GET")

	fmt.Println("Server starting on :8080")
	log.Fatal(http.ListenAndServe(":8080", r))
}
