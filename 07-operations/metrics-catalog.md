# Metrics catalog

## Counter metrics

- `router_requests_total`
- `router_decisions_total`
- `router_fallbacks_total`
- `router_provider_errors_total`
- `router_policy_blocks_total`
- `router_outcomes_total`

## Histogram metrics

- `router_overhead_ms`
- `router_feature_extraction_ms`
- `router_policy_eval_ms`
- `router_decision_ms`
- `provider_first_token_ms`
- `provider_total_latency_ms`

## Gauge metrics

- `event_queue_backlog`
- `provider_health_score`
- `tenant_budget_remaining_usd`

## Labels

Använd labels försiktigt för cardinality.

Rekommenderade:

- `tenant_plan`
- `model_tier`
- `provider`
- `route_class`
- `status`

Undvik:

- Raw tenant id i Prometheus om många tenants.
- Request id.
- Promptinnehåll.
