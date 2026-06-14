# Model Router för låg-latency prompt-routing

Detta paket är ett byggunderlag för en tjänst som routar varje prompt till rätt modell med låg extra latency. Tjänsten fungerar som en OpenAI-kompatibel proxy/gateway framför flera modellproviders och väljer modell baserat på uppgift, risk, kostnad, latency, policy, modellhälsa och historiska outcome-signaler.

Arbetsnamn i dokumenten: **model-router**.

## Kom igång (quickstart)

Kräver Go 1.22+.

### Snabbast — lokalt, utan databas (in-memory)

Routern kör hela fast-path-vägen (auth, classifier, policy, routing, fallback)
med in-memory-state, så du behöver varken Postgres eller Redis för att prova den.

```bash
make build

# Terminal 1: mock-provider (svarar som en OpenAI-kompatibel modell)
MOCK_PROVIDER_ADDR=:18080 ./bin/mock-provider

# Terminal 2: routern
LOCAL_API_KEY=local_router_key ./bin/router
```

Testa den:

```bash
# Chat completion (model: auto → routern väljer modell)
curl -s -X POST http://localhost:8080/v1/chat/completions \
  -H "Authorization: Bearer local_router_key" \
  -d '{"model":"auto","messages":[{"role":"user","content":"write a git commit message"}]}'

# Routingbeslut utan provideranrop (dry-run, med förklaringar)
./bin/routerctl -url http://localhost:8080 -key local_router_key \
  -message "review this auth change for security"
```

Andra endpoints: `GET /healthz`, `GET /readyz`, `GET /metrics`,
`GET /router/dashboard`, `POST /router/decision`.

### Fullt — med Postgres/Redis

```bash
cp .env.example .env
docker compose up -d postgres redis mock-provider
make migrate   # applicerar alla db/migrations/*.sql i ordning
make seed
make dev       # go run ./cmd/router
```

### Tester och evals

```bash
make test          # unit + integration (race)
make test-eval     # eval smoke
make test-policy   # policy golden-cases
make eval-report   # skriver eval-report/report.{json,txt}
make lint
```

## Mål

Bygg en produktionsduglig tjänst som kan:

1. Ta emot requests via OpenAI-kompatibla endpoints.
2. Extrahera ett strukturerat `JobDescriptor` på millisekunder.
3. Välja modell via policy + scoring utan att routingen blir flaskhals.
4. Skicka requesten till vald modell/provider.
5. Falla tillbaka eller eskalera vid timeout, låg confidence eller verifieringsfel.
6. Logga kostnad, latency, route-beslut, modellutfall och användarfeedback.
7. Förbättra routingpolicy över tid med evals och verklig outcome-data.

## Viktig produktprincip

Detta är inte bara en “billigaste modell”-proxy. Routern ska optimera för **riskjusterad nytta**:

```text
rätt kvalitet + rätt kostnad + rätt latency + rätt policy + rätt fallback
```

## Stack (implementerad)

Den ursprungliga MVP-skissen vägde FastAPI/LiteLLM mot Go; beslutet blev **Go**
(se `DECISION_LOG.md`, 2026-05-19). Nuvarande implementation:

- API/proxy: **Go 1.22**, stdlib `net/http` (1.22 `ServeMux`), `log/slog`.
- Providerlager: egna adaptrar (`internal/provider`); `mock-provider` för dev.
- Databas: Postgres (schema i `db/migrations/`).
- Snabb state/cache: in-memory idag (API-key/policy/health/decision-cache);
  Redis enligt `docker-compose.yml` för full uppsättning.
- Eventlogg: async event queue → strukturerad logg + Prometheus-metrics.
- Observability: Prometheus (`/metrics`) + inbyggd dashboard.
- Auth: API keys (hashade) med tenant/project-scope + RBAC.
- Deployment: Docker Compose för dev.

## Rekommenderad läsordning

1. `00-product/01-product-vision.md`
2. `00-product/02-prd.md`
3. `01-architecture/01-system-overview.md`
4. `01-architecture/03-request-lifecycle.md`
5. `01-architecture/04-routing-engine.md`
6. `01-architecture/05-low-latency-architecture.md`
7. `02-adr/ADR-0001-openai-compatible-proxy.md`
8. `03-backlog/backlog-index.md`
9. `04-sprints/sprint-index.md`
10. `05-issues/issue-index.md`

## Paketstruktur

```text
00-product/       Produktkrav, personas, mål, scope och roadmap
01-architecture/  Systemarkitektur, API, data, latency, säkerhet, drift
02-adr/           Arkitekturbeslut
03-backlog/       Epics och produktbacklog
04-sprints/       Sprintplaner
05-issues/        Implementerbara issues
06-engineering/   Tekniska referenser för routing, classifier, cache, tester
07-operations/    Runbooks, SLO, incidenthantering, releasechecklistor
08-templates/     Mallar för ADR, issue, epic, sprint, postmortem och policy
```

## Definition of Done för MVP

MVP är klar när:

- `/v1/chat/completions` kan proxya minst tre modeller via minst två providers.
- Routingbeslut lägger till mindre än 100 ms p95 overhead före modellrequest.
- Minst sex taskklasser kan routas via policy: trivial, enkel kod, svår kod, säkerhet, long-context och fallback.
- Beslutet loggas med modell, kostnad, latency, policyversion och route-förklaring.
- Provider-timeout ger fallback inom definierad latencybudget.
- Basdashboard visar spend, latency, route distribution och error rate.
- Evals kan köras lokalt och jämföra modeller på minst 50 testfall.

## Definition of Done för beta

Beta är klar när:

- Fler tenants och per-project policies fungerar.
- API keys kan skapas, roteras och begränsas.
- Modellhälsa uppdateras löpande och används i routebeslut.
- Outcome-feedback kan skickas in manuellt eller via SDK.
- Router kan förklara varför den valde en modell.
- Security baseline är implementerad: secret masking, audit log, data retention, provider allow/deny list.
- Produktionsrunbook och incidentprocess finns.
