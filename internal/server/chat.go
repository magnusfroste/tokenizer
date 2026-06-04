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
	"github.com/magnusfroste/tokenizer/internal/engine"
	"github.com/magnusfroste/tokenizer/internal/eventlog"
	"github.com/magnusfroste/tokenizer/internal/health"
	"github.com/magnusfroste/tokenizer/internal/middleware"
	"github.com/magnusfroste/tokenizer/internal/openai"
	"github.com/magnusfroste/tokenizer/internal/policy"
	"github.com/magnusfroste/tokenizer/internal/provider"
	"github.com/magnusfroste/tokenizer/internal/router"
	"github.com/magnusfroste/tokenizer/internal/tenant"
)

// firstTokenTimeoutMS is how long to wait for the first streaming chunk before
// attempting the next fallback provider.
const firstTokenTimeoutMS = 10_000

// ChatOptions configures the chat completions handler.
type ChatOptions struct {
	ContextPipeline        *contextproc.Pipeline
	ContextPipelineEnabled bool
	Logger                 *slog.Logger

	// Routing (Sprint 05). Optional — if Engine is nil the handler uses the
	// single Provider passed to ChatCompletionsHandler.
	Engine      *engine.Engine
	Adapters    map[string]provider.Adapter // provider ID → adapter
	PolicyCache *policy.Cache

	// Observability (Sprint 06). All optional.
	EventQueue    *eventlog.Queue
	HealthTracker *health.Tracker
}

// streamCandidate is one entry in the ordered streaming attempt list.
type streamCandidate struct {
	adapter         provider.StreamingAdapter
	providerModelID string
	modelID         string
	providerID      string
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

		// Build JobDescriptor for every request (fast, pure in-memory).
		var auth router.AuthTenantContext
		if t, ok := tenant.FromContext(r.Context()); ok {
			auth.TenantID = t.ID
			auth.ProjectID = t.Project
		}
		job := router.NewJobDescriptor(router.JobDescriptorInput{
			RequestID: middleware.RequestIDFromContext(r.Context()),
			Auth:      auth,
			Headers:   r.Header,
			Request:   &req,
		})

		// Context pipeline (optional).
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
			result := pipeline.Run(r.Context(), normalized, job)
			if result.TotalTokensSaved > 0 {
				w.Header().Set("X-Router-Context-Savings", strconv.Itoa(result.TotalTokensSaved))
			}
			if len(result.Applied) > 0 || len(result.Skipped) > 0 {
				logContextPipeline(r, cfg.Logger, result)
			}
		}

		// Routing engine path.
		if cfg.Engine != nil && len(cfg.Adapters) > 0 {
			pol := lookupPolicy(cfg.PolicyCache, job)
			routeStart := time.Now()
			health := cfg.healthSnapshot()
			dec, decErr := cfg.Engine.Decide(job, pol, health, req.Stream)
			routingMs := time.Since(routeStart).Milliseconds()

			// Always enqueue a decision event (blocked or not).
			cfg.enqueueDecision(job, dec, routingMs, decErr)

			if decErr != nil {
				if errors.Is(decErr, engine.ErrBlocked) {
					status := dec.BlockStatus
					if status == 0 {
						status = http.StatusForbidden
					}
					writeError(w, status, dec.BlockCode, dec.BlockReason)
					return
				}
				writeError(w, http.StatusBadGateway, "no_route", decErr.Error())
				return
			}

			w.Header().Set("X-Router-Selected-Model", dec.SelectedModel)
			w.Header().Set("X-Router-Policy-Version", dec.PolicyVersion)
			w.Header().Set("X-Router-Route-Class", string(job.TaskType))

			if req.Stream {
				candidates := buildStreamCandidates(dec, cfg.Adapters)
				if len(candidates) > 0 {
					normalized.Model = candidates[0].providerModelID
					streamWithFallback(w, r, candidates, normalized, cfg.Logger, cfg.HealthTracker, cfg.EventQueue, job.RequestID)
					return
				}
			}

			if adapter, ok := cfg.Adapters[dec.SelectedProvider]; ok {
				normalized.Model = dec.ProviderModelID
				start := time.Now()
				resp, err := adapter.Complete(r.Context(), normalized)
				durationMs := time.Since(start).Milliseconds()
				cfg.recordAttempt(job.RequestID, dec.SelectedProvider, dec.SelectedModel, 0, resp, err, durationMs, 0)
				if err != nil {
					status, code := mapProviderError(err)
					writeError(w, status, code, err.Error())
					return
				}
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(resp)
				return
			}
			// Fall through if adapter not found.
		}

		// Legacy path: single provider, no routing.
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

