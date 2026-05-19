# Routing policy reference

## Policystruktur

```yaml
version: pv_YYYY_MM_DD
metadata:
  owner: platform
  description: Default routing policy
settings:
  default_tier: balanced
  conservative_unknowns: true
  max_router_overhead_ms: 100
rules: []
```

## Matchvillkor

Stöd följande villkor i MVP:

- `task_type`
- `risk_level`
- `project`
- `tenant`
- `prompt_tokens_gt`
- `prompt_tokens_lt`
- `contains_any`
- `any_file_matches`
- `requires_tool_use`
- `requires_json_schema`
- `sensitivity`
- `router_mode`

## Route actions

- `tier`
- `model`
- `provider`
- `fallback_tier`
- `fallback_models`
- `verifier`
- `block`
- `max_cost_usd`
- `timeout_ms`
- `retention`

## Exempel

```yaml
rules:
  - id: cheap_for_commit_messages
    when:
      task_type: trivial_git
    route:
      tier: cheap
      max_cost_usd: 0.002

  - id: premium_for_security
    when:
      task_type: security_review
    route:
      tier: premium
      verifier: true

  - id: long_context_model
    when:
      prompt_tokens_gt: 100000
    route:
      require_capability: long_context
```

## Prioritet

Regler körs i följande grupper:

1. Block.
2. Force.
3. Constraints.
4. Hints.
5. Defaults.

## Policytest

Varje policy ska ha testfall.

```yaml
case: auth file should be premium
input:
  task_type: simple_code_edit
  files_touched:
    - src/auth/session.ts
expected:
  tier: premium
  verifier: true
  matched_rules:
    - auth_requires_premium
```
