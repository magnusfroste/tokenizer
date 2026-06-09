package server_test

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/magnusfroste/tokenizer/internal/auth"
	"github.com/magnusfroste/tokenizer/internal/openai"
	"github.com/magnusfroste/tokenizer/internal/provider"
	"github.com/magnusfroste/tokenizer/internal/server"
	"github.com/magnusfroste/tokenizer/internal/tenant"
)

// TestChatEndpointEnforcesScope verifies the chat route, wired through
// server.New, rejects a key that lacks the chat:completions scope (ISSUE-046).
func TestChatEndpointEnforcesScope(t *testing.T) {
	store := auth.NewInMemoryKeyStore()
	// Key is limited to router:decision — no chat scope.
	store.Add("scoped-key", &tenant.Tenant{
		ID: "tn", Project: "prj", KeyID: "key", Scopes: []string{auth.ScopeRouterDecision},
	})

	h := server.New(server.Config{
		Logger:   slog.New(slog.NewTextHandler(io.Discard, nil)),
		KeyStore: store,
		Provider: &provider.MockAdapter{},
	})
	srv := httptest.NewServer(h)
	defer srv.Close()

	body, _ := json.Marshal(openai.ChatRequest{
		Model:    "auto",
		Messages: []openai.Message{{Role: "user", Content: "ping"}},
	})
	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/v1/chat/completions", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer scoped-key")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", resp.StatusCode)
	}
	var env openai.ErrorEnvelope
	_ = json.NewDecoder(resp.Body).Decode(&env)
	if env.Error.Code != "insufficient_scope" {
		t.Errorf("error code = %q, want insufficient_scope", env.Error.Code)
	}
}
