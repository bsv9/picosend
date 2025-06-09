package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/mux"
)

func TestCreateSecretHandler(t *testing.T) {
	// Test valid request with encryption
	encryptionKey := generateEncryptionKey()
	
	// Create a mock encrypted content for testing
	// Note: In real usage, this would be properly encrypted by the frontend
	reqBody := CreateSecretRequest{
		Content:       "dGVzdCBzZWNyZXQ=", // base64 of "test secret" - this will fail decryption as expected
		EncryptionKey: encryptionKey,
	}
	jsonBody, _ := json.Marshal(reqBody)
	
	req := httptest.NewRequest("POST", "/api/secrets", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	
	createSecretHandler(w, req)
	
	// This should fail because we're not providing properly encrypted data
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for improperly encrypted data, got %d", w.Code)
	}
}

func TestCreateSecretHandler_EmptyContent(t *testing.T) {
	reqBody := CreateSecretRequest{
		Content:       "",
		EncryptionKey: generateEncryptionKey(),
	}
	jsonBody, _ := json.Marshal(reqBody)
	
	req := httptest.NewRequest("POST", "/api/secrets", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	
	createSecretHandler(w, req)
	
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestCreateSecretHandler_MissingEncryptionKey(t *testing.T) {
	reqBody := CreateSecretRequest{
		Content:       "some content",
		EncryptionKey: "",
	}
	jsonBody, _ := json.Marshal(reqBody)
	
	req := httptest.NewRequest("POST", "/api/secrets", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	
	createSecretHandler(w, req)
	
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestCreateSecretHandler_InvalidJSON(t *testing.T) {
	req := httptest.NewRequest("POST", "/api/secrets", strings.NewReader("invalid json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	
	createSecretHandler(w, req)
	
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestGetSecretHandler(t *testing.T) {
	// First create a secret
	store = NewSecretStore() // Reset store for clean test
	secretContent := "test secret content"
	secretID := store.Store(secretContent)
	
	// Test retrieving the secret
	req := httptest.NewRequest("GET", "/api/secrets/"+secretID, nil)
	w := httptest.NewRecorder()
	
	// Setup mux vars
	req = mux.SetURLVars(req, map[string]string{"id": secretID})
	
	getSecretHandler(w, req)
	
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
	
	var response GetSecretResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Errorf("Failed to parse response: %v", err)
	}
	
	if response.Content != secretContent {
		t.Errorf("Expected content '%s', got '%s'", secretContent, response.Content)
	}
	
	if response.CreatedAt == "" {
		t.Error("Expected non-empty CreatedAt")
	}
}

func TestGetSecretHandler_NotFound(t *testing.T) {
	store = NewSecretStore() // Reset store for clean test
	
	req := httptest.NewRequest("GET", "/api/secrets/nonexistent", nil)
	w := httptest.NewRecorder()
	
	// Setup mux vars
	req = mux.SetURLVars(req, map[string]string{"id": "nonexistent"})
	
	getSecretHandler(w, req)
	
	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

func TestGetSecretHandler_OnlyOnce(t *testing.T) {
	// First create a secret
	store = NewSecretStore() // Reset store for clean test
	secretContent := "test secret content"
	secretID := store.Store(secretContent)
	
	// First retrieval should succeed
	req1 := httptest.NewRequest("GET", "/api/secrets/"+secretID, nil)
	w1 := httptest.NewRecorder()
	req1 = mux.SetURLVars(req1, map[string]string{"id": secretID})
	
	getSecretHandler(w1, req1)
	
	if w1.Code != http.StatusOK {
		t.Errorf("Expected status 200 on first retrieval, got %d", w1.Code)
	}
	
	// Second retrieval should fail
	req2 := httptest.NewRequest("GET", "/api/secrets/"+secretID, nil)
	w2 := httptest.NewRecorder()
	req2 = mux.SetURLVars(req2, map[string]string{"id": secretID})
	
	getSecretHandler(w2, req2)
	
	if w2.Code != http.StatusNotFound {
		t.Errorf("Expected status 404 on second retrieval, got %d", w2.Code)
	}
}

func TestVerifySecretHandler(t *testing.T) {
	// First create a secret
	store = NewSecretStore() // Reset store for clean test
	secretContent := "test secret content"
	secretID := store.Store(secretContent)
	
	// Test verify endpoint
	reqBody := VerifySecretRequest{VerificationCode: "ABC123"}
	jsonBody, _ := json.Marshal(reqBody)
	
	req := httptest.NewRequest("POST", "/api/secrets/"+secretID+"/verify", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	
	// Setup mux vars
	req = mux.SetURLVars(req, map[string]string{"id": secretID})
	
	verifySecretHandler(w, req)
	
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
	
	var response GetSecretResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Errorf("Failed to parse response: %v", err)
	}
	
	if response.Content != secretContent {
		t.Errorf("Expected content '%s', got '%s'", secretContent, response.Content)
	}
}

func TestVerifySecretHandler_InvalidCode(t *testing.T) {
	store = NewSecretStore() // Reset store for clean test
	secretID := store.Store("test content")
	
	// Test with invalid code (too short)
	reqBody := VerifySecretRequest{VerificationCode: "ABC"}
	jsonBody, _ := json.Marshal(reqBody)
	
	req := httptest.NewRequest("POST", "/api/secrets/"+secretID+"/verify", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	
	req = mux.SetURLVars(req, map[string]string{"id": secretID})
	
	verifySecretHandler(w, req)
	
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestVerifySecretHandler_EmptyCode(t *testing.T) {
	store = NewSecretStore() // Reset store for clean test
	secretID := store.Store("test content")
	
	// Test with empty code
	reqBody := VerifySecretRequest{VerificationCode: ""}
	jsonBody, _ := json.Marshal(reqBody)
	
	req := httptest.NewRequest("POST", "/api/secrets/"+secretID+"/verify", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	
	req = mux.SetURLVars(req, map[string]string{"id": secretID})
	
	verifySecretHandler(w, req)
	
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestVerifySecretHandler_NotFound(t *testing.T) {
	store = NewSecretStore() // Reset store for clean test
	
	reqBody := VerifySecretRequest{VerificationCode: "ABC123"}
	jsonBody, _ := json.Marshal(reqBody)
	
	req := httptest.NewRequest("POST", "/api/secrets/nonexistent/verify", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	
	req = mux.SetURLVars(req, map[string]string{"id": "nonexistent"})
	
	verifySecretHandler(w, req)
	
	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

func TestVerifySecretHandler_InvalidJSON(t *testing.T) {
	store = NewSecretStore() // Reset store for clean test
	secretID := store.Store("test content")
	
	req := httptest.NewRequest("POST", "/api/secrets/"+secretID+"/verify", strings.NewReader("invalid json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	
	req = mux.SetURLVars(req, map[string]string{"id": secretID})
	
	verifySecretHandler(w, req)
	
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}