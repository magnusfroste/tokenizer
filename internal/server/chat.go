package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/magnusfroste/tokenizer/internal/contextproc"
	"github.com/magnusfroste/tokenizer/internal/middleware"
	"github.com/magnusfroste/tokenizer/internal/openai"
	"github.com/magnusfroste/tokenizer/internal/provider"
	"github.com/magnusfroste/tokenizer/internal/router"
	"github.com/magnusfroste/tokenizer/internal/tenant"
)

type ChatOptions struct {
	ContextPipeline        *contextproc.Pipeline
	ContextPipelineEnabled bool
	Logger                 *slog.Logger
}

func ChatCompletionsHandler(p provider.Adapter, opts ...ChatOptions) http.HandlerFunc {
	var cfg ChatOptions
	if len(opts) > 0 {
		cfg = opts[0]
	}
	return func(w http.ResponseWriter, r *http.Request) {
		var req openai.ChatRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_request_error", err.Error())
			return
		}
		if len(req.Messages) == 0 {
			writeError(w, http.StatusBadRequest, "invalid_request_error", "messages cannot be empty")
			return
		}
		normalized := provider.NormalizeChatRequest(&req)
		if cfg.ContextPipelineEnabled {
			pipeline := cfg.ContextPipeline
			if pipeline == nil {
				pipeline = contextproc.NewNoopPipeline()
			}
			if pipeline.Logger == nil && cfg.Logger != nil {
				pipelineCopy := *pipeline
				pipelineCopy.Logger = cfg.Logger
				pipeline = &pipelineCopy
			}
			result := pipeline.Run(r.Context(), normalized, jobDescriptorForRequest(r, &req))
			if result.TotalTokensSaved > 0 {
				w.Header().Set("X-Router-Context-Savings", strconv.Itoa(result.TotalTokensSaved))
			}
			if len(result.Applied) > 0 || len(result.Skipped) > 0 {
				logContextPipeline(r, cfg.Logger, result)
			}
			// TODO: Re-estimate prompt_tokens_estimate after real processors mutate context.
		}

		if req.Stream {
			streamChatCompletion(w, r, p, normalized, cfg.Logger)
			return
		}

		resp, err := p.Complete(r.Context(), normalized)
		if err != nil {
			status, code := mapProviderError(err)
			writeError(w, status, code, err.Error())
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Router-Selected-Model", resp.Model)
		_ = json.NewEncoder(w).Encode(resp)
	}
}

func streamChatCompletion(w http.ResponseWriter, r *http.Request, p provider.Adapter, normalized *provider.NormalizedModelRequest, logger *slog.Logger) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "streaming_unavailable", "response writer does not support streaming")
		return
	}
	streamer, ok := p.(provider.StreamingAdapter)
	if !ok {
		err := fmt.Errorf("%w: streaming not supported by provider %s", provider.ErrProviderBadReq, p.Name())
		status, code := mapProviderError(err)
		writeError(w, status, code, err.Error())
		return
	}

	started := time.Now()
	chunks, err := streamer.Stream(r.Context(), normalized)
	if err != nil {
		status, code := mapProviderError(err)
		writeError(w, status, code, err.Error())
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	if normalized.Model != "" {
		w.Header().Set("X-Router-Selected-Model", normalized.Model)
	}

	firstChunkSent := false
	var firstTokenMs int64
	for chunk := range chunks {
		if chunk.Err != nil {
			if !firstChunkSent {
				status, code := mapProviderError(chunk.Err)
				writeError(w, status, code, chunk.Err.Error())
				return
			}
			writeSSEError(w, provider.ErrStreamInterrupted.Error(), chunk.Err.Error())
			flusher.Flush()
			logStreamResult(r, logger, p, normalized, true, firstTokenMs, chunk.Err)
			return
		}

		if !firstChunkSent {
			firstChunkSent = true
			firstTokenMs = time.Since(started).Milliseconds()
			w.Header().Set("X-Router-First-Token-Sent", "true")
			w.Header().Set("X-Router-First-Token-Ms", strconv.FormatInt(firstTokenMs, 10))
		}
		if chunk.Done {
			writeSSEData(w, []byte("[DONE]"))
			flusher.Flush()
			logStreamResult(r, logger, p, normalized, true, firstTokenMs, nil)
			return
		}
		if len(chunk.Data) == 0 {
			continue
		}
		writeSSEData(w, chunk.Data)
		flusher.Flush()
	}

	logStreamResult(r, logger, p, normalized, firstChunkSent, firstTokenMs, nil)
}

