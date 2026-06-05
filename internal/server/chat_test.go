package server

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/magnusfroste/tokenizer/internal/contextproc"
	"github.com/magnusfroste/tokenizer/internal/middleware"
	"github.com/magnusfroste/tokenizer/internal/openai"
	"github.com/magnusfroste/tokenizer/internal/provider"
	"github.com/magnusfroste/tokenizer/internal/router"
	"github.com/magnusfroste/tokenizer/internal/tenant"
)

type fakeAdapter struct {
	resp          *openai.ChatResponse
	err           error
	completeCalls int
}

func (f *fakeAdapter) Name() string { return "fake" }
func (f *fakeAdapter) Complete(ctx context.Context, req *provider.NormalizedModelRequest) (*openai.ChatResponse, error) {
	f.completeCalls++
	if f.err != nil {
		return nil, f.err
	}
	return f.resp, nil
}

type fakeStreamingAdapter struct {
	fakeAdapter
	streamChunks []provider.StreamChunk
	streamErr    error
	streamCalls  int
}

func (f *fakeStreamingAdapter) Stream(ctx context.Context, req *provider.NormalizedModelRequest) (<-chan provider.StreamChunk, error) {
	f.streamCalls++
	if f.streamErr != nil {
		return nil, f.streamErr
	}
	chunks := make(chan provider.StreamChunk, len(f.streamChunks))
	for _, chunk := range f.streamChunks {
		chunks <- chunk
	}
	close(chunks)
	return chunks, nil
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
	want := testChatResponse()
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

type chatTestProcessor struct {
	result contextproc.Result
	job    *router.JobDescriptor
	called bool
}

func (p *chatTestProcessor) Name() string { return "chat-test" }

func (p *chatTestProcessor) Process(ctx context.Context, req *provider.NormalizedModelRequest, job *router.JobDescriptor) (contextproc.Result, error) {
	p.called = true
	p.job = job
	return p.result, nil
}

func TestChat_ContextPipelineNoopDoesNotWriteSavingsHeader(t *testing.T) {
	processor := &chatTestProcessor{result: contextproc.Result{TokensSaved: 0}}
	h := chatHandlerWithProcessor(processor)

	rec := postChat(t, h, openai.ChatRequest{
		Model:    "auto",
		Messages: []openai.Message{{Role: "user", Content: "hello"}},
	})

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if !processor.called {
		t.Fatal("expected context processor to run")
	}
	if got := rec.Header().Get("X-Router-Context-Savings"); got != "" {
		t.Fatalf("expected no context savings header, got %q", got)
	}
}

func TestChat_ContextPipelineWritesSavingsHeader(t *testing.T) {
	processor := &chatTestProcessor{result: contextproc.Result{TokensSaved: 12}}
	h := chatHandlerWithProcessor(processor)

	rec := postChat(t, h, openai.ChatRequest{
		Model:    "auto",
		Messages: []openai.Message{{Role: "user", Content: "hello"}},
	})

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("X-Router-Context-Savings"); got != "12" {
		t.Fatalf("expected context savings header, got %q", got)
	}
}

func TestChat_ContextPipelineReceivesRouterJobDescriptor(t *testing.T) {
	processor := &chatTestProcessor{result: contextproc.Result{TokensSaved: 0}}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	base := ChatCompletionsHandler(&fakeAdapter{resp: testChatResponse()}, ChatOptions{
		ContextPipelineEnabled: true,
		ContextPipeline: &contextproc.Pipeline{
			Processors: []contextproc.Processor{processor},
			Logger:     logger,
		},
		Logger: logger,
	})
	h := middleware.RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := tenant.WithTenant(r.Context(), &tenant.Tenant{ID: "tn_auth", Project: "prj_auth"})
		base.ServeHTTP(w, r.WithContext(ctx))
	}))

	body := openai.ChatRequest{
		Model: "explicit-model",
		Messages: []openai.Message{
			{Role: "user", Content: "Fix auth payment handling in src/auth/session.ts"},
		},
		Metadata: map[string]any{
			"tenant_id":          "tn_untrusted",
			"project_id":         "prj_untrusted",
			"latency_preference": "fast",
			"risk_level":         "low",
		},
	}
	buf, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(buf))
	req.Header.Set("X-Router-Request-Id", "req_chat_descriptor")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if processor.job == nil {
		t.Fatal("expected processor to receive job descriptor")
	}
	if processor.job.RequestID != "req_chat_descriptor" {
		t.Fatalf("expected request id from middleware, got %q", processor.job.RequestID)
	}
	if processor.job.TenantID != "tn_auth" || processor.job.ProjectID != "prj_auth" {
		t.Fatalf("expected authenticated tenant context, got tenant=%q project=%q", processor.job.TenantID, processor.job.ProjectID)
	}
	if processor.job.TenantIDHint != "" || processor.job.ProjectIDHint != "" {
		t.Fatalf("expected auth tenant to win over untrusted hints, got tenant_hint=%q project_hint=%q", processor.job.TenantIDHint, processor.job.ProjectIDHint)
	}
	if processor.job.RiskLevel != router.RiskHigh || processor.job.RiskLevelHint != router.RiskLow {
		t.Fatalf("expected low risk as hint only, got risk=%q hint=%q", processor.job.RiskLevel, processor.job.RiskLevelHint)
	}
	if processor.job.LatencyPreference != router.LatencyFast {
		t.Fatalf("expected latency hint fast, got %q", processor.job.LatencyPreference)
	}
	if processor.job.ExplicitModel == nil || *processor.job.ExplicitModel != "explicit-model" {
		t.Fatalf("expected explicit model, got %#v", processor.job.ExplicitModel)
	}
}