// buildStreamCandidates converts a RouteDecision into an ordered attempt list
// for streaming. Primary is always first.
func buildStreamCandidates(dec engine.RouteDecision, adapters map[string]provider.Adapter) []streamCandidate {
	var result []streamCandidate
	add := func(providerID, providerModelID, modelID string) {
		a, ok := adapters[providerID]
		if !ok {
			return
		}
		sa, ok := a.(provider.StreamingAdapter)
		if !ok {
			return
		}
		result = append(result, streamCandidate{
			adapter:         sa,
			providerModelID: providerModelID,
			modelID:         modelID,
			providerID:      providerID,
		})
	}
	add(dec.SelectedProvider, dec.ProviderModelID, dec.SelectedModel)
	for _, fb := range dec.Fallbacks {
		add(fb.ProviderID, fb.ProviderModelID, fb.ModelID)
	}
	return result
}

// streamWithFallback attempts streaming from each candidate in order.
// It falls back to the next candidate if the connection fails or if no first
// token arrives within firstTokenTimeoutMS (ISSUE-030).
func streamWithFallback(
	w http.ResponseWriter,
	r *http.Request,
	candidates []streamCandidate,
	normalized *provider.NormalizedModelRequest,
	logger *slog.Logger,
	healthTracker *health.Tracker,
	queue *eventlog.Queue,
	requestID string,
) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "streaming_unavailable", "response writer does not support streaming")
		return
	}

	recordStream := func(c streamCandidate, i int, firstTokenMs, durationMs int64, err error) {
		if healthTracker != nil {
			if err == nil {
				healthTracker.RecordSuccess(c.providerID)
			} else {
				healthTracker.RecordFailure(c.providerID)
			}
		}
		if queue != nil {
			code := ""
			if err != nil {
				_, code = mapProviderError(err)
			}
			queue.Enqueue(eventlog.Event{
				Type: eventlog.EventTypeAttempt,
				Attempt: &eventlog.AttemptEvent{
					RequestID:    requestID,
					ProviderID:   c.providerID,
					ModelID:      c.modelID,
					AttemptIndex: i,
					Success:      err == nil,
					ErrorCode:    code,
					DurationMs:   durationMs,
					FirstTokenMs: firstTokenMs,
					AttemptedAt:  time.Now(),
				},
			})
		}
	}

	for i, c := range candidates {
		req := normalized.Clone()
		req.Model = c.providerModelID
		attemptStart := time.Now()

		chunks, err := c.adapter.Stream(r.Context(), req)
		if err != nil {
			logStreamAttempt(r, logger, c, i, "stream_open_error", err)
			recordStream(c, i, 0, time.Since(attemptStart).Milliseconds(), err)
			continue
		}

		// Wait for the first chunk within the first-token timeout.
		timer := time.NewTimer(firstTokenTimeoutMS * time.Millisecond)
		var firstChunk provider.StreamChunk
		var gotFirst bool
		select {
		case chunk, chanOk := <-chunks:
			timer.Stop()
			if !chanOk {
				logStreamAttempt(r, logger, c, i, "stream_channel_closed", nil)
				recordStream(c, i, 0, time.Since(attemptStart).Milliseconds(), fmt.Errorf("channel closed"))
				continue
			}
			firstChunk = chunk
			gotFirst = true
		case <-timer.C:
			logStreamAttempt(r, logger, c, i, "first_token_timeout", nil)
			recordStream(c, i, 0, time.Since(attemptStart).Milliseconds(), provider.ErrProviderTimeout)
			continue
		case <-r.Context().Done():
			timer.Stop()
			return
		}

		if !gotFirst {
			continue
		}

		firstTokenMs := time.Since(attemptStart).Milliseconds()

		// First token received — write headers and stream to completion.
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("X-Router-Selected-Model", c.modelID)
		w.Header().Set("X-Router-First-Token-Sent", "true")
		w.Header().Set("X-Router-First-Token-Ms", strconv.FormatInt(firstTokenMs, 10))
		started := time.Now()

		if firstChunk.Err != nil {
			if i+1 < len(candidates) {
				logStreamAttempt(r, logger, c, i, "first_chunk_error", firstChunk.Err)
				recordStream(c, i, firstTokenMs, time.Since(attemptStart).Milliseconds(), firstChunk.Err)
				continue
			}
			recordStream(c, i, firstTokenMs, time.Since(attemptStart).Milliseconds(), firstChunk.Err)
			status, code := mapProviderError(firstChunk.Err)
			writeError(w, status, code, firstChunk.Err.Error())
			return
		}
		if firstChunk.Done {
			writeSSEData(w, []byte("[DONE]"))
			flusher.Flush()
			recordStream(c, i, firstTokenMs, time.Since(attemptStart).Milliseconds(), nil)
			return
		}
		if len(firstChunk.Data) > 0 {
			writeSSEData(w, firstChunk.Data)
			flusher.Flush()
		}

		// Drain the rest.
		for chunk := range chunks {
			if chunk.Err != nil {
				writeSSEError(w, provider.ErrStreamInterrupted.Error(), chunk.Err.Error())
				flusher.Flush()
				logStreamResult(r, logger, c.providerID, c.modelID, true, time.Since(started).Milliseconds(), chunk.Err)
				recordStream(c, i, firstTokenMs, time.Since(attemptStart).Milliseconds(), chunk.Err)
				return
			}
			if chunk.Done {
				writeSSEData(w, []byte("[DONE]"))
				flusher.Flush()
				logStreamResult(r, logger, c.providerID, c.modelID, true, time.Since(started).Milliseconds(), nil)
				recordStream(c, i, firstTokenMs, time.Since(attemptStart).Milliseconds(), nil)
				return
			}
			if len(chunk.Data) > 0 {
				writeSSEData(w, chunk.Data)
				flusher.Flush()
			}
		}
		logStreamResult(r, logger, c.providerID, c.modelID, true, time.Since(started).Milliseconds(), nil)
		recordStream(c, i, firstTokenMs, time.Since(attemptStart).Milliseconds(), nil)
		return
	}

	// All candidates exhausted.
	writeError(w, http.StatusBadGateway, "provider_error", "all streaming candidates failed or timed out")
}

