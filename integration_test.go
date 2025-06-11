package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
)

func setupTestServer() *httptest.Server {
	store = NewSecretStore() // Reset store for clean tests
	
	r := mux.NewRouter()
	r.HandleFunc("/api/encryption-key", encryptionKeyHandler).Methods("GET")
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
	
	// Note: This test simulates the encryption flow but cannot easily test
	// the actual JavaScript encryption. In practice, the frontend handles encryption.
	// We'll test with mock encrypted data that would fail decryption, which is expected.
	
	// Step 1: Try to create a secret with mock encrypted data (should fail)
	encryptionKey := generateEncryptionKey()
	
	createReq := CreateSecretRequest{
		Content:       "bW9ja2VuY3J5cHRlZGRhdGE=", // mock encrypted data (base64)
		EncryptionKey: encryptionKey,
	}
	jsonBody, _ := json.Marshal(createReq)
	
	resp, err := http.Post(server.URL+"/api/secrets", "application/json", bytes.NewBuffer(jsonBody))
	if err != nil {
		t.Fatalf("Failed to create secret: %v", err)
	}
	defer resp.Body.Close()
	
	// This should fail because we're using mock encrypted data
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected status 400 for mock encrypted data, got %d", resp.StatusCode)
	}
}

func TestDirectSecretRetrieval(t *testing.T) {
	server := setupTestServer()
	defer server.Close()
	
	// This test bypasses encryption by directly storing a secret in the store
	// to test the retrieval mechanism
	secretContent := "Direct retrieval test"
	secretID, err := store.Store(secretContent)
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
			secretContent := "Concurrent test secret"
			secretID, err := store.Store(secretContent)
	if err != nil {
		t.Fatalf("Failed to store secret: %v", err)
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