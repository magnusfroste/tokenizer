package engine

import (
	"fmt"

	"github.com/magnusfroste/tokenizer/internal/policy"
	"github.com/magnusfroste/tokenizer/internal/registry"
	"github.com/magnusfroste/tokenizer/internal/router"
)

// FilterResult holds filtered candidates and per-model exclusion reasons.
type FilterResult struct {
	Candidates []registry.Model
	Excluded   map[string]string // model ID → human-readable reason
	Reasons    []string          // positive filtering notes
}

// minHealthThreshold is the minimum provider health score to be eligible.
const minHealthThreshold = 0.1

// FilterCandidates applies hard capability, policy, and health filters to the
// registry. Models that fail any filter are excluded. The function never calls
// a provider or LLM.
func FilterCandidates(
	job *router.JobDescriptor,
	route policy.Route,
	snapshot *registry.Snapshot,
	health HealthSnapshot,
	streaming bool,
) FilterResult {
	if health == nil {
		health = FullyHealthy
	}
	result := FilterResult{Excluded: make(map[string]string)}

	required := requiredCapabilities(job, streaming)
	policyCaps := capabilitiesFromPolicy(route.Constraints)
	merged := mergeCapabilities(required, policyCaps)

	candidates := snapshot.EnabledModelsWithCapabilities(merged)
	minTier := MinimumTierForTask(job.TaskType, job.RiskLevel, route)

	for _, model := range candidates {
		if reason := hardFilter(model, job, route, health, minTier); reason != "" {
			result.Excluded[model.ID] = reason
			continue
		}
		result.Candidates = append(result.Candidates, model)
	}

	result.Reasons = append(result.Reasons, fmt.Sprintf(
		"%d candidate(s) after filtering (chat=%v streaming=%v tools=%v vision=%v long_context=%v min_tier=%s)",
		len(result.Candidates), merged.Chat, merged.Streaming, merged.ToolCalls, merged.Vision, merged.LongContext, minTier,
	))
	return result
}

func hardFilter(
	model registry.Model,
	job *router.JobDescriptor,
	route policy.Route,
	health HealthSnapshot,
	minTier registry.Tier,
) string {
	// Minimum tier for task / risk combination.
	if !TierAtLeast(model.Tier, minTier) {
		return fmt.Sprintf("tier %s below minimum %s for task=%s risk=%s", model.Tier, minTier, job.TaskType, job.RiskLevel)
	}

	// Provider health.
	if h := health.ProviderHealth(model.ProviderID); h < minHealthThreshold {
		return fmt.Sprintf("provider %s health %.2f below threshold", model.ProviderID, h)
	}

	// Policy constraints.
	if c := route.Constraints; c != nil {
		if reason := providerModelConstraintReason(model, c); reason != "" {
			return reason
		}
		if c.MaxLatencyMS != nil && model.Latency.P95FirstTokenMS > 0 && model.Latency.P95FirstTokenMS > *c.MaxLatencyMS {
			return fmt.Sprintf("p95 latency %dms exceeds policy max %dms", model.Latency.P95FirstTokenMS, *c.MaxLatencyMS)
		}
		for _, denied := range c.DenyCapabilities {
			switch denied {
			case policy.CapStreaming:
				if model.Capabilities.Streaming {
					return "policy denies streaming capability"
				}
			case policy.CapToolUse:
				if model.Capabilities.ToolCalls {
					return "policy denies tool_use capability"
				}
			case policy.CapVision:
				if model.Capabilities.Vision {
					return "policy denies vision capability"
				}
			case policy.CapLongContext:
				if model.Capabilities.LongContext {
					return "policy denies long_context capability"
				}
			}
		}
	}

	// Policy force narrows to one model or provider.
	if f := route.Force; f != nil {
		if f.Model != "" && model.ID != f.Model && model.ProviderModelID != f.Model {
			return fmt.Sprintf("policy forces model %s", f.Model)
		}
		if f.Provider != "" && model.ProviderID != f.Provider {
			return fmt.Sprintf("policy forces provider %s", f.Provider)
		}
		if f.ModelProfile != "" && string(model.Tier) != string(f.ModelProfile) {
			return fmt.Sprintf("policy forces profile %s", f.ModelProfile)
		}
	}

	// RouterMode hard filters.
	if job.RouterMode == router.RouterModeCheap && minTier != registry.TierPremium {
		if model.Tier == registry.TierPremium {
			return "router_mode=cheap excludes premium tier"
		}
	}

	return ""
}