func testChatResponse() *openai.ChatResponse {
	return &openai.ChatResponse{
		ID:    "chatcmpl_test",
		Model: "balanced-coder",
		Choices: []openai.Choice{{
			Message: openai.Message{Role: "assistant", Content: "hi"},
		}},
	}
}

func chatHandlerWithProcessor(processor contextproc.Processor) http.Handler {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	return ChatCompletionsHandler(&fakeAdapter{resp: testChatResponse()}, ChatOptions{
		ContextPipelineEnabled: true,
		ContextPipeline: &contextproc.Pipeline{
			Processors: []contextproc.Processor{processor},
			Logger:     logger,
		},
		Logger: logger,
	})
}

func TestChat_EmptyMessagesRejected(t *testing.T) {
	h := ChatCompletionsHandler(&fakeAdapter{})

	rec := postChat(t, h, openai.ChatRequest{Model: "auto"})

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestChat_ProviderErrorMasksSecretsBeforeReachingClient(t *testing.T) {
	var logBuf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logBuf, &slog.HandlerOptions{Level: slog.LevelWarn}))

	secret := "postgres://app:topSecretPw99@db.internal:5432/app and Bearer sk-ant-AAAABBBBCCCCDDDDEEEE"
	h := ChatCompletionsHandler(
		&fakeAdapter{err: errors.New("upstream connect failed: " + secret)},
		ChatOptions{Logger: logger},
	)

	rec := postChat(t, h, openai.ChatRequest{
		Model:    "auto",
		Messages: []openai.Message{{Role: "user", Content: "hello"}},
	})

	body := rec.Body.String()
	for _, leaked := range []string{"topSecretPw99", "sk-ant-AAAABBBBCCCCDDDDEEEE"} {
		if strings.Contains(body, leaked) {
			t.Fatalf("secret leaked to client: response body still contains %q\nbody=%s", leaked, body)
		}
	}
	if !strings.Contains(body, "[REDACTED:") {
		t.Fatalf("expected a redaction marker in client error, got body=%s", body)
	}

	// A masking event must be logged, and it must not contain the secret either.
	logs := logBuf.String()
	if !strings.Contains(logs, "secret_masked") || !strings.Contains(logs, "masked_count") {
		t.Fatalf("expected a secret_masked event to be logged, got logs=%s", logs)
	}
	for _, leaked := range []string{"topSecretPw99", "sk-ant-AAAABBBBCCCCDDDDEEEE"} {
		if strings.Contains(logs, leaked) {
			t.Fatalf("secret leaked into masking event log: %q present\nlogs=%s", leaked, logs)
		}
	}
}

func TestChat_StreamSendsSSEChunksInOrder(t *testing.T) {
	adapter := &fakeStreamingAdapter{
		streamChunks: []provider.StreamChunk{
			{Data: []byte(`{"id":"chunk_1","choices":[{"delta":{"content":"hel"}}]}`)},
			{Data: []byte(`{"id":"chunk_1","choices":[{"delta":{"content":"lo"}}]}`)},
			{Done: true},
		},
	}
	h := ChatCompletionsHandler(adapter)

	rec := postChat(t, h, openai.ChatRequest{
		Model:    "auto",
		Messages: []openai.Message{{Role: "user", Content: "x"}},
		Stream:   true,
	})

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("Content-Type"); got != "text/event-stream" {
		t.Fatalf("expected SSE content type, got %q", got)
	}
	if !rec.Flushed {
		t.Fatal("expected streaming handler to flush chunks")
	}
	body := rec.Body.String()
	wantOrder := []string{
		`data: {"id":"chunk_1","choices":[{"delta":{"content":"hel"}}]}`,
		`data: {"id":"chunk_1","choices":[{"delta":{"content":"lo"}}]}`,
		"data: [DONE]",
	}
	last := -1
	for _, want := range wantOrder {
		idx := strings.Index(body, want)
		if idx <= last {
			t.Fatalf("chunk %q not found in order in body %q", want, body)
		}
		last = idx
	}
	if adapter.completeCalls != 0 {
		t.Fatalf("streaming path called Complete %d times", adapter.completeCalls)
	}
	if adapter.streamCalls != 1 {
		t.Fatalf("expected one Stream call, got %d", adapter.streamCalls)
	}
}

