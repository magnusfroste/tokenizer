# Issue index

Issues är skrivna som implementerbara tickets. Filnamn följer `ISSUE-XXX-title.md`.

## Triage snapshot 2026-05-19

Klart enligt kod- och test-evidens:

- ISSUE-001 till ISSUE-013.
- ISSUE-014 till ISSUE-019 — Classifier foundation: JobDescriptor, tokenestimat, feature extraction, task/riskregler och latency guard.
- ISSUE-020 till ISSUE-024 — Policy DSL v1, parser/validation, compiled policy cache, explanations och policy test runner.
- ISSUE-061 — Rebrand `tokenix` → `tokenizer`.
- ISSUE-062 — Context-processor pipeline (interface only).

Redo för agent:

- Inga öppna issues. Hela backlogen (ISSUE-001 till ISSUE-063) är `done` per
  2026-06-13 — se listan nedan.

Klart i sprint 05–08 (kod- och test-evidens):

- ISSUE-025 till ISSUE-041 — Routing/scoring/fallback (S05), observability (S06), evals/feedback (S07).
- ISSUE-042 — Secret masking v1.
- ISSUE-043 — Provider allow/deny per projekt.
- ISSUE-044 — Audit log (`internal/audit`: policy-reload, API-key-ändringar, blockerade requests).
- ISSUE-045 — Retention settings (`internal/retention`: per-tenant retention, prompt-logging-switch, cleanup-sweeper i `cmd/worker`).
- ISSUE-046 — API key scopes (`auth.RequireScope` per endpoint + `tenant.Tenant.HasScope`).
- ISSUE-056 — Beta release checklist (`07-operations/beta-release-checklist.md` med sign-off-process).
- ISSUE-047 — CLI för decision debug (`cmd/routerctl`).
- ISSUE-048 — SDK metadata helpers (`internal/sdk` + `examples/sdk-metadata`).
- ISSUE-049 — Spend simulator (`spend.Simulator`: baseline premium, besparing, riskjusterad besparing).
- ISSUE-050 — CI-integration för evals (dedikerade eval/policy-steg + `cmd/eval-report` artifact).
- ISSUE-051 — Budget caps (`internal/budget`: per tenant/projekt, warning, block/downgrade).
- ISSUE-052 — Route decision cache (`internal/decisioncache`: versionerad nyckel, endast låg-risk).
- ISSUE-060 — Global conservative mode (engine feature flag; osäkra tasks → ≥ balanced).
- ISSUE-053 — Model-specific prompt adapter skeleton (`internal/provider`: disabled-by-default system-role prompt mutator).
- ISSUE-054 — A/B policy simulation (`engine.DecisionComparison`, offline eval policy comparison, comparison artifacts).
- ISSUE-055 — Shadow routing (opt-in shadow policy decision, eventlog comparison payload, dashboard diff).
- ISSUE-057 — RBAC skeleton (`tenant.Role`, role middleware, admin-only dashboard).
- ISSUE-058 — Trained lightweight classifier experiment (offline-only dataset/training/baseline comparison; no production rollout).
- ISSUE-059 — Cost-quality frontier report (eval report frontier, deterministic recommendations).
- ISSUE-063 — Policy-gated context pipeline activation (`route.force.context_pipeline`, runtime policy cache, server-side activation).

Inga issues är markerade `needs-triage`, `needs-info`, `ready-for-human` eller `wontfix` efter denna pass.

## Spec detail pass 2026-05-19

High-risk open issues with expanded implementation contracts, acceptance criteria, verification notes, dependencies and non-goals:

- ISSUE-014 — `JobDescriptor` contract.
- ISSUE-015 — Fast token estimator.
- ISSUE-016 — Code-signal feature extraction.
- ISSUE-017 — Task classification rules.
- ISSUE-018 — Risk classification rules.
- ISSUE-020 — Policy DSL v1.
- ISSUE-021 — Policy parser and validation.
- ISSUE-022 — Compiled policy cache.
- ISSUE-025 — Candidate filtering.
- ISSUE-027 — Fallback planning.

## Rekommenderad prioritet

P0:

- ISSUE-001 till ISSUE-036.

P1:

- ISSUE-037 till ISSUE-056.

P2:

- ISSUE-057 och framåt.

## Tillagda via triage (post-sprint-1)

- ISSUE-061 — Rebrand `tokenix` → `tokenizer` (module path + product name). `type: refactor`, `state: done`, klar 2026-05-19.
- ISSUE-062 — Context-processor pipeline (interface only). `type: design`, `state: done`, klar 2026-05-19. Designval landade i ADR-0013.
- ISSUE-063 — Policy-gated context pipeline activation. `type: backend`, `state: done`, klar 2026-06-12. Tracks ADR-0013 tenant-policy opt-in before real processors ship.

## Labelstandard

- `category: enhancement|bug`
- `state: needs-triage|needs-info|ready-for-agent|ready-for-human|wontfix|done`
- `epic: EPIC-XX`
- `priority: P0|P1|P2`
- `type: backend|frontend|infra|security|data|product|test|refactor|design`
- `sprint: 00..08`

## Arbetsflöde

1. Skapa branch från issue.
2. Implementera.
3. Lägg till test.
4. Uppdatera dokumentation om kontrakt ändras.
5. Kontrollera latency om issue påverkar fast path.
6. Merge efter review.
