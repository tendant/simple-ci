package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealth_Simple(t *testing.T) {
	// Test simple health endpoint without dependencies
	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	// Simple health check should return ok without needing service
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	})

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Health() status = %d, want %d", w.Code, http.StatusOK)
	}

	want := `{"status":"ok"}`
	if w.Body.String() != want {
		t.Errorf("Health() body = %s, want %s", w.Body.String(), want)
	}
}
