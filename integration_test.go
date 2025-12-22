package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/mux"
)

func setupTestServer() *httptest.Server {
	store = NewSecretStore() // Reset store for clean tests

	r := mux.NewRouter()
	r.HandleFunc("/api/secrets", createSecretHandler).Methods("POST")
	r.HandleFunc("/api/secrets/{id}", getSecretHandler).Methods("GET")
	r.HandleFunc("/api/secrets/{id}/verify", verifySecretHandler).Methods("POST")
	r.HandleFunc("/s/{id}", viewSecretHandler).Methods("GET")
	r.HandleFunc("/", homeHandler).Methods("GET")

	return httptest.NewServer(r)
}

func TestFullSecretFlow(t *testing.T) {
	server := setupTestServer()
	defer server.Close()

	// With client-side encryption, the server just stores encrypted content as-is
	// We simulate this by sending base64 encoded "encrypted" content
	encryptedContent := base64.StdEncoding.EncodeToString([]byte("mock encrypted content"))

	createReq := CreateSecretRequest{
		Content:  encryptedContent,
		Lifetime: 60,
	}
	jsonBody, _ := json.Marshal(createReq)

	resp, err := http.Post(server.URL+"/api/secrets", "application/json", bytes.NewBuffer(jsonBody))
	if err != nil {
		t.Fatalf("Failed to create secret: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var createResp CreateSecretResponse
	err = json.NewDecoder(resp.Body).Decode(&createResp)
	if err != nil {
		t.Fatalf("Failed to decode create response: %v", err)
	}

	// Verify the secret via the verify endpoint
	verifyReq := VerifySecretRequest{VerificationCode: "ABC123"}
	verifyJsonBody, _ := json.Marshal(verifyReq)

	verifyResp, err := http.Post(server.URL+"/api/secrets/"+createResp.ID+"/verify", "application/json", bytes.NewBuffer(verifyJsonBody))
	if err != nil {
		t.Fatalf("Failed to verify secret: %v", err)
	}
	defer verifyResp.Body.Close()

	if verifyResp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200 for verify, got %d", verifyResp.StatusCode)
	}

	var getResp GetSecretResponse
	err = json.NewDecoder(verifyResp.Body).Decode(&getResp)
	if err != nil {
		t.Fatalf("Failed to decode verify response: %v", err)
	}

	// Content should be returned as-is (encrypted)
	if getResp.Content != encryptedContent {
		t.Errorf("Expected encrypted content '%s', got '%s'", encryptedContent, getResp.Content)
	}

	// Try to access again - should fail since secret is deleted
	secondResp, err := http.Post(server.URL+"/api/secrets/"+createResp.ID+"/verify", "application/json", bytes.NewBuffer(verifyJsonBody))
	if err != nil {
		t.Fatalf("Failed to make second verify request: %v", err)
	}
	defer secondResp.Body.Close()

	if secondResp.StatusCode != http.StatusNotFound {
		t.Errorf("Expected status 404 for second access, got %d", secondResp.StatusCode)
	}
}

func TestDirectSecretRetrieval(t *testing.T) {
	server := setupTestServer()
	defer server.Close()

	// This test bypasses encryption by directly storing a secret in the store
	// to test the retrieval mechanism
	secretContent := base64.StdEncoding.EncodeToString([]byte("Direct retrieval test"))
	secretID, err := store.Store(secretContent, 24*time.Hour)
	if err != nil {
		t.Fatalf("Failed to store secret: %v", err)
	}

	// Test direct GET retrieval
	resp, err := http.Get(server.URL + "/api/secrets/" + secretID)
	if err != nil {
		t.Fatalf("Failed to get secret: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var getResp GetSecretResponse
	err = json.NewDecoder(resp.Body).Decode(&getResp)
	if err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Content should be returned as-is (encrypted)
	if getResp.Content != secretContent {
		t.Errorf("Expected content '%s', got '%s'", secretContent, getResp.Content)
	}
}

func TestHomePageHandler(t *testing.T) {
	server := setupTestServer()
	defer server.Close()

	resp, err := http.Get(server.URL + "/")
	if err != nil {
		t.Fatalf("Failed to get home page: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType != "text/html; charset=utf-8" {
		t.Errorf("Expected HTML content type, got %s", contentType)
	}
}

func TestViewSecretPageHandler(t *testing.T) {
	server := setupTestServer()
	defer server.Close()

	// Test with any ID (page should load regardless)
	resp, err := http.Get(server.URL + "/s/test123")
	if err != nil {
		t.Fatalf("Failed to get view secret page: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType != "text/html; charset=utf-8" {
		t.Errorf("Expected HTML content type, got %s", contentType)
	}
}

func TestConcurrentSecretOperations(t *testing.T) {
	server := setupTestServer()
	defer server.Close()

	const numSecrets = 10
	secretIDs := make([]string, numSecrets)

	// Create multiple secrets directly in store for testing concurrent access
	done := make(chan bool, numSecrets)
	for i := 0; i < numSecrets; i++ {
		go func(index int) {
			secretContent := base64.StdEncoding.EncodeToString([]byte("Concurrent test secret"))
			secretID, err := store.Store(secretContent, 24*time.Hour)
			if err != nil {
				t.Errorf("Failed to store secret: %v", err)
			}
			secretIDs[index] = secretID
			done <- true
		}(i)
	}

	// Wait for all creations to complete
	for i := 0; i < numSecrets; i++ {
		<-done
	}

	// Retrieve all secrets concurrently
	for i := 0; i < numSecrets; i++ {
		go func(index int) {
			if secretIDs[index] == "" {
				done <- false
				return
			}

			resp, err := http.Get(server.URL + "/api/secrets/" + secretIDs[index])
			if err != nil {
				t.Errorf("Failed to get secret %d: %v", index, err)
				done <- false
				return
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				t.Errorf("Expected status 200 for secret %d, got %d", index, resp.StatusCode)
				done <- false
				return
			}

			done <- true
		}(i)
	}

	// Wait for all retrievals to complete
	for i := 0; i < numSecrets; i++ {
		<-done
	}
}
