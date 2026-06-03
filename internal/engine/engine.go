package engine

import (
	"errors"
	"fmt"

	"github.com/magnusfroste/tokenizer/internal/policy"
	"github.com/magnusfroste/tokenizer/internal/registry"
	"github.com/magnusfroste/tokenizer/internal/router"
)

var (
	ErrNoRoute  = errors.New("engine: no route found")
	ErrBlocked  = errors.New("engine: request blocked by policy")
	ErrDisabled = errors.New("engine: routing disabled")
)

// Engine is the stateless routing decision engine.
// It always reads the active snapshot from the store so hot-reloads are transparent.
type Engine struct {
	Registry *registry.Store
	Weights  Weights
}

// New creates an Engine backed by the given registry store, using default weights.
func New(store *registry.Store) *Engine {
	return &Engine{
		Registry: store,
		Weights:  DefaultWeights(),
	}
}

// Decide computes a RouteDecision for the given job.
//
//   - pol may be nil (no policy constraints applied).
//   - health may be nil (all providers assumed fully healthy).
//   - streaming indicates whether the upstream request will use server-sent events.
//
// Decide never calls any provider, LLM, or external service.
func (e *Engine) Decide(
	job *router.JobDescriptor,
	pol *policy.CompiledPolicy,
	health HealthSnapshot,
	streaming bool,
) (RouteDecision, error) {
	snap, err := e.Registry.Active()
	if err != nil {
		return RouteDecision{}, fmt.Errorf("engine: registry unavailable: %w", err)
	}
	if health == nil {
		health = FullyHealthy
	}

	// RouterMode=disabled skips routing entirely.
	if job.RouterMode == router.RouterModeDisabled {
		return decideDisabled(job, pol)
	}

	// Evaluate compiled policy.
	eval := evaluatePolicy(pol, job)
	if eval.Blocked {
		b := eval.Route.Block
		dec := RouteDecision{
			Blocked:         true,
			BlockReason:     b.Reason,
			DecisionReasons: eval.Explanations,
			PolicyVersion:   eval.PolicyVersion,
			BlockStatus:     403,
		}
		if b.Code != "" {
			dec.BlockCode = b.Code
		}
		if b.Status != 0 {
			dec.BlockStatus = b.Status
		}
		return dec, ErrBlocked
	}

	// Explicit model in request (client override).
	if job.ExplicitModel != nil {
		return decidePinned(*job.ExplicitModel, "explicit model in request", job, eval, snap)
	}

	// Policy force.model pins the model.
	if eval.Route.Force != nil && eval.Route.Force.Model != "" {
		return decidePinned(eval.Route.Force.Model, "policy force.model", job, eval, snap)
	}

	// Filter candidates.
	filterRes := FilterCandidates(job, eval.Route, snap, health, streaming)
	if len(filterRes.Candidates) == 0 {
		return RouteDecision{}, fmt.Errorf("%w: no models pass filters (task=%s risk=%s streaming=%v excluded=%d)",
			ErrNoRoute, job.TaskType, job.RiskLevel, streaming, len(filterRes.Excluded))
	}

	// Score and rank.
	minTier := MinimumTierForTask(job.TaskType, job.RiskLevel, eval.Route)
	scored := ScoreCandidates(filterRes.Candidates, job, minTier, health, e.Weights)
	primary := scored[0].Model

	// Build fallback chain.
	fallbacks := BuildFallbackChain(primary, scored, job, eval.Route)

	// Cost estimate.
	costUSD := estimateCostMicroUSD(primary, job) / 1_000_000

	return RouteDecision{
		RouteID:          routeID(job, primary.ID),
		SelectedModel:    primary.ID,
		SelectedProvider: primary.ProviderID,
		ProviderModelID:  primary.ProviderModelID,
		Fallbacks:        fallbacks,
		TimeoutMS:        resolveTimeout(pol, eval.Route, job),
		RequiresVerifier: resolveVerifier(job, eval.Route),
		DecisionReasons:  buildReasons(job, eval, filterRes, scored, primary, fallbacks),
		PolicyVersion:    eval.PolicyVersion,
		EstimatedCostUSD: costUSD,
	}, nil
}

func decideDisabled(job *router.JobDescriptor, pol *policy.CompiledPolicy) (RouteDecision, error) {
	if job.ExplicitModel == nil {
		return RouteDecision{}, fmt.Errorf("%w: router_mode=disabled requires an explicit model field", ErrDisabled)
	}
	eval := evaluatePolicy(pol, job)
	return RouteDecision{
		RouteID:         routeID(job, *job.ExplicitModel),
		SelectedModel:   *job.ExplicitModel,
		ProviderModelID: *job.ExplicitModel,
		DecisionReasons: append(eval.Explanations, "routing disabled, using client model as-is"),
		PolicyVersion:   eval.PolicyVersion,
	}, nil
}