func writeSSEData(w http.ResponseWriter, data []byte) {
	_, _ = w.Write([]byte("data: "))
	_, _ = w.Write(data)
	_, _ = w.Write([]byte("\n\n"))
}

func writeSSEError(w http.ResponseWriter, code, msg string) {
	body, _ := json.Marshal(openai.ErrorEnvelope{
		Error: openai.ErrorBody{Message: msg, Type: code, Code: code},
	})
	_, _ = w.Write([]byte("event: error\n"))
	writeSSEData(w, body)
}

func logStreamResult(r *http.Request, logger *slog.Logger, p provider.Adapter, normalized *provider.NormalizedModelRequest, firstChunkSent bool, firstTokenMs int64, err error) {
	if logger == nil {
		logger = slog.Default()
	}
	attrs := []any{
		"request_id", middleware.RequestIDFromContext(r.Context()),
		"provider", p.Name(),
		"model", normalized.Model,
		"first_token_sent", firstChunkSent,
		"first_token_ms", firstTokenMs,
	}
	if err != nil {
		attrs = append(attrs, "stream_error", err.Error())
		logger.WarnContext(r.Context(), "chat_stream_interrupted", attrs...)
		return
	}
	logger.InfoContext(r.Context(), "chat_stream_completed", attrs...)
}

func jobDescriptorForRequest(r *http.Request, req *openai.ChatRequest) *router.JobDescriptor {
	job := &router.JobDescriptor{
		RequestID:       middleware.RequestIDFromContext(r.Context()),
		TaskType:        "unknown",
		RiskLevel:       "unknown",
		RouterMode:      req.Model,
		Metadata:        req.Metadata,
		RequiresToolUse: len(req.Tools) > 0,
	}
	if t, ok := tenant.FromContext(r.Context()); ok {
		job.TenantID = t.ID
		job.ProjectID = t.Project
	}
	if req.Model != "" && req.Model != "auto" {
		model := req.Model
		job.ExplicitModel = &model
	}
	return job
}

func logContextPipeline(r *http.Request, logger *slog.Logger, result contextproc.PipelineResult) {
	if logger == nil {
		logger = slog.Default()
	}
	logger.InfoContext(r.Context(), "context_pipeline",
		"request_id", middleware.RequestIDFromContext(r.Context()),
		"context_processors_applied", result.Applied,
		"context_processors_skipped", result.Skipped,
		"context_tokens_saved", result.TotalTokensSaved,
	)
}

func mapProviderError(err error) (status int, code string) {
	switch {
	case errors.Is(err, provider.ErrProviderTimeout):
		return http.StatusGatewayTimeout, "provider_timeout"
	case errors.Is(err, provider.ErrProviderRateLimit):
		return http.StatusTooManyRequests, "provider_rate_limit"
	case errors.Is(err, provider.ErrProviderAuth):
		return http.StatusBadGateway, "provider_auth_error"
	case errors.Is(err, provider.ErrProvider5xx):
		return http.StatusBadGateway, "provider_5xx"
	case errors.Is(err, provider.ErrProviderBadReq):
		return http.StatusBadRequest, "provider_bad_request"
	case errors.Is(err, provider.ErrProviderBadResp):
		return http.StatusBadGateway, "provider_bad_response"
	case errors.Is(err, provider.ErrModelUnavailable):
		return http.StatusBadGateway, "model_unavailable"
	case errors.Is(err, provider.ErrStreamInterrupted):
		return http.StatusBadGateway, "stream_interrupted"
	default:
		return http.StatusBadGateway, "provider_error"
	}
}

func writeError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(openai.ErrorEnvelope{
		Error: openai.ErrorBody{Message: msg, Type: code, Code: code},
	})
}
