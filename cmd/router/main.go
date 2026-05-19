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

	"github.com/magnusfroste/tokenix/internal/auth"
	"github.com/magnusfroste/tokenix/internal/provider"
	"github.com/magnusfroste/tokenix/internal/server"
	"github.com/magnusfroste/tokenix/internal/tenant"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
		Level: parseLogLevel(os.Getenv("LOG_LEVEL")),
	}))

	keyStore := auth.NewInMemoryKeyStore()
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

	handler := server.New(server.Config{
		Logger:   logger,
		KeyStore: keyStore,
		Provider: mock,
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
