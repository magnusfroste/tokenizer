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

- ISSUE-025 till ISSUE-060.
- ISSUE-063 — Policy-gated context pipeline activation.

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
- ISSUE-063 — Policy-gated context pipeline activation. `type: backend`, `state: ready-for-agent`. Tracks ADR-0013 tenant-policy opt-in before real processors ship.

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
