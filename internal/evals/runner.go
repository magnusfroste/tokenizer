package evals

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/magnusfroste/tokenizer/internal/engine"
	"github.com/magnusfroste/tokenizer/internal/openai"
	"github.com/magnusfroste/tokenizer/internal/registry"
	"github.com/magnusfroste/tokenizer/internal/router"
)

// Runner executes eval cases against the routing engine.
type Runner struct {
	Engine   *engine.Engine
	Snapshot *registry.Snapshot
}

// NewRunner builds a Runner backed by the default registry.
func NewRunner() (*Runner, error) {
	snap, err := registry.DefaultSnapshot()
	if err != nil {
		return nil, err
	}
	store, err := registry.NewStore(snap)
	if err != nil {
		return nil, err
	}
	return &Runner{Engine: engine.New(store), Snapshot: snap}, nil
}

// CaseResult is the outcome of a single eval case.
type CaseResult struct {
	Case             Case
	Pass             bool
	ClassifiedTask   string
	SelectedModel    string
	SelectedTier     string
	ExpectedTier     string
	EstimatedCostUSD float64
	RoutingMicros    int64
	Reason           string // failure reason when !Pass
}

// Report aggregates results across the whole dataset.
type Report struct {
	Total          int
	Passed         int
	Results        []CaseResult
	TotalCostUSD   float64
	MeanRoutingMic float64
}

// PassRate returns the fraction of cases that routed as expected.
func (r Report) PassRate() float64 {
	if r.Total == 0 {
		return 0
	}
	return float64(r.Passed) / float64(r.Total)
}

// Run executes every case in the dataset and returns a Report.
func (rn *Runner) Run(ds *Dataset) (Report, error) {
	report := Report{Total: len(ds.Cases)}
	var totalMicros int64
	for _, c := range ds.Cases {
		res := rn.runCase(c)
		if res.Pass {
			report.Passed++
		}
		report.TotalCostUSD += res.EstimatedCostUSD
		totalMicros += res.RoutingMicros
		report.Results = append(report.Results, res)
	}
	if report.Total > 0 {
		report.MeanRoutingMic = float64(totalMicros) / float64(report.Total)
	}
	return report, nil
}

func (rn *Runner) runCase(c Case) CaseResult {
	req := &openai.ChatRequest{
		Model:    "auto",
		Messages: []openai.Message{{Role: "user", Content: c.PromptText()}},
	}
	if c.ExplicitModel != "" {
		req.Model = c.ExplicitModel
	}
	if c.RouterMode != "" {
		req.Metadata = map[string]any{"router_mode": c.RouterMode}
	}

	job := router.NewJobDescriptor(router.JobDescriptorInput{
		RequestID: "eval_" + c.ID,
		Request:   req,
	})

	start := time.Now()
	dec, err := rn.Engine.Decide(job, nil, engine.FullyHealthy, false)
	micros := time.Since(start).Microseconds()

	res := CaseResult{
		Case:           c,
		ClassifiedTask: string(job.TaskType),
		ExpectedTier:   c.ExpectedRoute.Tier,
		RoutingMicros:  micros,
	}
	if err != nil {
		res.Reason = fmt.Sprintf("engine error: %v", err)
		return res
	}

	res.SelectedModel = dec.SelectedModel
	res.EstimatedCostUSD = dec.EstimatedCostUSD
	if model, ok := rn.Snapshot.Model(dec.SelectedModel); ok {
		res.SelectedTier = string(model.Tier)
	}

	// Model pin takes precedence when specified.
	if c.ExpectedRoute.Model != "" {
		res.Pass = dec.SelectedModel == c.ExpectedRoute.Model
		if !res.Pass {
			res.Reason = fmt.Sprintf("expected model %s, got %s", c.ExpectedRoute.Model, dec.SelectedModel)
		}
		return res
	}

	res.Pass = res.SelectedTier == c.ExpectedRoute.Tier
	if !res.Pass {
		res.Reason = fmt.Sprintf("expected tier %s, got %s (model %s, classified %s)",
			c.ExpectedRoute.Tier, res.SelectedTier, dec.SelectedModel, res.ClassifiedTask)
	}
	return res
}

// FormatReport renders a human-readable text report.
func FormatReport(r Report) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Eval report: %d/%d passed (%.1f%%)\n", r.Passed, r.Total, r.PassRate()*100)
	fmt.Fprintf(&b, "Total estimated cost: $%.6f  ·  mean routing: %.1fµs\n", r.TotalCostUSD, r.MeanRoutingMic)

	// Group failures for readability.
	var failures []CaseResult
	for _, res := range r.Results {
		if !res.Pass {
			failures = append(failures, res)
		}
	}
	if len(failures) == 0 {
		b.WriteString("All cases routed as expected.\n")
		return b.String()
	}
	sort.Slice(failures, func(i, j int) bool { return failures[i].Case.ID < failures[j].Case.ID })
	fmt.Fprintf(&b, "\nFailures (%d):\n", len(failures))
	for _, f := range failures {
		fmt.Fprintf(&b, "  - %s (%s): %s\n", f.Case.ID, f.Case.Name, f.Reason)
	}
	return b.String()
}