func decidePinned(
	modelRef string,
	source string,
	job *router.JobDescriptor,
	eval policy.Evaluation,
	snap *registry.Snapshot,
) (RouteDecision, error) {
	model, ok := snap.Model(modelRef)
	if !ok {
		// Fall back to searching by ProviderModelID.
		found := findByProviderModelID(snap, modelRef)
		if found == nil {
			return RouteDecision{}, fmt.Errorf("%w: %s %q not found in registry", ErrNoRoute, source, modelRef)
		}
		model = *found
	}
	return RouteDecision{
		RouteID:          routeID(job, model.ID),
		SelectedModel:    model.ID,
		SelectedProvider: model.ProviderID,
		ProviderModelID:  model.ProviderModelID,
		DecisionReasons:  append(eval.Explanations, fmt.Sprintf("%s: %q", source, modelRef)),
		PolicyVersion:    eval.PolicyVersion,
	}, nil
}

func findByProviderModelID(snap *registry.Snapshot, providerModelID string) *registry.Model {
	for _, m := range snap.EnabledModelsWithCapabilities(registry.Capabilities{Chat: true}) {
		if m.ProviderModelID == providerModelID {
			return &m
		}
	}
	return nil
}

func evaluatePolicy(pol *policy.CompiledPolicy, job *router.JobDescriptor) policy.Evaluation {
	if pol == nil {
		return policy.Evaluation{}
	}
	return pol.Evaluate(policy.EvaluationInput{
		TenantID:             job.TenantID,
		ProjectID:            job.ProjectID,
		TaskType:             string(job.TaskType),
		RiskLevel:            string(job.RiskLevel),
		Sensitivity:          string(job.Sensitivity),
		PromptTokensEstimate: job.PromptTokensEstimate,
		Keywords:             append([]string(nil), job.Keywords...),
		FilesTouched:         append([]string(nil), job.FilesTouched...),
		RequiresToolUse:      job.RequiresToolUse,
		RequiresJSONSchema:   job.RequiresJSONSchema,
		RequiresVision:       job.RequiresVision,
		RouterMode:           string(job.RouterMode),
	})
}

func resolveTimeout(pol *policy.CompiledPolicy, route policy.Route, job *router.JobDescriptor) int {
	if f := route.Force; f != nil && f.TimeoutMS != nil {
		return *f.TimeoutMS
	}
	if d := route.Defaults; d != nil && d.TimeoutMS != nil {
		return *d.TimeoutMS
	}
	if pol != nil && pol.Settings().DefaultTimeoutMS > 0 {
		return pol.Settings().DefaultTimeoutMS
	}
	if job.RequiresLargeContext {
		return 60_000
	}
	return 30_000
}

func resolveVerifier(job *router.JobDescriptor, route policy.Route) bool {
	if f := route.Force; f != nil && f.Verifier != nil {
		return *f.Verifier
	}
	if d := route.Defaults; d != nil && d.Verifier != nil {
		return *d.Verifier
	}
	return job.TaskType == router.TaskSecurityReview || job.TaskType == router.TaskDatabaseMigration
}

func buildReasons(
	job *router.JobDescriptor,
	eval policy.Evaluation,
	filterRes FilterResult,
	scored []ScoredCandidate,
	primary registry.Model,
	fallbacks []FallbackEntry,
) []string {
	reasons := make([]string, 0, 6)
	reasons = append(reasons, fmt.Sprintf("task=%s risk=%s sensitivity=%s tokens≈%d",
		job.TaskType, job.RiskLevel, job.Sensitivity, job.PromptTokensEstimate))
	reasons = append(reasons, filterRes.Reasons...)
	reasons = append(reasons, eval.Explanations...)
	if len(scored) > 0 {
		reasons = append(reasons, fmt.Sprintf("selected %s (tier=%s provider=%s score=%.4f)",
			primary.ID, primary.Tier, primary.ProviderID, scored[0].Score))
		reasons = append(reasons, scored[0].Reasons...)
	}
	reasons = append(reasons, FallbackChainReason(primary, fallbacks))
	return reasons
}

func routeID(job *router.JobDescriptor, modelID string) string {
	if job.RequestID != "" {
		return fmt.Sprintf("route_%s_%s", job.RequestID, modelID)
	}
	return fmt.Sprintf("route_%s", modelID)
}