// MinimumTierForTask returns the lowest acceptable model tier for the task/risk
// combination after applying any policy profile force.
func MinimumTierForTask(task router.TaskType, risk router.RiskLevel, route policy.Route) registry.Tier {
	if f := route.Force; f != nil && f.ModelProfile != "" {
		switch f.ModelProfile {
		case policy.ProfilePremium:
			return registry.TierPremium
		case policy.ProfileBalanced:
			return registry.TierBalanced
		case policy.ProfileCheap:
			return registry.TierCheap
		}
	}
	switch task {
	case router.TaskSecurityReview, router.TaskDatabaseMigration, router.TaskUnknownHighRisk:
		return registry.TierPremium
	case router.TaskHardCodeDebugging, router.TaskLongContextAnalysis:
		return registry.TierBalanced
	default:
		if risk == router.RiskCritical || risk == router.RiskHigh {
			return registry.TierBalanced
		}
		return registry.TierCheap
	}
}

// TierAtLeast reports whether tier meets or exceeds minimum.
func TierAtLeast(tier, minimum registry.Tier) bool {
	return TierOrdinal(tier) >= TierOrdinal(minimum)
}

// TierOrdinal maps a tier to a sortable integer (higher = more capable).
func TierOrdinal(tier registry.Tier) int {
	switch tier {
	case registry.TierCheap:
		return 0
	case registry.TierBalanced:
		return 1
	case registry.TierPremium:
		return 2
	default:
		return -1
	}
}

func requiredCapabilities(job *router.JobDescriptor, streaming bool) registry.Capabilities {
	return registry.Capabilities{
		Chat:        true,
		Streaming:   streaming,
		ToolCalls:   job.RequiresToolUse,
		JSONSchema:  job.RequiresJSONSchema,
		Vision:      job.RequiresVision,
		LongContext: job.RequiresLargeContext,
	}
}

func capabilitiesFromPolicy(c *policy.Constraints) registry.Capabilities {
	if c == nil {
		return registry.Capabilities{}
	}
	var caps registry.Capabilities
	for _, cap := range c.RequireCapabilities {
		switch cap {
		case policy.CapStreaming:
			caps.Streaming = true
		case policy.CapToolUse:
			caps.ToolCalls = true
		case policy.CapJSONSchema:
			caps.JSONSchema = true
		case policy.CapVision:
			caps.Vision = true
		case policy.CapLongContext:
			caps.LongContext = true
		}
	}
	return caps
}

func mergeCapabilities(a, b registry.Capabilities) registry.Capabilities {
	return registry.Capabilities{
		Chat:        a.Chat || b.Chat,
		Streaming:   a.Streaming || b.Streaming,
		ToolCalls:   a.ToolCalls || b.ToolCalls,
		JSONSchema:  a.JSONSchema || b.JSONSchema,
		Vision:      a.Vision || b.Vision,
		LongContext: a.LongContext || b.LongContext,
	}
}

// providerModelConstraintReason returns a non-empty exclusion reason if the
// model violates the policy's provider/model allow or deny lists. It is the
// single source of truth for these checks, shared by candidate filtering and by
// pinned-model decisions (explicit client model, policy force.model, disabled
// mode) — so no override path can ever bypass a project's denylist or allowlist.
func providerModelConstraintReason(model registry.Model, c *policy.Constraints) string {
	if c == nil {
		return ""
	}
	if len(c.AllowedProviders) > 0 && !containsStr(c.AllowedProviders, model.ProviderID) {
		return fmt.Sprintf("provider %s not in allowed_providers", model.ProviderID)
	}
	if containsStr(c.DeniedProviders, model.ProviderID) {
		return fmt.Sprintf("provider %s in denied_providers", model.ProviderID)
	}
	if len(c.AllowedModels) > 0 && !containsStr(c.AllowedModels, model.ID) {
		return fmt.Sprintf("model %s not in allowed_models", model.ID)
	}
	if containsStr(c.DeniedModels, model.ID) {
		return fmt.Sprintf("model %s in denied_models", model.ID)
	}
	return ""
}

func containsStr(slice []string, value string) bool {
	for _, s := range slice {
		if s == value {
			return true
		}
	}
	return false
}
