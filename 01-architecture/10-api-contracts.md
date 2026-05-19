# API-kontrakt

## OpenAI-kompatibel endpoint

### POST `/v1/chat/completions`

Request liknar OpenAI chat completions.

```json
{
  "model": "auto",
  "messages": [
    {"role": "system", "content": "You are a coding assistant."},
    {"role": "user", "content": "Fix this TypeScript error..."}
  ],
  "temperature": 0.2,
  "stream": true,
  "metadata": {
    "project": "checkout-api",
    "task_type": "code_debugging",
    "files_touched": ["src/payments/checkout.ts"],
    "risk": "high"
  }
}
```

### Headers

```text
Authorization: Bearer router_api_key
x-router-mode: auto | cheap | balanced | premium | disabled
x-router-max-cost-usd: 0.25
x-router-explain: true
x-router-project: checkout-api
```

### Response headers

```text
x-router-request-id: req_123
x-router-selected-model: premium-reasoning
x-router-selected-provider: provider_a
x-router-policy-version: pv_2026_05_19
x-router-route-class: high_risk_code
x-router-overhead-ms: 47
```

## Router decision endpoint

### POST `/router/decision`

Används för debugging, simulation och SDK.

```json
{
  "messages": [...],
  "metadata": {
    "task_type": "simple_code_edit",
    "risk": "medium"
  },
  "dry_run": true
}
```

Response:

```json
{
  "selected_model": "balanced-coder",
  "selected_provider": "provider_a",
  "fallbacks": ["premium-reasoning"],
  "requires_verifier": false,
  "estimated_cost_usd": 0.012,
  "explanation": [
    "Task classified as simple_code_edit",
    "Balanced tier meets capability requirements",
    "Provider health is good"
  ]
}
```

## Outcome feedback

### POST `/router/outcomes`

```json
{
  "request_id": "req_123",
  "outcome_type": "accepted",
  "value": true,
  "source": "cli",
  "metadata": {
    "tests_passed": true,
    "user_edited_response": false
  }
}
```

## Policy management

### GET `/router/policies/current`

Returnerar aktiv policy.

### POST `/router/policies/simulate`

Kör testfall mot policy utan att aktivera.

### POST `/router/policies/activate`

Aktiverar policyversion.

## Model registry

### GET `/router/models`

Returnerar modeller som tenant får använda.

### GET `/router/models/health`

Returnerar aktuell health score.

## Health

### GET `/healthz`

Process lever.

### GET `/readyz`

Process är redo och har laddat policy/registry.

### GET `/metrics`

Prometheus metrics.
