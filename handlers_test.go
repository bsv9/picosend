package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/mux"
)

// Test create secret handler with encrypted content (client-side encryption model)
func TestCreateSecretHandler(t *testing.T) {
	store = NewSecretStore() // Reset store for clean test

	// Create a mock encrypted content (base64 encoded)
	// With client-side encryption, the server just stores this as-is
	encryptedContent := base64.StdEncoding.EncodeToString([]byte("mock encrypted content"))

	reqBody := CreateSecretRequest{
		Content:  encryptedContent,
		Lifetime: 60,
	}
	jsonBody, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/api/secrets", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	createSecretHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d. Body: %s", w.Code, w.Body.String())
	}

	var response CreateSecretResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if response.ID == "" {
		t.Error("Expected non-empty secret ID")
	}
}

func TestCreateSecretHandler_EmptyContent(t *testing.T) {
	reqBody := CreateSecretRequest{
		Content:  "",
		Lifetime: 60,
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
	secretContent := base64.StdEncoding.EncodeToString([]byte("encrypted test content"))
	secretID, err := store.Store(secretContent, 24*time.Hour)
	if err != nil {
		t.Fatalf("Failed to store secret: %v", err)
	}

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
	err = json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Errorf("Failed to parse response: %v", err)
	}

	// Content should be returned as-is (encrypted)
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
	secretContent := base64.StdEncoding.EncodeToString([]byte("encrypted test content"))
	secretID, err := store.Store(secretContent, 24*time.Hour)
	if err != nil {
		t.Fatalf("Failed to store secret: %v", err)
	}

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
	secretContent := base64.StdEncoding.EncodeToString([]byte("encrypted test content"))
	secretID, err := store.Store(secretContent, 24*time.Hour)
	if err != nil {
		t.Fatalf("Failed to store secret: %v", err)
	}

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
	err = json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Errorf("Failed to parse response: %v", err)
	}

	// Content should be returned as-is (encrypted)
	if response.Content != secretContent {
		t.Errorf("Expected content '%s', got '%s'", secretContent, response.Content)
	}
}

func TestVerifySecretHandler_InvalidCode(t *testing.T) {
	store = NewSecretStore() // Reset store for clean test
	secretID, err := store.Store(base64.StdEncoding.EncodeToString([]byte("test content")), 24*time.Hour)
	if err != nil {
		t.Fatalf("Failed to store secret: %v", err)
	}

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
	secretID, err := store.Store(base64.StdEncoding.EncodeToString([]byte("test content")), 24*time.Hour)
	if err != nil {
		t.Fatalf("Failed to store secret: %v", err)
	}

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
	secretID, err := store.Store(base64.StdEncoding.EncodeToString([]byte("test content")), 24*time.Hour)
	if err != nil {
		t.Fatalf("Failed to store secret: %v", err)
	}

	req := httptest.NewRequest("POST", "/api/secrets/"+secretID+"/verify", strings.NewReader("invalid json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	req = mux.SetURLVars(req, map[string]string{"id": secretID})

	verifySecretHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestCreateSecretHandler_ContentTooLong(t *testing.T) {
	store = NewSecretStore() // Reset store for clean test

	// Test with content that exceeds MaxSecretLength*2 characters (for base64 encoding)
	longContent := base64.StdEncoding.EncodeToString([]byte(strings.Repeat("a", MaxSecretLength*2+1)))
	reqBody := CreateSecretRequest{
		Content:  longContent,
		Lifetime: 60,
	}
	jsonBody, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/api/secrets", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	createSecretHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for content too long, got %d", w.Code)
	}

	if !strings.Contains(w.Body.String(), "exceeds maximum length") {
		t.Errorf("Expected error message about length limit, got: %s", w.Body.String())
	}
}

func TestCreateSecretHandler_ContentAtLimit(t *testing.T) {
	store = NewSecretStore() // Reset store for clean test

	// Test with content exactly at the MaxSecretLength*2 character limit
	// Note: The limit is on the encoded (base64) content length, not the original content
	// So we create a string that, when base64 encoded, equals exactly MaxSecretLength*2
	// Base64 encoding adds ~33% overhead, so we need raw content of about MaxSecretLength*2/1.33
	rawLen := (MaxSecretLength * 2 * 3) / 4 // Account for base64 encoding overhead
	contentAtLimit := base64.StdEncoding.EncodeToString([]byte(strings.Repeat("a", rawLen)))

	// Make sure the encoded content is at or under the limit
	if len(contentAtLimit) > MaxSecretLength*2 {
		t.Fatalf("Test error: encoded content length %d exceeds limit %d", len(contentAtLimit), MaxSecretLength*2)
	}

	reqBody := CreateSecretRequest{
		Content:  contentAtLimit,
		Lifetime: 60,
	}
	jsonBody, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/api/secrets", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	createSecretHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200 for content at limit, got %d. Body: %s", w.Code, w.Body.String())
	}
}

func TestCreateSecretHandler_MaxSecretsLimit(t *testing.T) {
	store = NewSecretStore() // Reset store for clean test

	// Create a simple encrypted content
	encryptedContent := base64.StdEncoding.EncodeToString([]byte("test content"))

	// Fill up to the limit
	for i := 0; i < MaxUnreadSecrets; i++ {
		reqBody := CreateSecretRequest{
			Content:  encryptedContent,
			Lifetime: 60,
		}
		jsonBody, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("POST", "/api/secrets", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		createSecretHandler(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("Expected status 200 for secret %d, got %d. Body: %s", i, w.Code, w.Body.String())
		}
	}

	// Try to create one more - should fail with 429
	reqBody := CreateSecretRequest{
		Content:  encryptedContent,
		Lifetime: 60,
	}
	jsonBody, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/api/secrets", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	createSecretHandler(w, req)

	if w.Code != http.StatusTooManyRequests {
		t.Errorf("Expected status 429, got %d. Body: %s", w.Code, w.Body.String())
	}

	expectedError := fmt.Sprintf("maximum number of unread secrets (%d) reached", MaxUnreadSecrets)
	if !strings.Contains(w.Body.String(), expectedError) {
		t.Errorf("Expected error message to contain '%s', got '%s'", expectedError, w.Body.String())
	}
}
