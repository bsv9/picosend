package main

import (
	"fmt"
	"testing"
	"time"
)

func TestSecretStore_Store(t *testing.T) {
	store := NewSecretStore()
	
	content := "test secret content"
	id, err := store.Store(content)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	
	if id == "" {
		t.Error("Expected non-empty ID")
	}
	
	if len(id) != 16 {
		t.Errorf("Expected ID length of 16, got %d", len(id))
	}
}

func TestSecretStore_Get(t *testing.T) {
	store := NewSecretStore()
	
	content := "test secret content"
	id, err := store.Store(content)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	
	// First retrieval should succeed
	secret, found := store.Get(id)
	if !found {
		t.Error("Expected to find the secret")
	}
	
	if secret.Content != content {
		t.Errorf("Expected content '%s', got '%s'", content, secret.Content)
	}
	
	if secret.ID != id {
		t.Errorf("Expected ID '%s', got '%s'", id, secret.ID)
	}
	
	// Verify timestamp is recent
	if time.Since(secret.CreatedAt) > time.Minute {
		t.Error("Expected recent creation time")
	}
}

func TestSecretStore_GetOnlyOnce(t *testing.T) {
	store := NewSecretStore()
	
	content := "test secret content"
	id, err := store.Store(content)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	
	// First retrieval should succeed
	_, found := store.Get(id)
	if !found {
		t.Error("Expected to find the secret on first retrieval")
	}
	
	// Second retrieval should fail (secret should be deleted)
	_, found = store.Get(id)
	if found {
		t.Error("Expected secret to be deleted after first retrieval")
	}
}

func TestSecretStore_GetNonExistent(t *testing.T) {
	store := NewSecretStore()
	
	// Try to get a secret that doesn't exist
	_, found := store.Get("nonexistent")
	if found {
		t.Error("Expected not to find non-existent secret")
	}
}

func TestGenerateID(t *testing.T) {
	id1 := generateID()
	id2 := generateID()
	
	if id1 == id2 {
		t.Error("Expected different IDs on subsequent calls")
	}
	
	if len(id1) != 16 {
		t.Errorf("Expected ID length of 16, got %d", len(id1))
	}
	
	if len(id2) != 16 {
		t.Errorf("Expected ID length of 16, got %d", len(id2))
	}
	
	// Check that ID contains only base64url characters (A-Z, a-z, 0-9, -, _)
	for _, char := range id1 {
		if !((char >= '0' && char <= '9') || (char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') || char == '-' || char == '_') {
			t.Errorf("Expected base64url character, got '%c'", char)
		}
	}
}

func TestSecretStore_Concurrent(t *testing.T) {
	store := NewSecretStore()
	
	// Test concurrent access
	done := make(chan bool, 10)
	
	for i := 0; i < 10; i++ {
		go func(i int) {
			content := "test secret content"
			id, err := store.Store(content)
			if err != nil {
				t.Errorf("Goroutine %d: Expected no error, got %v", i, err)
				done <- true
				return
			}
			
			secret, found := store.Get(id)
			if !found {
				t.Errorf("Goroutine %d: Expected to find the secret", i)
			}
			
			if secret.Content != content {
				t.Errorf("Goroutine %d: Expected content '%s', got '%s'", i, content, secret.Content)
			}
			
			done <- true
		}(i)
	}
	
	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestSecretStore_MaxLimit(t *testing.T) {
	store := NewSecretStore()
	
	// Store secrets up to the limit
	content := "test secret"
	for i := 0; i < MaxUnreadSecrets; i++ {
		_, err := store.Store(content)
		if err != nil {
			t.Fatalf("Expected no error storing secret %d, got %v", i, err)
		}
	}
	
	// Verify we have reached the limit
	if store.Count() != MaxUnreadSecrets {
		t.Errorf("Expected %d secrets, got %d", MaxUnreadSecrets, store.Count())
	}
	
	// Try to store one more - should fail
	_, err := store.Store(content)
	if err == nil {
		t.Error("Expected error when exceeding max secrets limit")
	}
	
	expectedError := fmt.Sprintf("maximum number of unread secrets (%d) reached", MaxUnreadSecrets)
	if err.Error() != expectedError {
		t.Errorf("Expected error message '%s', got '%s'", expectedError, err.Error())
	}
}

func TestSecretStore_MemoryCleanup(t *testing.T) {
	store := NewSecretStore()
	
	// Store a secret
	content := "test secret"
	id, err := store.Store(content)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	
	// Verify it's in memory
	if store.Count() != 1 {
		t.Errorf("Expected 1 secret in memory, got %d", store.Count())
	}
	
	// Retrieve the secret
	_, found := store.Get(id)
	if !found {
		t.Error("Expected to find the secret")
	}
	
	// Verify memory is cleaned up
	if store.Count() != 0 {
		t.Errorf("Expected 0 secrets in memory after retrieval, got %d", store.Count())
	}
}

func TestSecretStore_LimitAfterCleanup(t *testing.T) {
	store := NewSecretStore()
	
	content := "test secret"
	
	// Fill up to the limit
	ids := make([]string, MaxUnreadSecrets)
	for i := 0; i < MaxUnreadSecrets; i++ {
		id, err := store.Store(content)
		if err != nil {
			t.Fatalf("Expected no error storing secret %d, got %v", i, err)
		}
		ids[i] = id
	}
	
	// Should be at limit
	_, err := store.Store(content)
	if err == nil {
		t.Error("Expected error when at limit")
	}
	
	// Read and delete half the secrets
	for i := 0; i < MaxUnreadSecrets/2; i++ {
		_, found := store.Get(ids[i])
		if !found {
			t.Errorf("Expected to find secret %d", i)
		}
	}
	
	// Should now be able to store new secrets
	for i := 0; i < MaxUnreadSecrets/2; i++ {
		_, err := store.Store(content)
		if err != nil {
			t.Errorf("Expected no error after cleanup, got %v", err)
		}
	}
	
	// Should be at limit again
	_, err = store.Store(content)
	if err == nil {
		t.Error("Expected error when back at limit")
	}
}