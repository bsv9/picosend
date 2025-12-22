package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/mux"
)

// Test that encrypted content is stored as-is without decryption on server
func TestCreateSecretHandlerWithEncryptedContent(t *testing.T) {
	// Reset store for clean test
	store = NewSecretStore()

	// Create a mock encrypted content (base64 encoded with IV prepended)
	// This simulates what the frontend would send
	testContent := base64.StdEncoding.EncodeToString([]byte("mock encrypted data with iv prefix"))

	reqBody := CreateSecretRequest{
		Content:  testContent,
		Lifetime: 60,
	}
	jsonBody, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/api/secrets", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	createSecretHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp CreateSecretResponse
	err := json.NewDecoder(w.Body).Decode(&resp)
	if err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp.ID == "" {
		t.Error("Expected non-empty secret ID")
	}

	// Verify the encrypted content is stored as-is
	secret, found := store.Get(resp.ID)
	if !found {
		t.Fatal("Secret not found in store")
	}

	if secret.Content != testContent {
		t.Error("Encrypted content was modified by server")
	}
}

// Test that server accepts encrypted content without encryption key
func TestCreateSecretHandlerNoEncryptionKeyRequired(t *testing.T) {
	// Reset store for clean test
	store = NewSecretStore()

	testContent := base64.StdEncoding.EncodeToString([]byte("mock encrypted content"))

	reqBody := CreateSecretRequest{
		Content:  testContent,
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
}

// Test encrypted content length validation
func TestEncryptedContentLengthValidation(t *testing.T) {
	// Reset store for clean test
	store = NewSecretStore()

	// Create content larger than allowed (MaxSecretLength * 2 for base64)
	largeContent := strings.Repeat("a", MaxSecretLength*2+1)
	encodedContent := base64.StdEncoding.EncodeToString([]byte(largeContent))

	reqBody := CreateSecretRequest{
		Content:  encodedContent,
		Lifetime: 60,
	}
	jsonBody, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/api/secrets", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	createSecretHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for oversized content, got %d", w.Code)
	}
}

// Test that encrypted content returned by verify endpoint
func TestVerifySecretHandlerReturnsEncryptedContent(t *testing.T) {
	// Reset store for clean test
	store = NewSecretStore()

	// Create a secret with encrypted content via the API
	testContent := base64.StdEncoding.EncodeToString([]byte("encrypted test content"))

	createReq := CreateSecretRequest{
		Content:  testContent,
		Lifetime: 60,
	}
	jsonBody, _ := json.Marshal(createReq)

	createReqHTTP := httptest.NewRequest("POST", "/api/secrets", bytes.NewBuffer(jsonBody))
	createReqHTTP.Header.Set("Content-Type", "application/json")
	createW := httptest.NewRecorder()

	createSecretHandler(createW, createReqHTTP)

	if createW.Code != http.StatusOK {
		t.Fatalf("Failed to create secret, got status %d", createW.Code)
	}

	var createResp CreateSecretResponse
	json.NewDecoder(createW.Body).Decode(&createResp)

	// Now verify the secret
	verifyReqBody := VerifySecretRequest{
		VerificationCode: "ABC123",
	}
	verifyJsonBody, _ := json.Marshal(verifyReqBody)

	req := httptest.NewRequest("POST", "/api/secrets/"+createResp.ID+"/verify", bytes.NewBuffer(verifyJsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Setup mux vars
	req = mux.SetURLVars(req, map[string]string{"id": createResp.ID})

	verifySecretHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp GetSecretResponse
	err := json.NewDecoder(w.Body).Decode(&resp)
	if err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// The content should be encrypted (returned as-is from storage)
	if resp.Content != testContent {
		t.Errorf("Expected encrypted content %s, got %s", testContent, resp.Content)
	}
}

// Test empty content validation
func TestCreateSecretHandlerEmptyContent(t *testing.T) {
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

// Test default lifetime when not specified
func TestCreateSecretHandlerDefaultLifetime(t *testing.T) {
	// Reset store for clean test
	store = NewSecretStore()

	testContent := base64.StdEncoding.EncodeToString([]byte("test content"))

	reqBody := CreateSecretRequest{
		Content:  testContent,
		Lifetime: 0, // Should default to 24 hours
	}
	jsonBody, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/api/secrets", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	createSecretHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp CreateSecretResponse
	json.NewDecoder(w.Body).Decode(&resp)

	// Verify the secret has the correct expiration (approximately 24 hours)
	secret, _ := store.Get(resp.ID)
	expectedExpiry := secret.CreatedAt.Add(24 * time.Hour)
	timeDiff := secret.ExpiresAt.Sub(expectedExpiry)

	// Allow 1 second tolerance for test execution time
	if timeDiff < -time.Second || timeDiff > time.Second {
		t.Errorf("Expected ~24 hour lifetime, got diff of %v", timeDiff)
	}
}
