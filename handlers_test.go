package main

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
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

func TestCreateSecretHandler_ContentTooLong(t *testing.T) {
	// Test with content that exceeds MaxSecretLength characters
	longContent := strings.Repeat("a", MaxSecretLength+1)
	reqBody := CreateSecretRequest{
		Content:       longContent,
		EncryptionKey: generateEncryptionKey(),
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
	// Test with content exactly at the MaxSecretLength character limit
	contentAtLimit := strings.Repeat("a", MaxSecretLength)
	reqBody := CreateSecretRequest{
		Content:       contentAtLimit,
		EncryptionKey: generateEncryptionKey(),
	}
	jsonBody, _ := json.Marshal(reqBody)
	
	req := httptest.NewRequest("POST", "/api/secrets", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	
	createSecretHandler(w, req)
	
	// This should fail because we're not providing properly encrypted data
	// but the length validation should pass
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for improperly encrypted data, got %d", w.Code)
	}
	
	// Make sure it's not failing due to length
	if strings.Contains(w.Body.String(), "exceeds maximum length") {
		t.Errorf("Should not fail due to length at limit, got: %s", w.Body.String())
	}
}

// simulateEncryption properly encrypts data like the frontend would (without manual PKCS7 padding)
func simulateEncryption(plaintext, keyBase64 string) (string, error) {
	// Decode the base64 key
	key, err := base64.StdEncoding.DecodeString(keyBase64)
	if err != nil {
		return "", err
	}
	
	// Create cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	
	// Convert string to bytes
	plainBytes := []byte(plaintext)
	
	// Add PKCS7 padding (simulating what Web Crypto API does automatically)
	blockSize := aes.BlockSize
	padding := blockSize - (len(plainBytes) % blockSize)
	paddedData := make([]byte, len(plainBytes)+padding)
	copy(paddedData, plainBytes)
	for i := len(plainBytes); i < len(paddedData); i++ {
		paddedData[i] = byte(padding)
	}
	
	// Generate random IV
	iv := make([]byte, blockSize)
	if _, err := rand.Read(iv); err != nil {
		return "", err
	}
	
	// Encrypt
	mode := cipher.NewCBCEncrypter(block, iv)
	mode.CryptBlocks(paddedData, paddedData)
	
	// Combine IV and encrypted data
	combined := make([]byte, len(iv)+len(paddedData))
	copy(combined, iv)
	copy(combined[len(iv):], paddedData)
	
	// Return as base64
	return base64.StdEncoding.EncodeToString(combined), nil
}

func TestSmallSecretEncryptionDecryption(t *testing.T) {
	// Test with various small secret sizes to verify padding works correctly
	testCases := []struct {
		name    string
		content string
	}{
		{"1 char", "a"},
		{"5 chars", "hello"},
		{"15 chars", "123456789012345"},
		{"16 chars", "1234567890123456"}, // exactly one block
		{"17 chars", "12345678901234567"}, // one block + 1 char
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			store = NewSecretStore() // Reset store for clean test
			
			encryptionKey := generateEncryptionKey()
			
			// Encrypt the content properly using our encrypt logic
			encryptedContent, err := simulateEncryption(tc.content, encryptionKey)
			if err != nil {
				t.Fatalf("Failed to encrypt test content: %v", err)
			}
			
			// Create secret via API
			reqBody := CreateSecretRequest{
				Content:       encryptedContent,
				EncryptionKey: encryptionKey,
			}
			jsonBody, _ := json.Marshal(reqBody)
			
			req := httptest.NewRequest("POST", "/api/secrets", bytes.NewBuffer(jsonBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			
			createSecretHandler(w, req)
			
			if w.Code != http.StatusOK {
				t.Fatalf("Expected status 200, got %d. Body: %s", w.Code, w.Body.String())
			}
			
			var createResponse CreateSecretResponse
			err = json.Unmarshal(w.Body.Bytes(), &createResponse)
			if err != nil {
				t.Fatalf("Failed to parse create response: %v", err)
			}
			
			// Retrieve the secret via verify endpoint
			verifyReqBody := VerifySecretRequest{VerificationCode: "ABC123"}
			verifyJsonBody, _ := json.Marshal(verifyReqBody)
			
			verifyReq := httptest.NewRequest("POST", "/api/secrets/"+createResponse.ID+"/verify", bytes.NewBuffer(verifyJsonBody))
			verifyReq.Header.Set("Content-Type", "application/json")
			verifyW := httptest.NewRecorder()
			verifyReq = mux.SetURLVars(verifyReq, map[string]string{"id": createResponse.ID})
			
			verifySecretHandler(verifyW, verifyReq)
			
			if verifyW.Code != http.StatusOK {
				t.Fatalf("Expected status 200 for verify, got %d. Body: %s", verifyW.Code, verifyW.Body.String())
			}
			
			var getResponse GetSecretResponse
			err = json.Unmarshal(verifyW.Body.Bytes(), &getResponse)
			if err != nil {
				t.Fatalf("Failed to parse verify response: %v", err)
			}
			
			// Verify the content matches exactly (no padding)
			if getResponse.Content != tc.content {
				t.Errorf("Expected content '%s' (len=%d), got '%s' (len=%d)", 
					tc.content, len(tc.content), getResponse.Content, len(getResponse.Content))
			}
			
			// Check for any padding characters in the response
			for i, ch := range getResponse.Content {
				if ch < 32 && ch != 9 && ch != 10 && ch != 13 { // Allow tab, newline, carriage return
					t.Errorf("Found control character (padding?) at position %d: \\u%04x", i, ch)
				}
			}
			
			// Verify exact length
			if len(getResponse.Content) != len(tc.content) {
				t.Errorf("Expected content length %d, got %d", len(tc.content), len(getResponse.Content))
			}
		})
	}
}