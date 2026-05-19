# Eval case template

```yaml
id: eval_001
name: example task
task_type: simple_code_edit
risk_level: medium
prompt: |
  Put prompt here.
expected_route:
  tier: balanced
quality_check:
  type: rule_or_judge
  criteria:
    - follows instruction
    - no hallucinated API
```
