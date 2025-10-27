package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/mux"
)

func TestQRCodeHandler(t *testing.T) {
	store := NewSecretStore()

	// Store a test secret
	secretID, err := store.Store("test secret content", 24*time.Hour)
	if err != nil {
		t.Fatalf("Failed to store secret: %v", err)
	}

	// Create request to get QR code
	req := httptest.NewRequest("GET", "/api/secrets/"+secretID+"/qr", nil)

	// Set up the router with the handler
	router := mux.NewRouter()
	router.HandleFunc("/api/secrets/{id}/qr", qrCodeHandler)

	// Create response recorder
	rr := httptest.NewRecorder()

	// Serve the request
	router.ServeHTTP(rr, req)

	// Check the status code
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	// Check the content type
	contentType := rr.Header().Get("Content-Type")
	if contentType != "image/png" {
		t.Errorf("handler returned wrong content type: got %v want %v", contentType, "image/png")
	}

	// Check that the response body is not empty (QR code image)
	if rr.Body.Len() == 0 {
		t.Error("handler returned empty body")
	}

	// Check that the response starts with PNG header
	pngHeader := []byte{0x89, 0x50, 0x4E, 0x47}
	body := rr.Body.Bytes()
	if len(body) < 4 {
		t.Error("response body too short to contain PNG header")
	} else {
		for i := 0; i < 4; i++ {
			if body[i] != pngHeader[i] {
				t.Error("response does not contain valid PNG header")
				break
			}
		}
	}
}

func TestQRCodeHandler_InvalidID(t *testing.T) {
	// Test with a non-existent secret ID
	req := httptest.NewRequest("GET", "/api/secrets/nonexistent/qr", nil)

	router := mux.NewRouter()
	router.HandleFunc("/api/secrets/{id}/qr", qrCodeHandler)

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	// Should still generate QR code (doesn't validate if secret exists)
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}
}
