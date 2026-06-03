// Package server wires the HTTP router. It exposes a single New() that
// returns an http.Handler with all middleware and routes registered.
package server

import (
	"log/slog"
	"net/http"

	"github.com/magnusfroste/tokenizer/internal/auth"
	"github.com/magnusfroste/tokenizer/internal/contextproc"
	"github.com/magnusfroste/tokenizer/internal/engine"
	"github.com/magnusfroste/tokenizer/internal/middleware"
	"github.com/magnusfroste/tokenizer/internal/policy"
	"github.com/magnusfroste/tokenizer/internal/provider"
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
}

func New(cfg Config) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", HealthzHandler())
	mux.HandleFunc("GET /readyz", ReadyzHandler(cfg.Readiness...))

	chat := ChatCompletionsHandler(cfg.Provider, ChatOptions{
		ContextPipeline:        cfg.ContextPipeline,
		ContextPipelineEnabled: cfg.ContextPipelineEnabled,
		Logger:                 cfg.Logger,
		Engine:                 cfg.Engine,
		Adapters:               cfg.Adapters,
		PolicyCache:            cfg.PolicyCache,
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

	return middleware.RequestID(middleware.Logger(cfg.Logger)(mux))
}
