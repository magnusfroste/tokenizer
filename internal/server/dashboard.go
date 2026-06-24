package server

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/magnusfroste/tokenizer/internal/eventlog"
	"github.com/magnusfroste/tokenizer/internal/health"
	"github.com/magnusfroste/tokenizer/internal/outcomes"
	"github.com/magnusfroste/tokenizer/internal/spend"
)

// DashboardOptions configures the dashboard handler.
type DashboardOptions struct {
	Spend       *spend.Tracker
	Health      *health.Tracker
	Outcomes    *outcomes.Store
	Comparisons *eventlog.ComparisonTracker
	RequestLog  *eventlog.RequestLogTracker
	Logger      *slog.Logger
	Version     string // registry version label

	// Premium-tier per-token pricing (micros per million tokens), used to
	// compute the "saved vs all-premium" baseline. Zero disables the card.
	PremiumInputMicrosPerMTok  int64
	PremiumOutputMicrosPerMTok int64
}

// SavingsSummary quantifies the headline value proposition: what the observed
// traffic actually cost versus what it would have cost if every request had used
// the premium model (same token usage, premium pricing).
type SavingsSummary struct {
	ActualUSD          float64 `json:"actual_usd"`
	PremiumBaselineUSD float64 `json:"premium_baseline_usd"`
	SavedUSD           float64 `json:"saved_usd"`
	SavedPct           float64 `json:"saved_pct"`
}

// DashboardData is the JSON payload returned by /router/dashboard/data.
type DashboardData struct {
	Version        string                      `json:"registry_version"`
	TotalRequests  int64                       `json:"total_requests"`
	TotalCostUSD   float64                     `json:"total_cost_usd"`
	Savings        SavingsSummary              `json:"savings"`
	RoutesByModel  []spend.ModelRow            `json:"routes_by_model"`
	SpendByTenant  []spend.TenantRow           `json:"spend_by_tenant"`
	ProviderHealth map[string]float64          `json:"provider_health"`
	Acceptance     []outcomes.AcceptanceRow    `json:"acceptance"`
	OutcomeCount   int                         `json:"outcome_count"`
	ShadowSummary  eventlog.ComparisonSummary  `json:"shadow_summary"`
	ShadowRecent   []eventlog.ComparisonRecord `json:"shadow_recent"`
	RecentRequests []eventlog.RequestLogRecord `json:"recent_requests"`
	TaskFilter     string                      `json:"task_filter,omitempty"`
}

// DashboardHandler returns the /router/dashboard HTML handler and
// /router/dashboard/data JSON handler.
func DashboardHandler(opts DashboardOptions) (html http.HandlerFunc, data http.HandlerFunc) {
	data = func(w http.ResponseWriter, r *http.Request) {
		d := buildDashboardData(opts, r.URL.Query().Get("task"))
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(d)
	}

	html = func(w http.ResponseWriter, r *http.Request) {
		d := buildDashboardData(opts, r.URL.Query().Get("task"))
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := dashboardTmpl.Execute(w, d); err != nil {
			slog.Default().Error("dashboard template error", "err", err)
		}
	}
	return html, data
}

func buildDashboardData(opts DashboardOptions, taskFilter string) DashboardData {
	d := DashboardData{
		Version:        opts.Version,
		ProviderHealth: map[string]float64{},
		TaskFilter:     taskFilter,
	}
	if opts.Spend != nil {
		d.TotalRequests = opts.Spend.TotalRequests()
		d.TotalCostUSD = opts.Spend.TotalCostUSD()
		d.RoutesByModel = opts.Spend.ByModel()
		d.SpendByTenant = opts.Spend.ByTenant()
		d.Savings = computeSavings(d.RoutesByModel, d.TotalCostUSD, opts.PremiumInputMicrosPerMTok, opts.PremiumOutputMicrosPerMTok)
	}
	if opts.Health != nil {
		d.ProviderHealth = opts.Health.Providers()
	}
	if opts.Outcomes != nil {
		d.OutcomeCount = opts.Outcomes.Count()
		d.Acceptance = opts.Outcomes.Acceptance(taskFilter)
	}
	if opts.Comparisons != nil {
		d.ShadowSummary = opts.Comparisons.Summary()
		d.ShadowRecent = opts.Comparisons.Recent(taskFilter)
	}
	if opts.RequestLog != nil {
		d.RecentRequests = opts.RequestLog.Recent(50)
	}
	return d
}

