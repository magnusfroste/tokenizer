// Package server wires the HTTP router. It exposes a single New() that
// returns an http.Handler with all middleware and routes registered.
package server

import (
	"log/slog"
	"net/http"

	"github.com/magnusfroste/tokenizer/internal/auth"
	"github.com/magnusfroste/tokenizer/internal/middleware"
	"github.com/magnusfroste/tokenizer/internal/provider"
)

type Config struct {
	Logger    *slog.Logger
	KeyStore  auth.KeyStore
	Provider  provider.Adapter
	Readiness []ReadyzChecker
}

func New(cfg Config) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", HealthzHandler())
	mux.HandleFunc("GET /readyz", ReadyzHandler(cfg.Readiness...))

	chat := ChatCompletionsHandler(cfg.Provider)
	mux.Handle("POST /v1/chat/completions", auth.Middleware(cfg.KeyStore)(chat))

	return middleware.RequestID(middleware.Logger(cfg.Logger)(mux))
}
