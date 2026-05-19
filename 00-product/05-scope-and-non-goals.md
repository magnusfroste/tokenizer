# Scope och non-goals

## Scope

Model-router ska vara ett transparent routinglager mellan klienter/agenter och modellproviders.

Primära funktioner:

- Ta emot OpenAI-kompatibla requests.
- Klassificera prompt och metadata.
- Välja modell/provider.
- Exekvera requesten.
- Hantera streaming.
- Hantera fallback och retries.
- Logga kostnad, latency och beslut.
- Stödja policyer och budget.
- Stödja evals och feedback.

## Non-goals för MVP

- Bygga en egen foundation model.
- Ersätta full agent-runtime.
- Bygga komplett IDE.
- Försöka garantera korrekthet i modelloutput.
- Optimera alla modellproviders från dag ett.
- Implementera full enterprise compliance från start.

## Gränssnitt mot agenter

Routern ska inte behöva veta allt om agenten, men den ska kunna ta emot metadata.

Exempel på metadata:

```json
{
  "task_type": "code_debugging",
  "risk": "high",
  "repo": "checkout-service",
  "files_touched": ["src/auth.ts", "db/migrations/2026_05_19.sql"],
  "latency_preference": "balanced",
  "quality_preference": "high"
}
```

## När användaren ska kunna overridea routern

Användaren eller klienten ska kunna sätta:

- `model`: explicit modell.
- `router_mode`: `auto`, `cheap`, `balanced`, `premium`, `disabled`.
- `max_cost_usd`.
- `risk`: explicit riskklass.
- `requires_verifier`: bool.

Policy kan dock blockera vissa overrides.
