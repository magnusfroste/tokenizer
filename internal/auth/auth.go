// Package auth implements the API-key bearer authentication used by the
// proxy. Keys are stored as their SHA-256 hash; the plaintext is never
// retained server-side once the store has been populated.
package auth

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strings"
	"sync"

	"github.com/magnusfroste/tokenix/internal/openai"
	"github.com/magnusfroste/tokenix/internal/tenant"
)

type KeyStore interface {
	Lookup(hashedKey string) (*tenant.Tenant, bool)
}

type InMemoryKeyStore struct {
	mu   sync.RWMutex
	keys map[string]*tenant.Tenant
}

func NewInMemoryKeyStore() *InMemoryKeyStore {
	return &InMemoryKeyStore{keys: make(map[string]*tenant.Tenant)}
}

// Add stores the SHA-256 of the plaintext key. The plaintext is discarded.
func (s *InMemoryKeyStore) Add(plaintext string, t *tenant.Tenant) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.keys[hashKey(plaintext)] = t
}

func (s *InMemoryKeyStore) Lookup(hashed string) (*tenant.Tenant, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	t, ok := s.keys[hashed]
	return t, ok
}

func hashKey(plaintext string) string {
	sum := sha256.Sum256([]byte(plaintext))
	return hex.EncodeToString(sum[:])
}

// Middleware enforces a Bearer token. On success it attaches the tenant to
// the request context. On failure it returns 401 with an OpenAI-style error
// envelope so SDKs can parse the response uniformly.
func Middleware(store KeyStore) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h := r.Header.Get("Authorization")
			if !strings.HasPrefix(h, "Bearer ") {
				writeUnauthorized(w, "missing bearer token")
				return
			}
			plaintext := strings.TrimSpace(strings.TrimPrefix(h, "Bearer "))
			if plaintext == "" {
				writeUnauthorized(w, "empty bearer token")
				return
			}
			t, ok := store.Lookup(hashKey(plaintext))
			if !ok {
				writeUnauthorized(w, "invalid api key")
				return
			}
			next.ServeHTTP(w, r.WithContext(tenant.WithTenant(r.Context(), t)))
		})
	}
}

func writeUnauthorized(w http.ResponseWriter, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	_ = json.NewEncoder(w).Encode(openai.ErrorEnvelope{
		Error: openai.ErrorBody{Message: msg, Type: "unauthorized"},
	})
}
