# Issue index

Issues är skrivna som implementerbara tickets. Filnamn följer `ISSUE-XXX-title.md`.

## Triage snapshot 2026-05-19

Klart enligt kod- och test-evidens:

- ISSUE-001 till ISSUE-013.
- ISSUE-061 — Rebrand `tokenix` → `tokenizer`.
- ISSUE-062 — Context-processor pipeline (interface only).

Redo för agent:

- ISSUE-014 till ISSUE-060.
- ISSUE-063 — Policy-gated context pipeline activation.

Inga issues är markerade `needs-triage`, `needs-info`, `ready-for-human` eller `wontfix` efter denna pass.

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
