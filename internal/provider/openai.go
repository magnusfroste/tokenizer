package provider

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/magnusfroste/tokenizer/internal/openai"
)

const chatCompletionsPath = "/v1/chat/completions"

// OpenAIAdapter calls an OpenAI-compatible chat completions endpoint. It
// expects req.Model to already contain the provider model id selected upstream.
type OpenAIAdapter struct {
	BaseURL string
	APIKey  string
	Client  *http.Client
	Timeout time.Duration
}

func (a *OpenAIAdapter) Name() string { return "openai" }

func (a *OpenAIAdapter) Complete(ctx context.Context, req *NormalizedModelRequest) (*openai.ChatResponse, error) {
	if a.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, a.Timeout)
		defer cancel()
	}

	endpoint, err := chatCompletionsURL(a.BaseURL)
	if err != nil {
		return nil, err
	}
	body, err := json.Marshal(req.ToOpenAI())
	if err != nil {
		return nil, fmt.Errorf("openai adapter: marshal request: %w", err)
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("openai adapter: build request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if a.APIKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+a.APIKey)
	}

	client := a.Client
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(httpReq)
	if err != nil {
		if isTimeoutError(err) || errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return nil, fmt.Errorf("%w: request timed out", ErrProviderTimeout)
		}
		return nil, fmt.Errorf("openai adapter: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, mapOpenAIStatus(resp.StatusCode, resp.Body)
	}

	var out openai.ChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("%w: decode response", ErrProviderBadResp)
	}
	return &out, nil
}

func (a *OpenAIAdapter) Stream(ctx context.Context, req *NormalizedModelRequest) (<-chan StreamChunk, error) {
	var cancel context.CancelFunc
	if a.Timeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, a.Timeout)
	}

	endpoint, err := chatCompletionsURL(a.BaseURL)
	if err != nil {
		if cancel != nil {
			cancel()
		}
		return nil, err
	}
	outbound := req.Clone()
	outbound.Stream = true
	body, err := json.Marshal(outbound.ToOpenAI())
	if err != nil {
		if cancel != nil {
			cancel()
		}
		return nil, fmt.Errorf("openai adapter: marshal stream request: %w", err)
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		if cancel != nil {
			cancel()
		}
		return nil, fmt.Errorf("openai adapter: build stream request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "text/event-stream")
	if a.APIKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+a.APIKey)
	}

	client := a.Client
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(httpReq)
	if err != nil {
		if cancel != nil {
			cancel()
		}
		if isTimeoutError(err) || errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return nil, fmt.Errorf("%w: stream request timed out", ErrProviderTimeout)
		}
		return nil, fmt.Errorf("openai adapter: stream request failed: %w", err)
	}
	if resp.StatusCode >= 400 {
		defer resp.Body.Close()
		if cancel != nil {
			cancel()
		}
		return nil, mapOpenAIStatus(resp.StatusCode, resp.Body)
	}

	chunks := make(chan StreamChunk)
	go func() {
		defer close(chunks)
		defer resp.Body.Close()
		if cancel != nil {
			defer cancel()
		}

		reader := bufio.NewReader(resp.Body)
		doneSeen := false
		for {
			line, err := reader.ReadString('\n')
			if len(line) > 0 {
				if chunk, ok := parseSSEDataLine(line); ok {
					if chunk.Done {
						doneSeen = true
					}
					chunks <- chunk
					if chunk.Done {
						return
					}
				}
			}
			if err == nil {
				continue
			}
			if errors.Is(err, io.EOF) {
				if !doneSeen {
					chunks <- StreamChunk{Err: fmt.Errorf("%w: stream ended before done frame", ErrStreamInterrupted)}
				}
				return
			}
			chunks <- StreamChunk{Err: fmt.Errorf("%w: read stream: %v", ErrStreamInterrupted, err)}
			return
		}
	}()
	return chunks, nil
}

func parseSSEDataLine(line string) (StreamChunk, bool) {
	line = strings.TrimSpace(line)
	if !strings.HasPrefix(line, "data:") {
		return StreamChunk{}, false
	}
	payload := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
	if payload == "[DONE]" {
		return StreamChunk{Done: true}, true
	}
	if payload == "" {
		return StreamChunk{}, false
	}
	return StreamChunk{Data: []byte(payload)}, true
}

func chatCompletionsURL(baseURL string) (string, error) {
	if strings.TrimSpace(baseURL) == "" {
		return "", fmt.Errorf("%w: missing base url", ErrProviderBadReq)
	}
	parsed, err := url.Parse(baseURL)
	if err != nil {
		return "", fmt.Errorf("%w: invalid base url", ErrProviderBadReq)
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return "", fmt.Errorf("%w: invalid base url", ErrProviderBadReq)
	}
	basePath := strings.TrimRight(parsed.Path, "/")
	switch {
	case strings.HasSuffix(basePath, chatCompletionsPath):
		parsed.Path = basePath
	case strings.HasSuffix(basePath, "/v1"):
		parsed.Path = basePath + "/chat/completions"
	default:
		parsed.Path = basePath + chatCompletionsPath
	}
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return parsed.String(), nil
}

func mapOpenAIStatus(status int, body io.Reader) error {
	envelope := decodeErrorEnvelope(body)
	if isModelUnavailable(envelope) || status == http.StatusNotFound {
		return fmt.Errorf("%w: provider status %d", ErrModelUnavailable, status)
	}

	switch {
	case status == http.StatusUnauthorized || status == http.StatusForbidden:
		return fmt.Errorf("%w: provider status %d", ErrProviderAuth, status)
	case status == http.StatusTooManyRequests:
		return fmt.Errorf("%w: provider status %d", ErrProviderRateLimit, status)
	case status >= 500:
		return fmt.Errorf("%w: provider status %d", ErrProvider5xx, status)
	case status >= 400:
		return fmt.Errorf("%w: provider status %d", ErrProviderBadReq, status)
	default:
		return fmt.Errorf("%w: provider status %d", ErrProviderBadResp, status)
	}
}

func decodeErrorEnvelope(body io.Reader) openai.ErrorEnvelope {
	var envelope openai.ErrorEnvelope
	limited := io.LimitReader(body, 64*1024)
	_ = json.NewDecoder(limited).Decode(&envelope)
	return envelope
}

func isModelUnavailable(envelope openai.ErrorEnvelope) bool {
	errBody := envelope.Error
	haystack := strings.ToLower(strings.Join([]string{
		errBody.Code,
		errBody.Type,
		errBody.Message,
	}, " "))
	return strings.Contains(haystack, "model_not_found") ||
		strings.Contains(haystack, "model_not_available") ||
		strings.Contains(haystack, "model unavailable") ||
		strings.Contains(haystack, "does not exist") ||
		strings.Contains(haystack, "not found")
}

func isTimeoutError(err error) bool {
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	var netErr net.Error
	return errors.As(err, &netErr) && netErr.Timeout()
}
