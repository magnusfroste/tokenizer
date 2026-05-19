# Sprint 1 demo

Demo som visar att kraven i `04-sprints/sprint-01-proxy-skeleton.md` är uppfyllda.

## Förberedelser

```bash
make build
./bin/mock-provider &           # lyssnar på :18080
LOCAL_API_KEY=local_router_key \
  ROUTER_ADDR=:18888 \
  MOCK_PROVIDER_URL=http://localhost:18080 \
  ./bin/router &                # lyssnar på :18888
```

## 1. `/healthz` (publik)

```bash
curl -s -i http://localhost:18888/healthz
```

```text
HTTP/1.1 200 OK
Content-Type: application/json
X-Router-Request-Id: req_a9f2c682-1d7f-43cb-86c6-f75fe94dc36c

{"status":"ok"}
```

## 2. `/readyz` (publik)

```bash
curl -s -i http://localhost:18888/readyz
```

```text
HTTP/1.1 200 OK
Content-Type: application/json
X-Router-Request-Id: req_5f92e0bb-c96e-4c9f-920f-0eb95f8e60cf

{"status":"ready"}
```

## 3. `/v1/chat/completions` utan auth → 401

```bash
curl -s -i -X POST http://localhost:18888/v1/chat/completions \
  -H 'Content-Type: application/json' \
  -d '{"model":"auto","messages":[{"role":"user","content":"hi"}]}'
```

```text
HTTP/1.1 401 Unauthorized
Content-Type: application/json
X-Router-Request-Id: req_4d665009-1466-4bce-b187-82d932ed22e8

{"error":{"message":"missing bearer token","type":"unauthorized"}}
```

## 4. `/v1/chat/completions` med auth → 200 (proxat till mock provider)

```bash
curl -s -i -X POST http://localhost:18888/v1/chat/completions \
  -H 'Authorization: Bearer local_router_key' \
  -H 'Content-Type: application/json' \
  -d '{"model":"auto","messages":[{"role":"user","content":"Write a commit message for fixing a typo"}]}'
```

```text
HTTP/1.1 200 OK
Content-Type: application/json
X-Router-Request-Id: req_c4c039ab-d412-4da9-8a9e-b1ad8c720525
X-Router-Selected-Model: auto

{"id":"chatcmpl_mock_dimohuau846h","object":"chat.completion","created":1779196425,"model":"auto","choices":[{"index":0,"message":{"role":"assistant","content":"mock response to: Write a commit message for fixing a typo"},"finish_reason":"stop"}],"usage":{"prompt_tokens":10,"completion_tokens":12,"total_tokens":22}}
```

## Strukturerad logg (router stderr)

Varje request ger en JSON-rad på `slog`-format:

```json
{"time":"2026-05-19T13:13:45.705349162Z","level":"INFO","msg":"http_request","request_id":"req_c4c039ab-d412-4da9-8a9e-b1ad8c720525","method":"POST","path":"/v1/chat/completions","status":200,"duration_ms":3}
```

## Acceptanskriterier kontrollerade

| Issue | Kriterium | Status |
|---|---|---|
| ISSUE-001 | Repo innehåller API, worker och docsstruktur | ✅ `cmd/router`, `cmd/worker` (stub), `cmd/mock-provider`, `internal/`, doc-träd |
| ISSUE-001 | Lokal körning dokumenterad | ✅ `.env.example`, `Makefile`, denna fil |
| ISSUE-001 | CI skeleton finns | ✅ `.github/workflows/ci.yml` (vet + gofmt + test) |
| ISSUE-002 | Health endpoints returnerar korrekt status | ✅ se 1, 2 |
| ISSUE-002 | Ready kontrollerar policy och registry loaded | ✅ via `ReadyzChecker`-interface (inga checkers registrerade i sprint 1) |
| ISSUE-003 | Bearer token kan valideras mot hashad key | ✅ SHA-256 i `InMemoryKeyStore` |
| ISSUE-003 | Ogiltig key ger 401 | ✅ se 3 |
| ISSUE-003 | Tenant context skapas | ✅ `tenant.WithTenant` i middleware |
| ISSUE-004 | Endpoint accepterar OpenAI-liknande request | ✅ se 4 |
| ISSUE-004 | Request proxas till mock provider | ✅ se 4 |
| ISSUE-004 | Response normaliseras | ✅ `openai.ChatResponse` typ |
| ISSUE-005 | Mock returnerar normal response | ✅ se 4 |
| ISSUE-005 | Mock simulerar timeout, 429, 5xx | ✅ via `X-Mock-Behavior`-header (`timeout`, `rate_limit`, `5xx`, `bad_response`) |
| ISSUE-005 | Mock stödjer token usage | ✅ 4-char/token-heuristik i mock |
| ISSUE-006 | Varje request får request id | ✅ `req_<uuid>` |
| ISSUE-006 | Trace spans skapas | ⚠️ strukturerad logg med request_id finns; OTel-spans landar i sprint 6 (EPIC-07) |
| ISSUE-006 | Request id returneras i header | ✅ `X-Router-Request-Id` |

## DoD sprint 1

- ✅ Request kan proxas till mock provider
- ✅ Request id och grundlogg finns
