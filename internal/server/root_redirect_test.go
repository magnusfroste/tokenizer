package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRootRedirectsToDashboard(t *testing.T) {
	h := New(Config{})
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusFound {
		t.Fatalf("GET / want 302, got %d", rec.Code)
	}
	if loc := rec.Header().Get("Location"); loc != "/router/dashboard" {
		t.Fatalf("Location = %q, want /router/dashboard", loc)
	}
}
