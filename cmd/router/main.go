package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/magnusfroste/tokenizer/internal/audit"
	"github.com/magnusfroste/tokenizer/internal/auth"
	"github.com/magnusfroste/tokenizer/internal/budget"
	"github.com/magnusfroste/tokenizer/internal/decisioncache"
	"github.com/magnusfroste/tokenizer/internal/engine"
	"github.com/magnusfroste/tokenizer/internal/eventlog"
	"github.com/magnusfroste/tokenizer/internal/health"
	"github.com/magnusfroste/tokenizer/internal/outcomes"
	"github.com/magnusfroste/tokenizer/internal/policy"
	"github.com/magnusfroste/tokenizer/internal/provider"
	"github.com/magnusfroste/tokenizer/internal/registry"
	"github.com/magnusfroste/tokenizer/internal/retention"
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
			Scopes:  auth.AllScopes(),
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

	// Provider selection (ISSUE: OpenRouter integration). When OPENROUTER_API_KEY
	// is set, route through OpenRouter (a real OpenAI-compatible provider) via the
	// existing OpenAIAdapter; otherwise fall back to the in-process mock so the
	// build/dev path needs no credentials.
	var (
		snap            *registry.Snapshot
		primaryProvider provider.Adapter
		adapters        map[string]provider.Adapter
		snapErr         error
	)
	if orKey := strings.TrimSpace(os.Getenv("OPENROUTER_API_KEY")); orKey != "" {
		snap, snapErr = registry.OpenRouterSnapshot()
		orAdapter := &provider.OpenAIAdapter{
			BaseURL: "https://openrouter.ai/api/v1",
			APIKey:  orKey,
			// OpenRouter attribution (optional but recommended for app ranking).
			Referer: envDefault("OPENROUTER_REFERER", "https://github.com/magnusfroste/tokenizer"),
			Title:   envDefault("OPENROUTER_TITLE", "tokenizer"),
			Client:  &http.Client{Timeout: 60 * time.Second},
			Timeout: 60 * time.Second,
		}
		primaryProvider = orAdapter
		adapters = map[string]provider.Adapter{"openrouter": orAdapter}
		logger.Info("using OpenRouter provider", "base_url", "https://openrouter.ai/api/v1")
	} else {
		snap, snapErr = registry.DefaultSnapshot()
		primaryProvider = mock
		// In local dev the mock adapter serves all providers.
		adapters = map[string]provider.Adapter{
			"openai":    mock,
			"anthropic": mock,
		}
		logger.Info("using mock provider", "mock_provider", mockURL)
	}
	if snapErr != nil {
		logger.Error("failed to build registry", "err", snapErr)
		os.Exit(1)
	}
	store, err := registry.NewStore(snap)
	if err != nil {
		logger.Error("failed to create registry store", "err", err)
		os.Exit(1)
	}
	policyCache, err := loadRuntimePolicyCache(snap, os.Getenv("ROUTER_POLICY_PATH"))
	if err != nil {
		logger.Error("failed to build policy cache", "err", err)
		os.Exit(1)
	}
	shadowPolicyCache, err := loadOptionalRuntimePolicyCache(snap, os.Getenv("ROUTER_SHADOW_POLICY_PATH"))
	if err != nil {
		logger.Error("failed to build shadow policy cache", "err", err)
		os.Exit(1)
	}
	eng := engine.New(store)
	// Global conservative mode (ISSUE-060): incident lever that routes uncertain
	// classifications at a raised minimum tier. See 07-operations/runbook.md.
	if parseBoolEnv(os.Getenv("ROUTER_CONSERVATIVE_MODE")) {
		eng.SetConservative(true)
		logger.Info("global conservative mode enabled")
	}

	// Observability: health tracker, spend tracker, event queue.
	healthTracker := health.New()
	spendTracker := spend.New()
	// Spend/savings persistence (ISSUE-067): when ROUTER_DATA_DIR is set, the
	// spend aggregates are loaded at startup and periodically flushed to a file
	// on that volume so the dashboard's savings survive restarts/redeploys.
	spendFile := ""
	if dir := strings.TrimSpace(os.Getenv("ROUTER_DATA_DIR")); dir != "" {
		spendFile = filepath.Join(dir, "spend.json")
		if snap, err := spend.LoadJSON(spendFile); err != nil {
			logger.Warn("could not load persisted spend", "err", err, "file", spendFile)
		} else {
			spendTracker.Restore(snap)
			logger.Info("loaded persisted spend", "file", spendFile)
		}
	}
	outcomeStore := outcomes.NewStore()
	eventQueue := eventlog.NewQueue(0)
	comparisonTracker := eventlog.NewComparisonTracker(0)

	// Budget caps (ISSUE-051): a ledger accrues spend from the event queue and an
	// evaluator checks it on the request path. Caps are opt-in; ROUTER_BUDGET_USD
	// sets a default per-tenant cap for local dev.
	budgetCaps := budget.NewCaps()
	budgetLedger := budget.NewLedger()
	if usd := parseIntEnv(os.Getenv("ROUTER_BUDGET_USD"), 0); usd > 0 {
		budgetCaps.SetTenant("tn_local", budget.Cap{
			LimitMicroUSD: int64(usd) * 1_000_000,
			Action:        budget.ActionDowngrade,
		})
	}
	budgetEvaluator := budget.NewEvaluator(budgetCaps, budgetLedger)

	// Route decision cache (ISSUE-052): low-risk decisions reused within a short
	// TTL. Disabled when ROUTER_DECISION_CACHE_TTL_SECONDS is 0.
	decisionCache := decisioncache.New(
		time.Duration(parseIntEnv(os.Getenv("ROUTER_DECISION_CACHE_TTL_SECONDS"), 60))*time.Second,
		0,
	)

	// Build the fan-out event handler: logging + metrics + spend + budget ledger
	// + shadow comparison tracking.
	combinedHandler := buildEventHandler(logger, spendTracker, budgetLedger, comparisonTracker)

	// Start the queue worker in the background.
	workerCtx, workerCancel := context.WithCancel(context.Background())
	go eventQueue.Run(workerCtx, combinedHandler, logger)

	// Periodically flush spend aggregates so savings survive a crash/redeploy.
	if spendFile != "" {
		go func() {
			ticker := time.NewTicker(30 * time.Second)
			defer ticker.Stop()
			for {
				select {
				case <-ticker.C:
					if err := spendTracker.SaveJSON(spendFile); err != nil {
						logger.Warn("spend flush failed", "err", err)
					}
				case <-workerCtx.Done():
					return
				}
			}
		}()
	}

	// Retention/privacy settings (ISSUE-045). Prompt logging is off unless
	// ROUTER_PROMPT_LOGGING is explicitly enabled.
	retentionSettings := retention.NewSettings(
		parseIntEnv(os.Getenv("ROUTER_RETENTION_DAYS"), retention.DefaultRetentionDays),
		parseBoolEnv(os.Getenv("ROUTER_PROMPT_LOGGING")),
	)

	// Premium-tier pricing for the dashboard "saved vs all-premium" baseline.
	premInMicros, premOutMicros := premiumPricing(snap)

	handler := server.New(server.Config{
		Logger:                     logger,
		KeyStore:                   keyStore,
		Provider:                   primaryProvider,
		ContextPipelineEnabled:     parseBoolEnv(os.Getenv("ROUTER_CONTEXT_PIPELINE_ENABLED")),
		PromptAdapter:              buildPromptAdapter(parseBoolEnv(os.Getenv("ROUTER_PROMPT_ADAPTER_ENABLED"))),
		Engine:                     eng,
		Adapters:                   adapters,
		PolicyCache:                policyCache,
		ShadowPolicyCache:          shadowPolicyCache,
		HealthTracker:              healthTracker,
		SpendTracker:               spendTracker,
		ComparisonTracker:          comparisonTracker,
		EventQueue:                 eventQueue,
		RegistryVersion:            snap.RegistryVersion(),
		OutcomeStore:               outcomeStore,
		Auditor:                    auditSink,
		Retention:                  retentionSettings,
		Budget:                     budgetEvaluator,
		DecisionCache:              decisionCache,
		PremiumInputMicrosPerMTok:  premInMicros,
		PremiumOutputMicrosPerMTok: premOutMicros,
		DashboardPassword:          strings.TrimSpace(os.Getenv("ROUTER_DASHBOARD_PASSWORD")),
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
	if spendFile != "" {
		if err := spendTracker.SaveJSON(spendFile); err != nil {
			logger.Error("final spend flush failed", "err", err)
		} else {
			logger.Info("persisted spend on shutdown", "file", spendFile)
		}
	}
}

func loadRuntimePolicyCache(snapshot *registry.Snapshot, path string) (*policy.Cache, error) {
	return policy.NewRuntimeCache(snapshot, strings.TrimSpace(path))
}

func loadOptionalRuntimePolicyCache(snapshot *registry.Snapshot, path string) (*policy.Cache, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, nil
	}
	return policy.NewRuntimeCache(snapshot, path)
}

