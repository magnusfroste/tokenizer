# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project

Tokenizer is an OpenAI-compatible model router / gateway written in **Go**. It sits between clients and LLM providers, classifying each prompt and routing it to the right model based on task type, risk, cost, latency, and tenant policy — without using an LLM for routing decisions.

The design target is routing overhead p95 < 100 ms before any provider call.

## Development commands

```bash
cp .env.example .env
docker compose up -d postgres redis mock-provider
make migrate
make seed
make dev
```

Run tests:
```bash
make test              # unit + integration
make test-unit         # unit only
make test-policy       # policy golden cases
make test-eval         # eval smoke (50+ prompt cases)
go test ./internal/classifier/... -run TestFeatureExtraction
```

Lint:
```bash
make lint
```

Dry-run a routing decision without calling a provider:
```bash
curl -X POST http://localhost:8080/router/decision \
  -H "Authorization: Bearer local_router_key" \
  -d @examples/debug-request.json
```

Test a policy file against golden cases:
```bash
router policy test --policy policies/default.yaml --cases tests/policy-cases.yaml
```

## Architecture

### Request lifecycle

```
POST /v1/chat/completions
  → Auth (API key → tenant + policy context, cached)
  → Feature Extractor (rule-based, no LLM) → JobDescriptor
  → Policy Engine (precompiled, in-memory) → constraints
  → Routing Decision Engine (score candidates) → RouteDecision
  → Provider Executor (format translation + HTTP) → normalized response
  → Event Log (async)
  → Client
```

Response headers carry routing metadata: `x-router-request-id`, `x-router-selected-model`, `x-router-policy-version`, `x-router-route-class`.

### Key types

**`JobDescriptor`** — the internal contract produced by feature extraction. Fields include `task_type`, `risk_level`, `sensitivity`, `prompt_tokens_estimate`, `requires_reasoning`, `requires_tool_use`, `latency_preference`, `quality_preference`, `router_mode`. Defined in `06-engineering/02-job-descriptor-schema.md`.

**`RouteDecision`** — output of the routing engine: selected model, fallback chain, timeout, verifier flag, decision reasons, policy version. Defined in `01-architecture/04-routing-engine.md`.

**`NormalizedModelRequest` / `NormalizedModelResponse`** — internal wire format between router and provider adapters. Each provider has its own adapter that maps to/from this format.

### Core subsystems

**Feature extractor** — pure rule-based signal extraction (token count, code detection, keywords, file names, stack traces, SQL/auth/payment terms). Must never call an LLM. Target: p95 < 20 ms.

**Policy engine** — YAML policy compiled to in-memory rules. Evaluation order: block → force → constraints → hints → defaults. Rules match on `task_type`, `risk_level`, `sensitivity`, `prompt_tokens_gt/lt`, `contains_any`, `any_file_matches`, etc. Policy can be rolled back independently of code (hotreload).

**Routing algorithm** — filters candidates by capability + provider health, then scores:
```
score = quality_weight * predicted_quality
      + capability_weight * capability_match
      + health_weight * provider_health
      - cost_weight * estimated_cost
      - latency_weight * expected_latency
      - risk_penalty_for_underpowered_model
```
For risky tasks, fallback is always upward (more capable), never downward.

**Provider adapters** — one `ProviderAdapter` interface per provider (OpenAI, Anthropic, etc.). Responsible for request format translation, streaming chunk parsing, tool call mapping, token usage normalization, and error normalization to internal error types (`provider_timeout`, `provider_rate_limit`, `provider_5xx`, etc.).

**Fast path vs slow path** — the fast path (majority of requests) uses only in-memory rules, token estimates, and cached data. The slow path (uncertain/high-risk decisions) may invoke a local lightweight classifier or verifier model.

### Infrastructure

- **Postgres** — tenants, API keys, request logs, route attempts, event log, spend aggregation.
- **Redis** — API key cache, provider health cache, compiled policy cache, route decision cache.
- **Async event queue** — decision and attempt logging is non-blocking; failures here must not affect the request path.

### Task classes → model tiers

| Task class | Tier |
|---|---|
| `trivial_git`, `simple_shell` | cheap/local |
| `summarization`, `simple_code_edit` | balanced |
| `hard_code_debugging` | premium-reasoning |
| `security_review`, `database_migration` | premium + verifier |
| `long_context_analysis` | long-context model |
| `unknown_high_risk` | premium |

### Documentation structure

```
00-product/       Product vision, PRD, personas, roadmap
01-architecture/  System design (15 docs) — start here for any subsystem
02-adr/           Architecture decisions (12 ADRs)
03-backlog/       Epics (EPIC-01 through EPIC-10)
04-sprints/       Sprint plans (sprints 0–8)
05-issues/        Implementable issues (ISSUE-001 through ISSUE-060)
06-engineering/   Technical references: routing policy, classifier, latency, testing, CI/CD
07-operations/    Runbooks, SLO, incident response
08-templates/     ADR, issue, epic, sprint, policy, eval templates
```

Start at `01-architecture/01-system-overview.md` for any new subsystem. Each ISSUE file in `05-issues/` is a self-contained implementation task.

## Implementation constraints

- The fast-path routing decision must never call an LLM or external service — only in-memory data structures.
- Fallback chain must be constructed **before** the first provider call.
- Client metadata is not trusted — policy can override or escalate risk signals.
- Streaming fallback is only allowed before first token; after first token, restart requires explicit client opt-in.
- Policy and registry can be reloaded without a code deployment.

## Agent skills

### Issue tracker

Issues and PRDs are tracked as local markdown under `.scratch/<feature-slug>/`. See `docs/agents/issue-tracker.md`.

### Triage labels

This repo uses the default five-state triage vocabulary. See `docs/agents/triage-labels.md`.

### Domain docs

This repo uses a repo-specific single-context layout based on README, product docs, architecture docs, ADRs in `02-adr/`, and engineering references. See `docs/agents/domain.md`.
