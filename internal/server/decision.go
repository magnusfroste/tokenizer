package server

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/magnusfroste/tokenizer/internal/engine"
	"github.com/magnusfroste/tokenizer/internal/middleware"
	"github.com/magnusfroste/tokenizer/internal/openai"
	"github.com/magnusfroste/tokenizer/internal/policy"
	"github.com/magnusfroste/tokenizer/internal/router"
	"github.com/magnusfroste/tokenizer/internal/tenant"
)

// DecisionOptions configures the /router/decision handler.
type DecisionOptions struct {
	Engine      *engine.Engine
	PolicyCache *policy.Cache
	Logger      *slog.Logger
}

// decisionResponse extends RouteDecision with the computed JobDescriptor so
// callers can see exactly what the engine classified from the request.
type decisionResponse struct {
	engine.RouteDecision
	Job *router.JobDescriptor `json:"job,omitempty"`
}

// DecisionHandler handles POST /router/decision — a dry-run that returns the
// routing decision without making any provider call.
func DecisionHandler(opts DecisionOptions) http.HandlerFunc {
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

		if opts.Engine == nil {
			writeError(w, http.StatusServiceUnavailable, "engine_unavailable", "routing engine not configured")
			return
		}

		pol := lookupPolicy(opts.PolicyCache, job)
		dec, err := opts.Engine.Decide(job, pol, engine.FullyHealthy, req.Stream)
		if err != nil {
			if errors.Is(err, engine.ErrBlocked) {
				status := dec.BlockStatus
				if status == 0 {
					status = http.StatusForbidden
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(status)
				_ = json.NewEncoder(w).Encode(decisionResponse{RouteDecision: dec})
				return
			}
			writeError(w, http.StatusUnprocessableEntity, "no_route", err.Error())
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Router-Selected-Model", dec.SelectedModel)
		if dec.PolicyVersion != "" {
			w.Header().Set("X-Router-Policy-Version", dec.PolicyVersion)
		}

		resp := decisionResponse{RouteDecision: dec}
		if policy.ExplainEnabled(r.Header) {
			resp.Job = job
		}

		_ = json.NewEncoder(w).Encode(resp)

		if l := opts.Logger; l != nil {
			l.InfoContext(r.Context(), "router_decision",
				"request_id", middleware.RequestIDFromContext(r.Context()),
				"selected_model", dec.SelectedModel,
				"selected_provider", dec.SelectedProvider,
				"policy_version", dec.PolicyVersion,
				"timeout_ms", dec.TimeoutMS,
				"fallback_count", len(dec.Fallbacks),
			)
		}
	}
}

func lookupPolicy(cache *policy.Cache, job *router.JobDescriptor) *policy.CompiledPolicy {
	if cache == nil {
		return nil
	}
	scope := policy.Scope{TenantID: job.TenantID, ProjectID: job.ProjectID}
	pol, _ := cache.Active(scope)
	return pol
}
