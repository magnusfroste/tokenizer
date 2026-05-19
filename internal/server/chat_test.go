package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/magnusfroste/tokenizer/internal/openai"
	"github.com/magnusfroste/tokenizer/internal/provider"
)

type fakeAdapter struct {
	resp *openai.ChatResponse
	err  error
}

func (f *fakeAdapter) Name() string { return "fake" }
func (f *fakeAdapter) Complete(ctx context.Context, req *openai.ChatRequest) (*openai.ChatResponse, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.resp, nil
}

func postChat(t *testing.T, h http.Handler, body any) *httptest.ResponseRecorder {
	t.Helper()
	buf, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(buf))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestChat_HappyPath(t *testing.T) {
	want := &openai.ChatResponse{
		ID:    "chatcmpl_test",
		Model: "balanced-coder",
		Choices: []openai.Choice{{
			Message: openai.Message{Role: "assistant", Content: "hi"},
		}},
	}
	h := ChatCompletionsHandler(&fakeAdapter{resp: want})

	rec := postChat(t, h, openai.ChatRequest{
		Model:    "auto",
		Messages: []openai.Message{{Role: "user", Content: "hello"}},
	})

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("X-Router-Selected-Model"); got != "balanced-coder" {
		t.Fatalf("expected selected-model header, got %q", got)
	}
	var resp openai.ChatResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("response not parseable: %v", err)
	}
	if resp.ID != "chatcmpl_test" {
		t.Fatalf("response not echoed: %#v", resp)
	}
}

func TestChat_EmptyMessagesRejected(t *testing.T) {
	h := ChatCompletionsHandler(&fakeAdapter{})

	rec := postChat(t, h, openai.ChatRequest{Model: "auto"})

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestChat_StreamingNotSupported(t *testing.T) {
	h := ChatCompletionsHandler(&fakeAdapter{})

	rec := postChat(t, h, openai.ChatRequest{
		Model:    "auto",
		Messages: []openai.Message{{Role: "user", Content: "x"}},
		Stream:   true,
	})

	if rec.Code != http.StatusNotImplemented {
		t.Fatalf("expected 501, got %d", rec.Code)
	}
}

func TestChat_ProviderErrorsMappedToStatus(t *testing.T) {
	cases := []struct {
		name   string
		err    error
		status int
	}{
		{"timeout", provider.ErrProviderTimeout, http.StatusGatewayTimeout},
		{"rate_limit", provider.ErrProviderRateLimit, http.StatusTooManyRequests},
		{"5xx", provider.ErrProvider5xx, http.StatusBadGateway},
		{"bad_response", provider.ErrProviderBadResp, http.StatusBadGateway},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := ChatCompletionsHandler(&fakeAdapter{err: tc.err})
			rec := postChat(t, h, openai.ChatRequest{
				Model:    "auto",
				Messages: []openai.Message{{Role: "user", Content: "x"}},
			})
			if rec.Code != tc.status {
				t.Fatalf("expected %d, got %d body=%s", tc.status, rec.Code, rec.Body.String())
			}
		})
	}
}
