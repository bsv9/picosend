package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGenerateEncryptionKey(t *testing.T) {
	key1 := generateEncryptionKey()
	key2 := generateEncryptionKey()
	
	if key1 == key2 {
		t.Error("Expected different encryption keys on subsequent calls")
	}
	
	// Keys should be base64 encoded 32-byte keys (44 characters in base64)
	if len(key1) != 44 {
		t.Errorf("Expected key length of 44 characters, got %d", len(key1))
	}
	
	if len(key2) != 44 {
		t.Errorf("Expected key length of 44 characters, got %d", len(key2))
	}
}

func TestEncryptionKeyHandler(t *testing.T) {
	store = NewSecretStore() // Reset store for clean test
	
	// Create test server
	server := setupTestServer()
	defer server.Close()
	
	// Test encryption key endpoint
	resp, err := http.Get(server.URL + "/api/encryption-key")
	if err != nil {
		t.Fatalf("Failed to get encryption key: %v", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
	
	var keyResp EncryptionKeyResponse
	err = json.NewDecoder(resp.Body).Decode(&keyResp)
	if err != nil {
		t.Fatalf("Failed to decode key response: %v", err)
	}
	
	if keyResp.Key == "" {
		t.Error("Expected non-empty encryption key")
	}
	
	if len(keyResp.Key) != 44 {
		t.Errorf("Expected key length of 44 characters, got %d", len(keyResp.Key))
	}
}

func TestDecryptFunction(t *testing.T) {
	// Test with a known encrypted value
	key := "dGVzdGtleWZvcmVuY3J5cHRpb250ZXN0MzIwYnl0ZXM="  // base64 encoded 32-byte key

	// Test with invalid base64 key
	_, err := decrypt("validbase64", "invalidbase64!")
	if err == nil {
		t.Error("Expected error with invalid base64 key")
	}

	// Test with invalid base64 data
	_, err = decrypt("invalidbase64!", key)
	if err == nil {
		t.Error("Expected error with invalid base64 data")
	}

	// Test with too short ciphertext
	shortData := "dGVzdA==" // base64 for "test" - too short for AES block
	_, err = decrypt(shortData, key)
	if err == nil {
		t.Error("Expected error with too short ciphertext")
	}
}

func TestDecryptFunction_AdditionalErrors(t *testing.T) {
	// Valid 32-byte key
	validKey := "dGVzdGtleWZvcmVuY3J5cHRpb250ZXN0MzIwYnl0ZXM="

	// Test with invalid key size (not 16, 24, or 32 bytes)
	invalidKey := base64.StdEncoding.EncodeToString([]byte("shortkey"))
	_, err := decrypt("AAAAAAAAAAAAAAAAAAAAAA==", invalidKey)
	if err == nil {
		t.Error("Expected error with invalid key size")
	}

	// Test with ciphertext not multiple of block size
	// Create 17 bytes (16 + 1, not a multiple of 16)
	invalidCiphertext := base64.StdEncoding.EncodeToString(make([]byte, 17))
	_, err = decrypt(invalidCiphertext, validKey)
	if err == nil {
		t.Error("Expected error with ciphertext not multiple of block size")
	}
	if err != nil && !strings.Contains(err.Error(), "not a multiple of the block size") {
		t.Errorf("Expected 'not a multiple of block size' error, got: %v", err)
	}

	// Test with only IV, no ciphertext after
	// Create exactly 16 bytes (just the IV)
	onlyIV := base64.StdEncoding.EncodeToString(make([]byte, 16))
	_, err = decrypt(onlyIV, validKey)
	if err == nil {
		t.Error("Expected error with no ciphertext after IV")
	}
	if err != nil && !strings.Contains(err.Error(), "no ciphertext after IV") {
		t.Errorf("Expected 'no ciphertext after IV' error, got: %v", err)
	}

	// Test with invalid padding - padding byte is 0
	// Create 32 bytes (16 IV + 16 data) with last byte as 0 (invalid padding)
	invalidPadding := make([]byte, 32)
	invalidPadding[31] = 0 // Invalid padding value
	_, err = decrypt(base64.StdEncoding.EncodeToString(invalidPadding), validKey)
	if err == nil {
		t.Error("Expected error with invalid padding value of 0")
	}

	// Test with invalid padding - padding byte > 16
	invalidPadding2 := make([]byte, 32)
	invalidPadding2[31] = 17 // Padding > block size
	_, err = decrypt(base64.StdEncoding.EncodeToString(invalidPadding2), validKey)
	if err == nil {
		t.Error("Expected error with padding value > block size")
	}

	// Test with invalid PKCS7 padding - padding bytes don't match
	// Create 48 bytes (16 IV + 32 data) with invalid PKCS7 padding
	invalidPKCS7 := make([]byte, 48)
	invalidPKCS7[47] = 3  // Says padding is 3 bytes
	invalidPKCS7[46] = 3  // Match
	invalidPKCS7[45] = 2  // Doesn't match - should be 3
	_, err = decrypt(base64.StdEncoding.EncodeToString(invalidPKCS7), validKey)
	if err == nil {
		t.Error("Expected error with invalid PKCS7 padding")
	}
	if err != nil && !strings.Contains(err.Error(), "PKCS7 padding") && !strings.Contains(err.Error(), "invalid padding") {
		t.Errorf("Expected padding error, got: %v", err)
	}
}

func TestCreateSecretHandlerWithMockEncryption(t *testing.T) {
	// Reset store for clean test
	store = NewSecretStore()
	
	// Create a test encryption key
	encryptionKey := generateEncryptionKey()
	
	// Create a mock encrypted content that will fail decryption (as expected)
	testContent := "VGhpcyBpcyBhIHRlc3Qgc2VjcmV0IG1lc3NhZ2U=" // base64 encoded test
	
	// Test with mock encrypted content
	reqBody := CreateSecretRequest{
		Content:       testContent,
		EncryptionKey: encryptionKey,
	}
	jsonBody, _ := json.Marshal(reqBody)
	
	req := httptest.NewRequest("POST", "/api/secrets", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	
	// This should fail because we're using mock encrypted data that can't be properly decrypted
	createSecretHandler(w, req)
	
	// We expect this to fail with mock data
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for mock encrypted data, got %d", w.Code)
	}
}

func TestCreateSecretHandlerMissingEncryptionKey(t *testing.T) {
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
	
	if !strings.Contains(w.Body.String(), "Encryption key cannot be empty") {
		t.Error("Expected error message about empty encryption key")
	}
}

func TestEncryptionKeyValidation(t *testing.T) {
	// This test validates the encryption key format
	key := generateEncryptionKey()
	
	// Test that the key generation works
	if len(key) != 44 {
		t.Errorf("Expected key length of 44, got %d", len(key))
	}
	
	// Test that we can validate the key format
	keyBytes, err := base64.StdEncoding.DecodeString(key)
	if err != nil {
		t.Errorf("Generated key should be valid base64: %v", err)
	}
	
	if len(keyBytes) != 32 {
		t.Errorf("Decoded key should be 32 bytes, got %d", len(keyBytes))
	}
}