// computeSavings estimates spend versus an all-premium baseline: every request's
// actual token usage repriced at the premium model's per-token rate. The
// baseline is an estimate (premium might generate different-length outputs), so
// it is the conservative, observable "what you'd pay without routing" figure.
func computeSavings(rows []spend.ModelRow, actualUSD float64, premInMicrosPerMTok, premOutMicrosPerMTok int64) SavingsSummary {
	s := SavingsSummary{ActualUSD: actualUSD}
	if premInMicrosPerMTok <= 0 && premOutMicrosPerMTok <= 0 {
		return s
	}
	var premiumUSD float64
	for _, r := range rows {
		// micros = tokens * (microsPerMillionTok / 1e6); USD = micros / 1e6.
		premiumUSD += (float64(r.InputTokens)*float64(premInMicrosPerMTok) +
			float64(r.OutputTokens)*float64(premOutMicrosPerMTok)) / 1e12
	}
	s.PremiumBaselineUSD = premiumUSD
	s.SavedUSD = premiumUSD - actualUSD
	if premiumUSD > 0 {
		s.SavedPct = (1 - actualUSD/premiumUSD) * 100
	}
	return s
}

var dashboardTmpl = template.Must(template.New("dashboard").Funcs(template.FuncMap{
	"usd":   func(v float64) string { return fmt.Sprintf("$%.6f", v) },
	"pct":   func(v float64) string { return fmt.Sprintf("%.0f%%", v*100) },
	"clock": func(t time.Time) string { return t.Local().Format("15:04:05") },
	"healthClass": func(v float64) string {
		switch {
		case v >= 0.9:
			return "ok"
		case v >= 0.5:
			return "warn"
		default:
			return "bad"
		}
	},
	"bar": func(requests, total int64) string {
		if total == 0 {
			return "0%"
		}
		pct := float64(requests) / float64(total) * 100
		return fmt.Sprintf("%.0f%%", pct)
	},
	"repeat": func(n int, s string) string { return strings.Repeat(s, n) },
}).Parse(`<!DOCTYPE html>
<html lang="sv">
<head>
<meta charset="utf-8">
<title>Tokenizer — Router Dashboard</title>
<style>
*{box-sizing:border-box;margin:0;padding:0}
body{font-family:system-ui,sans-serif;background:#0f1117;color:#e2e8f0;padding:2rem}
h1{font-size:1.5rem;font-weight:700;margin-bottom:0.25rem;color:#f8fafc}
.subtitle{font-size:0.85rem;color:#64748b;margin-bottom:2rem}
.grid{display:grid;grid-template-columns:repeat(auto-fit,minmax(180px,1fr));gap:1rem;margin-bottom:2rem}
.card{background:#1e2330;border:1px solid #2d3748;border-radius:10px;padding:1.25rem}
.card-label{font-size:0.75rem;color:#64748b;text-transform:uppercase;letter-spacing:.05em;margin-bottom:0.35rem}
.card-value{font-size:1.75rem;font-weight:700;color:#f8fafc}
.card-sub{font-size:0.78rem;color:#94a3b8;margin-top:0.2rem}
.hero{background:linear-gradient(135deg,#064e3b,#065f46);border:1px solid #10b981;border-radius:12px;padding:1.5rem 1.75rem;margin-bottom:1.5rem}
.hero-label{font-size:0.78rem;color:#a7f3d0;text-transform:uppercase;letter-spacing:.06em}
.hero-value{font-size:2.6rem;font-weight:800;color:#34d399;line-height:1.1;margin:0.15rem 0}
.hero-sub{font-size:0.95rem;color:#d1fae5}
table{width:100%;border-collapse:collapse;margin-bottom:2rem}
th{text-align:left;font-size:0.72rem;color:#64748b;text-transform:uppercase;letter-spacing:.05em;padding:0.6rem 0.75rem;border-bottom:1px solid #2d3748}
td{padding:0.6rem 0.75rem;border-bottom:1px solid #1e2330;font-size:0.88rem}
tr:hover td{background:#1e2330}
.bar-bg{background:#2d3748;border-radius:4px;height:6px;width:120px;display:inline-block;vertical-align:middle;margin-left:0.5rem}
.bar-fill{background:#3b82f6;border-radius:4px;height:6px;display:block}
h2{font-size:1rem;font-weight:600;margin-bottom:0.75rem;color:#cbd5e1}
section{margin-bottom:2.5rem}
.ok{color:#22c55e}.warn{color:#f59e0b}.bad{color:#ef4444}
.dot{display:inline-block;width:8px;height:8px;border-radius:50%;margin-right:6px;vertical-align:middle}
.dot-ok{background:#22c55e}.dot-warn{background:#f59e0b}.dot-bad{background:#ef4444}
.mono{font-family:ui-monospace,monospace;font-size:0.82rem}
</style>
</head>
<body>
<h1>⚡ Tokenizer Router Dashboard</h1>
<p class="subtitle">Registry: {{.Version}} &nbsp;·&nbsp; Live in-memory aggregation</p>

{{if gt .Savings.PremiumBaselineUSD 0.0}}
<div class="hero">
  <div class="hero-label">💸 Saved vs all-premium</div>
  <div class="hero-value">{{printf "%.1f%%" .Savings.SavedPct}} cheaper</div>
  <div class="hero-sub">Saved <strong>{{usd .Savings.SavedUSD}}</strong> — you paid {{usd .Savings.ActualUSD}}; routing everything to the premium model would have cost {{usd .Savings.PremiumBaselineUSD}}.</div>
</div>
{{end}}

<div class="grid">
  <div class="card">
    <div class="card-label">Total requests</div>
    <div class="card-value">{{.TotalRequests}}</div>
    <div class="card-sub">since last restart</div>
  </div>
  <div class="card">
    <div class="card-label">Estimated spend</div>
    <div class="card-value">{{usd .TotalCostUSD}}</div>
    <div class="card-sub">USD (estimate)</div>
  </div>
  <div class="card">
    <div class="card-label">Models tracked</div>
    <div class="card-value">{{len .RoutesByModel}}</div>
    <div class="card-sub">with requests</div>
  </div>
  <div class="card">
    <div class="card-label">Tenants</div>
    <div class="card-value">{{len .SpendByTenant}}</div>
    <div class="card-sub">active</div>
  </div>
  <div class="card">
    <div class="card-label">Shadow comparisons</div>
    <div class="card-value">{{.ShadowSummary.Total}}</div>
    <div class="card-sub">{{.ShadowSummary.ChangedCount}} changed vs actual</div>
  </div>
  <div class="card">
    <div class="card-label">Shadow cost delta</div>
    <div class="card-value">{{usd .ShadowSummary.EstimatedCostDeltaUSD}}</div>
    <div class="card-sub">shadow minus actual</div>
  </div>
</div>

<section>
<h2>Route distribution</h2>
<table>
<thead><tr><th>Model</th><th>Provider</th><th>Requests</th><th>Distribution</th><th>Input tokens</th><th>Output tokens</th><th>Cost (est.)</th></tr></thead>
<tbody>
{{$total := .TotalRequests}}
{{range .RoutesByModel}}
<tr>
  <td class="mono">{{.ModelID}}</td>
  <td class="mono">{{.ProviderID}}</td>
  <td>{{.Requests}}</td>
  <td>
    <span class="bar-bg"><span class="bar-fill" style="width:{{bar .Requests $total}}"></span></span>
    <span style="font-size:0.78rem;color:#94a3b8;margin-left:4px">{{bar .Requests $total}}</span>
  </td>
  <td>{{.InputTokens}}</td>
  <td>{{.OutputTokens}}</td>
  <td>{{usd .CostUSD}}</td>
</tr>
{{else}}<tr><td colspan="7" style="color:#64748b;text-align:center;padding:1.5rem">No requests yet — send some!</td></tr>
{{end}}
</tbody>
</table>
</section>

<section>
<h2>Recent requests</h2>
<table>
<thead><tr><th>Time</th><th>Task</th><th>Risk</th><th>Selected model</th><th>Provider</th><th>In</th><th>Out</th><th>Cost</th></tr></thead>
<tbody>
{{range .RecentRequests}}
<tr>
  <td class="mono">{{clock .Time}}</td>
  <td>{{if .Blocked}}<span class="bad">blocked</span> {{end}}{{.TaskType}}</td>
  <td>{{.RiskLevel}}</td>
  <td class="mono">{{.Model}}</td>
  <td class="mono">{{.Provider}}</td>
  <td>{{.InputTokens}}</td>
  <td>{{.OutputTokens}}</td>
  <td>{{usd .CostUSD}}</td>
</tr>
{{else}}<tr><td colspan="8" style="color:#64748b;text-align:center;padding:1.5rem">No requests yet — send some!</td></tr>
{{end}}
</tbody>
</table>
</section>

<section>
<h2>Provider health</h2>
<table>
<thead><tr><th>Provider</th><th>Health score</th><th>Status</th></tr></thead>
<tbody>
{{range $id, $score := .ProviderHealth}}
<tr>
  <td class="mono">{{$id}}</td>
  <td>{{pct $score}}</td>
  <td><span class="dot dot-{{healthClass $score}}"></span><span class="{{healthClass $score}}">{{if ge $score 0.9}}Healthy{{else if ge $score 0.5}}Degraded{{else}}Down{{end}}</span></td>
</tr>
{{else}}<tr><td colspan="3" style="color:#64748b;text-align:center;padding:1.5rem">No provider calls recorded yet</td></tr>
{{end}}
</tbody>
</table>
</section>

<section>
<h2>Shadow routing {{if .TaskFilter}}<span style="font-size:0.78rem;color:#64748b">· filtered: {{.TaskFilter}}</span>{{end}}</h2>
<p style="font-size:0.78rem;color:#64748b;margin-bottom:0.6rem">
  {{.ShadowSummary.ChangedCount}} / {{.ShadowSummary.Total}} comparisons changed · route {{.ShadowSummary.RouteChangedCount}} · fallback {{.ShadowSummary.FallbackChangedCount}} · timeout {{.ShadowSummary.TimeoutChangedCount}} · verifier {{.ShadowSummary.VerifierChangedCount}} · policy version {{.ShadowSummary.PolicyVersionChangedCount}} · cost {{.ShadowSummary.CostChangedCount}}
</p>
<table>
<thead><tr><th>Request</th><th>Task</th><th>Actual</th><th>Shadow</th><th>Changed</th><th>Cost delta</th><th>Policy versions</th></tr></thead>
<tbody>
{{range .ShadowRecent}}
<tr>
  <td class="mono">{{.RequestID}}</td>
  <td class="mono">{{.TaskType}}</td>
  <td class="mono">{{.Comparison.Primary.SelectedProvider}}/{{.Comparison.Primary.SelectedModel}}</td>
  <td class="mono">{{.Comparison.Secondary.SelectedProvider}}/{{.Comparison.Secondary.SelectedModel}}</td>
  <td>{{if .Comparison.Changed}}yes{{else}}no{{end}}</td>
  <td>{{usd .Comparison.EstimatedCostDeltaUSD}}</td>
  <td class="mono">{{.Comparison.Primary.PolicyVersion}} → {{.Comparison.Secondary.PolicyVersion}}</td>
</tr>
{{else}}<tr><td colspan="7" style="color:#64748b;text-align:center;padding:1.5rem">No shadow comparisons recorded yet</td></tr>
{{end}}
</tbody>
</table>
</section>

<section>
<h2>Acceptance feedback {{if .TaskFilter}}<span style="font-size:0.78rem;color:#64748b">· filtered: {{.TaskFilter}}</span>{{end}}</h2>
<p style="font-size:0.78rem;color:#64748b;margin-bottom:0.6rem">{{.OutcomeCount}} outcome(s) reported · filter by task class via <span class="mono">?task=&lt;task_type&gt;</span></p>
<table>
<thead><tr><th>Model</th><th>Task class</th><th>Outcomes</th><th>Accepted</th><th>Rejected</th><th>Partial</th><th>Acceptance rate</th></tr></thead>
<tbody>
{{range .Acceptance}}
<tr>
  <td class="mono">{{.Model}}</td>
  <td class="mono">{{.TaskType}}</td>
  <td>{{.Total}}</td>
  <td class="ok">{{.Accepted}}</td>
  <td class="bad">{{.Rejected}}</td>
  <td class="warn">{{.Partial}}</td>
  <td>
    <span class="bar-bg"><span class="bar-fill" style="width:{{pct .AcceptanceRate}};background:{{if ge .AcceptanceRate 0.7}}#22c55e{{else if ge .AcceptanceRate 0.4}}#f59e0b{{else}}#ef4444{{end}}"></span></span>
    <span style="font-size:0.78rem;color:#94a3b8;margin-left:4px">{{pct .AcceptanceRate}}</span>
  </td>
</tr>
{{else}}<tr><td colspan="7" style="color:#64748b;text-align:center;padding:1.5rem">No outcomes reported yet — POST to /router/outcomes</td></tr>
{{end}}
</tbody>
</table>
</section>

<section>
<h2>Spend by tenant</h2>
<table>
<thead><tr><th>Tenant</th><th>Requests</th><th>Cost (est.)</th></tr></thead>
<tbody>
{{range .SpendByTenant}}
<tr>
  <td class="mono">{{.TenantID}}</td>
  <td>{{.Requests}}</td>
  <td>{{usd .CostUSD}}</td>
</tr>
{{else}}<tr><td colspan="3" style="color:#64748b;text-align:center;padding:1.5rem">No tenant data yet</td></tr>
{{end}}
</tbody>
</table>
</section>

<p style="font-size:0.72rem;color:#334155">
  JSON: <a href="/router/dashboard/data" style="color:#3b82f6">/router/dashboard/data</a> &nbsp;·&nbsp;
  Metrics: <a href="/metrics" style="color:#3b82f6">/metrics</a>
</p>
</body>
</html>
`))
