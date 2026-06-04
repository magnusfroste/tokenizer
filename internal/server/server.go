// Package server wires the HTTP router. It exposes a single New() that
// returns an http.Handler with all middleware and routes registered.
package server

import (
	"log/slog"
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/magnusfroste/tokenizer/internal/auth"
	"github.com/magnusfroste/tokenizer/internal/contextproc"
	"github.com/magnusfroste/tokenizer/internal/engine"
	"github.com/magnusfroste/tokenizer/internal/eventlog"
	"github.com/magnusfroste/tokenizer/internal/health"
	"github.com/magnusfroste/tokenizer/internal/middleware"
	"github.com/magnusfroste/tokenizer/internal/policy"
	"github.com/magnusfroste/tokenizer/internal/provider"
	"github.com/magnusfroste/tokenizer/internal/spend"
)

type Config struct {
	Logger                 *slog.Logger
	KeyStore               auth.KeyStore
	Provider               provider.Adapter // legacy: used when Engine is nil
	ContextPipeline        *contextproc.Pipeline
	ContextPipelineEnabled bool
	Readiness              []ReadyzChecker

	// Routing engine (Sprint 05). Optional — if nil, Provider is used directly.
	Engine      *engine.Engine
	Adapters    map[string]provider.Adapter // provider ID → adapter
	PolicyCache *policy.Cache

	// Observability (Sprint 06). All optional.
	HealthTracker   *health.Tracker
	SpendTracker    *spend.Tracker
	EventQueue      *eventlog.Queue
	RegistryVersion string // shown on dashboard
}

func New(cfg Config) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", HealthzHandler())
	mux.HandleFunc("GET /readyz", ReadyzHandler(cfg.Readiness...))

	// Prometheus metrics.
	mux.Handle("GET /metrics", promhttp.Handler())

	chat := ChatCompletionsHandler(cfg.Provider, ChatOptions{
		ContextPipeline:        cfg.ContextPipeline,
		ContextPipelineEnabled: cfg.ContextPipelineEnabled,
		Logger:                 cfg.Logger,
		Engine:                 cfg.Engine,
		Adapters:               cfg.Adapters,
		PolicyCache:            cfg.PolicyCache,
		HealthTracker:          cfg.HealthTracker,
		EventQueue:             cfg.EventQueue,
	})
	mux.Handle("POST /v1/chat/completions", auth.Middleware(cfg.KeyStore)(chat))

	if cfg.Engine != nil {
		decision := DecisionHandler(DecisionOptions{
			Engine:      cfg.Engine,
			PolicyCache: cfg.PolicyCache,
			Logger:      cfg.Logger,
		})
		mux.Handle("POST /router/decision", auth.Middleware(cfg.KeyStore)(decision))
	}

	// Dashboard (no auth — read-only aggregated stats).
	htmlH, dataH := DashboardHandler(DashboardOptions{
		Spend:   cfg.SpendTracker,
		Health:  cfg.HealthTracker,
		Logger:  cfg.Logger,
		Version: cfg.RegistryVersion,
	})
	mux.HandleFunc("GET /router/dashboard", htmlH)
	mux.HandleFunc("GET /router/dashboard/data", dataH)

	return middleware.RequestID(middleware.Logger(cfg.Logger)(mux))
}