// streamChatCompletion is the legacy single-provider streaming path.
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
			logStreamResult(r, logger, p.Name(), normalized.Model, true, firstTokenMs, chunk.Err)
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
			logStreamResult(r, logger, p.Name(), normalized.Model, true, firstTokenMs, nil)
			return
		}
		if len(chunk.Data) == 0 {
			continue
		}
		writeSSEData(w, chunk.Data)
		flusher.Flush()
	}
	logStreamResult(r, logger, p.Name(), normalized.Model, firstChunkSent, firstTokenMs, nil)
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

func logStreamAttempt(r *http.Request, logger *slog.Logger, c streamCandidate, attempt int, reason string, err error) {
	if logger == nil {
		logger = slog.Default()
	}
	attrs := []any{
		"request_id", middleware.RequestIDFromContext(r.Context()),
		"attempt", attempt,
		"model", c.modelID,
		"provider", c.providerID,
		"reason", reason,
	}
	if err != nil {
		attrs = append(attrs, "error", err.Error())
	}
	logger.WarnContext(r.Context(), "stream_fallback", attrs...)
}

func logStreamResult(r *http.Request, logger *slog.Logger, providerName, model string, firstChunkSent bool, firstTokenMs int64, err error) {
	if logger == nil {
		logger = slog.Default()
	}
	attrs := []any{
		"request_id", middleware.RequestIDFromContext(r.Context()),
		"provider", providerName,
		"model", model,
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

// --- Observability helpers ---

// healthSnapshot returns the current health snapshot; if no tracker is
// configured returns engine.FullyHealthy.
func (o *ChatOptions) healthSnapshot() engine.HealthSnapshot {
	if o.HealthTracker != nil {
		return o.HealthTracker
	}
	return engine.FullyHealthy
}

// enqueueDecision enqueues a DecisionEvent if a queue is configured.
func (o *ChatOptions) enqueueDecision(job *router.JobDescriptor, dec engine.RouteDecision, routingMs int64, decErr error) {
	if o.EventQueue == nil {
		return
	}
	d := &eventlog.DecisionEvent{
		RequestID:         job.RequestID,
		TenantID:          job.TenantID,
		ProjectID:         job.ProjectID,
		TaskType:          string(job.TaskType),
		RiskLevel:         string(job.RiskLevel),
		Sensitivity:       string(job.Sensitivity),
		SelectedModel:     dec.SelectedModel,
		SelectedProvider:  dec.SelectedProvider,
		PolicyVersion:     dec.PolicyVersion,
		PromptTokens:      job.PromptTokensEstimate,
		EstimatedCostUSD:  dec.EstimatedCostUSD,
		RoutingDurationMs: routingMs,
		Blocked:           dec.Blocked,
		BlockCode:         dec.BlockCode,
		DecidedAt:         time.Now(),
	}
	o.EventQueue.Enqueue(eventlog.Event{Type: eventlog.EventTypeDecision, Decision: d})
}

// recordAttempt updates the health tracker and enqueues an AttemptEvent.
func (o *ChatOptions) recordAttempt(requestID, providerID, modelID string, attemptIdx int, resp *openai.ChatResponse, err error, durationMs, firstTokenMs int64) {
	success := err == nil
	if o.HealthTracker != nil {
		if success {
			o.HealthTracker.RecordSuccess(providerID)
		} else {
			o.HealthTracker.RecordFailure(providerID)
		}
	}
	if o.EventQueue == nil {
		return
	}
	a := &eventlog.AttemptEvent{
		RequestID:    requestID,
		ProviderID:   providerID,
		ModelID:      modelID,
		AttemptIndex: attemptIdx,
		Success:      success,
		DurationMs:   durationMs,
		FirstTokenMs: firstTokenMs,
		AttemptedAt:  time.Now(),
	}
	if err != nil {
		a.ErrorCode = mapProviderErrorCode(err)
	}
	if resp != nil {
		a.InputTokens = resp.Usage.PromptTokens
		a.OutputTokens = resp.Usage.CompletionTokens
	}
	o.EventQueue.Enqueue(eventlog.Event{Type: eventlog.EventTypeAttempt, Attempt: a})
}

func mapProviderErrorCode(err error) string {
	_, code := mapProviderError(err)
	return code
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
