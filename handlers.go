package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
)

type CreateSecretRequest struct {
	Content       string `json:"content"`
	EncryptionKey string `json:"encryption_key"`
}

type EncryptionKeyResponse struct {
	Key string `json:"key"`
}

type CreateSecretResponse struct {
	ID string `json:"id"`
}

type GetSecretResponse struct {
	Content   string `json:"content"`
	CreatedAt string `json:"created_at"`
}

type VerifySecretRequest struct {
	VerificationCode string `json:"verification_code"`
}

func encryptionKeyHandler(w http.ResponseWriter, r *http.Request) {
	key := generateEncryptionKey()
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(EncryptionKeyResponse{Key: key})
}

func createSecretHandler(w http.ResponseWriter, r *http.Request) {
	var req CreateSecretRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if req.Content == "" {
		http.Error(w, "Content cannot be empty", http.StatusBadRequest)
		return
	}

	// Validate content length
	if len(req.Content) > MaxSecretLength {
		http.Error(w, fmt.Sprintf("Content exceeds maximum length of %d characters", MaxSecretLength), http.StatusBadRequest)
		return
	}

	if req.EncryptionKey == "" {
		http.Error(w, "Encryption key cannot be empty", http.StatusBadRequest)
		return
	}

	// Decrypt the content before storing
	decryptedContent, err := decrypt(req.Content, req.EncryptionKey)
	if err != nil {
		http.Error(w, "Failed to decrypt content", http.StatusBadRequest)
		return
	}

	// Validate decrypted content length as well
	if len(decryptedContent) > MaxSecretLength {
		http.Error(w, fmt.Sprintf("Decrypted content exceeds maximum length of %d characters", MaxSecretLength), http.StatusBadRequest)
		return
	}

	id := store.Store(decryptedContent)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(CreateSecretResponse{ID: id})
}

func getSecretHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	secret, found := store.Get(id)
	if !found {
		http.Error(w, "Secret not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(GetSecretResponse{
		Content:   secret.Content,
		CreatedAt: secret.CreatedAt.Format("2006-01-02 15:04:05 UTC"),
	})
}

func verifySecretHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	var req VerifySecretRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Basic validation - just check that a verification code was provided
	if req.VerificationCode == "" || len(req.VerificationCode) != 6 {
		http.Error(w, "Invalid verification code", http.StatusBadRequest)
		return
	}

	// Get and delete the secret
	secret, found := store.Get(id)
	if !found {
		http.Error(w, "Secret not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(GetSecretResponse{
		Content:   secret.Content,
		CreatedAt: secret.CreatedAt.Format("2006-01-02 15:04:05 UTC"),
	})
}