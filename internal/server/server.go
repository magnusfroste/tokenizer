// Package server wires the HTTP router. It exposes a single New() that
// returns an http.Handler with all middleware and routes registered.
package server

import (
	"log/slog"
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/magnusfroste/tokenizer/internal/audit"
	"github.com/magnusfroste/tokenizer/internal/auth"
	"github.com/magnusfroste/tokenizer/internal/budget"
	"github.com/magnusfroste/tokenizer/internal/contextproc"
	"github.com/magnusfroste/tokenizer/internal/decisioncache"
	"github.com/magnusfroste/tokenizer/internal/engine"
	"github.com/magnusfroste/tokenizer/internal/eventlog"
	"github.com/magnusfroste/tokenizer/internal/health"
	"github.com/magnusfroste/tokenizer/internal/middleware"
	"github.com/magnusfroste/tokenizer/internal/outcomes"
	"github.com/magnusfroste/tokenizer/internal/policy"
	"github.com/magnusfroste/tokenizer/internal/provider"
	"github.com/magnusfroste/tokenizer/internal/retention"
	"github.com/magnusfroste/tokenizer/internal/spend"
)

type Config struct {
	Logger                 *slog.Logger
	KeyStore               auth.KeyStore
	Provider               provider.Adapter // legacy: used when Engine is nil
	ContextPipeline        *contextproc.Pipeline
	ContextPipelineEnabled bool
	PromptAdapter          *provider.PromptAdapter
	Readiness              []ReadyzChecker

	// Routing engine (Sprint 05). Optional — if nil, Provider is used directly.
	Engine            *engine.Engine
	Adapters          map[string]provider.Adapter // provider ID → adapter
	PolicyCache       *policy.Cache
	ShadowPolicyCache *policy.Cache

	// Observability (Sprint 06). All optional.
	HealthTracker     *health.Tracker
	SpendTracker      *spend.Tracker
	EventQueue        *eventlog.Queue
	ComparisonTracker *eventlog.ComparisonTracker
	RegistryVersion   string // shown on dashboard

	// Feedback (Sprint 07). Optional.
	OutcomeStore *outcomes.Store

	// Security (Sprint 08). Optional audit trail for blocked requests and
	// control-plane changes (ISSUE-044).
	Auditor audit.Sink

	// Retention/privacy settings (ISSUE-045). Optional; gates prompt logging.
	Retention *retention.Settings

	// Budget caps (ISSUE-051). Optional; blocks or downgrades over-budget scopes.
	Budget *budget.Evaluator

	// Route decision cache (ISSUE-052). Optional; caches low-risk decisions.
	DecisionCache *decisioncache.Cache

	// Premium-tier per-token pricing (micros per million tokens) for the
	// dashboard "saved vs all-premium" baseline. Optional.
	PremiumInputMicrosPerMTok  int64
	PremiumOutputMicrosPerMTok int64
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
		PromptAdapter:          cfg.PromptAdapter,
		Logger:                 cfg.Logger,
		Engine:                 cfg.Engine,
		Adapters:               cfg.Adapters,
		PolicyCache:            cfg.PolicyCache,
		ShadowPolicyCache:      cfg.ShadowPolicyCache,
		HealthTracker:          cfg.HealthTracker,
		EventQueue:             cfg.EventQueue,
		Auditor:                cfg.Auditor,
		Retention:              cfg.Retention,
		Budget:                 cfg.Budget,
		DecisionCache:          cfg.DecisionCache,
		RegistryVersion:        cfg.RegistryVersion,
	})
	mux.Handle("POST /v1/chat/completions",
		auth.Middleware(cfg.KeyStore)(auth.RequireScope(auth.ScopeChatCompletions)(chat)))

	// OpenAI-compatible model discovery. Requires a valid key (like OpenAI) but
	// no granular scope — listing models is read-only and benign.
	if cfg.Engine != nil {
		mux.Handle("GET /v1/models",
			auth.Middleware(cfg.KeyStore)(ModelsHandler(cfg.Engine)))
	}

	if cfg.Engine != nil {
		decision := DecisionHandler(DecisionOptions{
			Engine:      cfg.Engine,
			PolicyCache: cfg.PolicyCache,
			Logger:      cfg.Logger,
		})
		mux.Handle("POST /router/decision",
			auth.Middleware(cfg.KeyStore)(auth.RequireScope(auth.ScopeRouterDecision)(decision)))
	}

	// Outcome feedback API (ISSUE-039).
	if cfg.OutcomeStore != nil {
		outcome := OutcomeHandler(OutcomeOptions{Store: cfg.OutcomeStore, Logger: cfg.Logger})
		mux.Handle("POST /router/outcomes",
			auth.Middleware(cfg.KeyStore)(auth.RequireScope(auth.ScopeRouterOutcomes)(outcome)))
	}

	// Dashboard (no auth — read-only aggregated stats).
	htmlH, dataH := DashboardHandler(DashboardOptions{
		Spend:                      cfg.SpendTracker,
		Health:                     cfg.HealthTracker,
		Outcomes:                   cfg.OutcomeStore,
		Comparisons:                cfg.ComparisonTracker,
		Logger:                     cfg.Logger,
		Version:                    cfg.RegistryVersion,
		PremiumInputMicrosPerMTok:  cfg.PremiumInputMicrosPerMTok,
		PremiumOutputMicrosPerMTok: cfg.PremiumOutputMicrosPerMTok,
	})
	mux.Handle("GET /router/dashboard",
		auth.Middleware(cfg.KeyStore)(auth.RequireRole(auth.RoleAdmin)(http.HandlerFunc(htmlH))))
	mux.Handle("GET /router/dashboard/data",
		auth.Middleware(cfg.KeyStore)(auth.RequireRole(auth.RoleAdmin)(http.HandlerFunc(dataH))))

	return middleware.RequestID(middleware.Logger(cfg.Logger)(mux))
}