func TestChat_StreamFirstTokenHeadersSet(t *testing.T) {
	h := ChatCompletionsHandler(&fakeStreamingAdapter{
		streamChunks: []provider.StreamChunk{{Data: []byte(`{"id":"chunk_1"}`)}},
	})

	rec := postChat(t, h, openai.ChatRequest{
		Model:    "auto",
		Messages: []openai.Message{{Role: "user", Content: "x"}},
		Stream:   true,
	})

	if got := rec.Header().Get("X-Router-First-Token-Sent"); got != "true" {
		t.Fatalf("expected first-token header, got %q", got)
	}
	if got := rec.Header().Get("X-Router-First-Token-Ms"); got == "" {
		t.Fatal("expected first-token timing header")
	} else if _, err := strconv.ParseInt(got, 10, 64); err != nil {
		t.Fatalf("first-token header must be numeric, got %q", got)
	}
}

func TestChat_StreamSetupFailureReturnsJSONError(t *testing.T) {
	h := ChatCompletionsHandler(&fakeStreamingAdapter{
		streamErr: provider.ErrProviderRateLimit,
	})

	rec := postChat(t, h, openai.ChatRequest{
		Model:    "auto",
		Messages: []openai.Message{{Role: "user", Content: "x"}},
		Stream:   true,
	})

	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d body=%s", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("expected JSON content type, got %q", got)
	}
	var envelope openai.ErrorEnvelope
	if err := json.Unmarshal(rec.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("expected JSON error response: %v", err)
	}
	if envelope.Error.Code != "provider_rate_limit" {
		t.Fatalf("unexpected error code: %+v", envelope.Error)
	}
}

func TestChat_StreamInterruptedAfterFirstChunkWritesErrorEvent(t *testing.T) {
	h := ChatCompletionsHandler(&fakeStreamingAdapter{
		streamChunks: []provider.StreamChunk{
			{Data: []byte(`{"id":"chunk_1"}`)},
			{Err: errors.New("provider connection closed")},
		},
	})

	rec := postChat(t, h, openai.ChatRequest{
		Model:    "auto",
		Messages: []openai.Message{{Role: "user", Content: "x"}},
		Stream:   true,
	})

	if rec.Code != http.StatusOK {
		t.Fatalf("expected already-started stream to keep 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("X-Router-First-Token-Sent"); got != "true" {
		t.Fatalf("expected first-token-sent true, got %q", got)
	}
	body := rec.Body.String()
	if !strings.Contains(body, `data: {"id":"chunk_1"}`) {
		t.Fatalf("expected first chunk in body %q", body)
	}
	if !strings.Contains(body, "event: error") || !strings.Contains(body, "stream_interrupted") {
		t.Fatalf("expected SSE error marker in body %q", body)
	}
}

func TestChat_StreamingUnsupportedProviderReturnsProviderBadRequest(t *testing.T) {
	h := ChatCompletionsHandler(&fakeAdapter{})

	rec := postChat(t, h, openai.ChatRequest{
		Model:    "auto",
		Messages: []openai.Message{{Role: "user", Content: "x"}},
		Stream:   true,
	})

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", rec.Code, rec.Body.String())
	}
	var envelope openai.ErrorEnvelope
	if err := json.Unmarshal(rec.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("expected JSON error response: %v", err)
	}
	if envelope.Error.Code != "provider_bad_request" {
		t.Fatalf("unexpected error code: %+v", envelope.Error)
	}
}

func TestChat_NonStreamingStillUsesCompletePath(t *testing.T) {
	adapter := &fakeStreamingAdapter{
		fakeAdapter: fakeAdapter{resp: testChatResponse()},
		streamChunks: []provider.StreamChunk{
			{Data: []byte(`{"id":"should_not_stream"}`)},
		},
	}
	h := ChatCompletionsHandler(adapter)

	rec := postChat(t, h, openai.ChatRequest{
		Model:    "auto",
		Messages: []openai.Message{{Role: "user", Content: "x"}},
	})

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	if adapter.completeCalls != 1 {
		t.Fatalf("expected Complete once, got %d", adapter.completeCalls)
	}
	if adapter.streamCalls != 0 {
		t.Fatalf("non-streaming path called Stream %d times", adapter.streamCalls)
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
