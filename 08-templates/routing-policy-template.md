# Routing policy template

```yaml
version: pv_YYYY_MM_DD
metadata:
  owner: platform
  description: Default policy
settings:
  default_tier: balanced
  conservative_unknowns: true
rules:
  - id: example_rule
    when:
      task_type: trivial_git
    route:
      tier: cheap
      verifier: false
```
