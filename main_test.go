package main

import (
	"testing"
	"time"
)

func TestSecretStore_Store(t *testing.T) {
	store := NewSecretStore()
	
	content := "test secret content"
	id := store.Store(content)
	
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
	id := store.Store(content)
	
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
	id := store.Store(content)
	
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
			id := store.Store(content)
			
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