func buildPromptAdapter(enabled bool) *provider.PromptAdapter {
	if !enabled {
		return nil
	}
	return &provider.PromptAdapter{
		Enabled: true,
		ModelProfiles: map[string]string{
			"cheap-general":     "cheap",
			"balanced-coder":    "balanced",
			"premium-reasoning": "premium",
		},
		Rules: []provider.PromptAdapterRule{
			{
				Name: "cheap-system-cost-aware",
				Match: provider.PromptAdapterMatch{
					Profiles: []string{"cheap"},
				},
				Mutation: provider.SystemPromptMutation{
					Suffix: "\n\nPrefer concise answers and avoid unnecessary reasoning traces.",
				},
			},
			{
				Name: "premium-system-depth",
				Match: provider.PromptAdapterMatch{
					Profiles: []string{"premium"},
				},
				Mutation: provider.SystemPromptMutation{
					Prefix: "Use careful, high-assurance reasoning for this task.\n\n",
				},
			},
		},
	}
}

func buildEventHandler(logger *slog.Logger, spendTracker *spend.Tracker, budgetLedger *budget.Ledger, comparisonTracker *eventlog.ComparisonTracker) eventlog.Handler {
	handlers := []eventlog.Handler{
		&eventlog.LoggingHandler{Logger: logger},
		spendTracker,
		budgetLedger,
	}
	if comparisonTracker != nil {
		handlers = append(handlers, comparisonTracker)
	}
	return eventlog.MultiHandler(handlers...)
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

func envDefault(name, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(name)); v != "" {
		return v
	}
	return fallback
}

// premiumPricing returns the premium-tier model's input/output price (micros per
// million tokens) for the dashboard's all-premium savings baseline. Returns 0,0
// when no premium model exists in the snapshot.
func premiumPricing(snap *registry.Snapshot) (inMicros, outMicros int64) {
	for _, m := range snap.EnabledModelsWithCapabilities(registry.Capabilities{Chat: true}) {
		if m.Tier == registry.TierPremium {
			return m.Cost.InputMicrosPerMillionToken, m.Cost.OutputMicrosPerMillionToken
		}
	}
	return 0, 0
}

func parseIntEnv(s string, fallback int) int {
	if v, err := strconv.Atoi(strings.TrimSpace(s)); err == nil && v > 0 {
		return v
	}
	return fallback
}
