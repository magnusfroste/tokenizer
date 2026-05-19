# Model Router för låg-latency prompt-routing

Detta paket är ett byggunderlag för en tjänst som routar varje prompt till rätt modell med låg extra latency. Tjänsten fungerar som en OpenAI-kompatibel proxy/gateway framför flera modellproviders och väljer modell baserat på uppgift, risk, kostnad, latency, policy, modellhälsa och historiska outcome-signaler.

Arbetsnamn i dokumenten: **model-router**.

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

## Föreslagen stack för MVP

- API/proxy: FastAPI, Hono eller Go/Fiber.
- Providerlager: LiteLLM eller egen adaptermodul.
- Databas: Postgres.
- Snabb state/cache: Redis.
- Eventlogg: Postgres initialt; ClickHouse senare vid hög volym.
- Observability: OpenTelemetry, Prometheus, Grafana.
- Dashboard: Next.js eller intern adminpanel.
- Auth: API keys med tenant- och project-scope.
- Deployment: Docker Compose för dev, Kubernetes/Fly/Render/AWS ECS för prod.

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
