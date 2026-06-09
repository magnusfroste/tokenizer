package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/magnusfroste/tokenizer/internal/audit"
	"github.com/magnusfroste/tokenizer/internal/auth"
	"github.com/magnusfroste/tokenizer/internal/engine"
	"github.com/magnusfroste/tokenizer/internal/eventlog"
	"github.com/magnusfroste/tokenizer/internal/health"
	"github.com/magnusfroste/tokenizer/internal/outcomes"
	"github.com/magnusfroste/tokenizer/internal/provider"
	"github.com/magnusfroste/tokenizer/internal/registry"
	"github.com/magnusfroste/tokenizer/internal/server"
	"github.com/magnusfroste/tokenizer/internal/spend"
	"github.com/magnusfroste/tokenizer/internal/tenant"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
		Level: parseLogLevel(os.Getenv("LOG_LEVEL")),
	}))

	// Security audit trail (ISSUE-044): structured logs plus an in-memory ring
	// buffer for in-process retrieval. Wired into the key store before any keys
	// are added so seed mutations are captured too.
	auditSink := audit.MultiSink(&audit.LogSink{Logger: logger}, audit.NewMemorySink(0))

	keyStore := auth.NewInMemoryKeyStore()
	keyStore.SetAuditor(auditSink)
	if k := strings.TrimSpace(os.Getenv("LOCAL_API_KEY")); k != "" {
		keyStore.Add(k, &tenant.Tenant{
			ID:      "tn_local",
			Project: "prj_local",
			KeyID:   "key_local",
		})
		logger.Info("seeded local api key", "tenant", "tn_local")
	}

	mockURL := os.Getenv("MOCK_PROVIDER_URL")
	if mockURL == "" {
		mockURL = "http://localhost:18080"
	}

	mock := &provider.MockAdapter{
		BaseURL: mockURL,
		Client:  &http.Client{Timeout: 30 * time.Second},
	}

	snap, err := registry.DefaultSnapshot()
	if err != nil {
		logger.Error("failed to build registry", "err", err)
		os.Exit(1)
	}
	store, err := registry.NewStore(snap)
	if err != nil {
		logger.Error("failed to create registry store", "err", err)
		os.Exit(1)
	}
	eng := engine.New(store)

	// In local dev the mock adapter serves all providers.
	adapters := map[string]provider.Adapter{
		"openai":    mock,
		"anthropic": mock,
	}

	// Observability: health tracker, spend tracker, event queue.
	healthTracker := health.New()
	spendTracker := spend.New()
	outcomeStore := outcomes.NewStore()
	eventQueue := eventlog.NewQueue(0)

	// Build the fan-out event handler: logging + metrics + spend.
	loggingHandler := &eventlog.LoggingHandler{Logger: logger}
	combinedHandler := eventlog.MultiHandler(loggingHandler, spendTracker)

	// Start the queue worker in the background.
	workerCtx, workerCancel := context.WithCancel(context.Background())
	go eventQueue.Run(workerCtx, combinedHandler, logger)

	handler := server.New(server.Config{
		Logger:                 logger,
		KeyStore:               keyStore,
		Provider:               mock,
		ContextPipelineEnabled: parseBoolEnv(os.Getenv("ROUTER_CONTEXT_PIPELINE_ENABLED")),
		Engine:                 eng,
		Adapters:               adapters,
		HealthTracker:          healthTracker,
		SpendTracker:           spendTracker,
		EventQueue:             eventQueue,
		RegistryVersion:        snap.RegistryVersion(),
		OutcomeStore:           outcomeStore,
		Auditor:                auditSink,
	})

	addr := os.Getenv("ROUTER_ADDR")
	if addr == "" {
		addr = ":8080"
	}
	srv := &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		logger.Info("router starting", "addr", addr, "mock_provider", mockURL)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("server failed", "err", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	logger.Info("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("shutdown error", "err", err)
	}
	workerCancel() // drain event queue gracefully
}

func parseLogLevel(s string) slog.Level {
	switch strings.ToLower(s) {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

func parseBoolEnv(s string) bool {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}
