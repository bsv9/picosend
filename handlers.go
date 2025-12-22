package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"
)

type CreateSecretRequest struct {
	Content  string `json:"content"`
	Lifetime int    `json:"lifetime"` // Lifetime in minutes
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

	// Validate encrypted content length (base64 encoded, so can be larger than plaintext)
	if len(req.Content) > MaxSecretLength*2 {
		http.Error(w, fmt.Sprintf("Content exceeds maximum length of %d characters", MaxSecretLength*2), http.StatusBadRequest)
		return
	}

	// Parse lifetime (default to 24 hours if not specified or invalid)
	lifetime := time.Duration(req.Lifetime) * time.Minute
	if req.Lifetime <= 0 {
		lifetime = 24 * time.Hour
	}

	// Store encrypted content as-is (no decryption on server)
	id, err := store.Store(req.Content, lifetime)
	if err != nil {
		http.Error(w, err.Error(), http.StatusTooManyRequests)
		return
	}

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

