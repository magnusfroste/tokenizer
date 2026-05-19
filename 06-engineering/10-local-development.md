# Local development

## Lokal miljö

Komponenter:

- Router API.
- Worker.
- Postgres.
- Redis.
- Mock provider.

## Föreslagen startsekvens

```bash
cp .env.example .env
docker compose up -d postgres redis mock-provider
make migrate
make seed
make dev
```

## Seed-data

Seed bör skapa:

- En tenant.
- Ett projekt.
- En API key.
- Tre modellprofiler: cheap, balanced, premium.
- Default policy.

## Lokal testrequest

```bash
curl -X POST http://localhost:8080/v1/chat/completions \
  -H "Authorization: Bearer local_router_key" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "auto",
    "messages": [{"role": "user", "content": "Write a commit message for this diff"}],
    "metadata": {"task_type": "trivial_git", "risk": "low"}
  }'
```

## Debugging

Använd `/router/decision` för dry-run innan du skickar till provider.

```bash
curl -X POST http://localhost:8080/router/decision \
  -H "Authorization: Bearer local_router_key" \
  -d @examples/debug-request.json
```

## Lokal policytest

```bash
router policy test --policy policies/default.yaml --cases tests/policy-cases.yaml
```
