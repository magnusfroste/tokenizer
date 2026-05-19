package server

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

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
		if req.Stream {
			writeError(w, http.StatusNotImplemented, "not_implemented", "streaming is not supported in sprint 1")
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
	case errors.Is(err, provider.ErrProvider5xx):
		return http.StatusBadGateway, "provider_5xx"
	case errors.Is(err, provider.ErrProviderBadResp):
		return http.StatusBadGateway, "provider_bad_response"
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
