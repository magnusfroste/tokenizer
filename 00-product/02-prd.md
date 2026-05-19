# PRD: Låg-latency model router

## Sammanfattning

Bygg en tjänst som exponerar OpenAI-kompatibla endpoints och routar inkommande prompter till optimal modell/provider. Beslutet ska baseras på tasktyp, risk, kontextbehov, budget, latencykrav, tenantpolicy, modellkapabilitet och modellhälsa.

## Problem

AI-användare betalar för mycket när premium-modeller används på enkla jobb. Samtidigt ökar risken när billiga modeller används på uppgifter som kräver djup reasoning, säkerhetstänk eller stor kontext.

## Mål

### Funktionella mål

- OpenAI-kompatibel proxy för chat completions.
- Snabb prompt-/taskklassificering.
- Policybaserat modellval.
- Fallback och eskalering.
- Kostnads- och latencymätning.
- Beslutsförklaring per request.
- Per-tenant modellallowlist och budget.
- Evals för att jämföra routingstrategier.

### Icke-funktionella mål

- Routingoverhead p95 under 100 ms för fast path.
- Routingoverhead p99 under 250 ms för fast path.
- Systemet ska kunna hantera provider-timeouts utan att tappa requesten om fallback finns.
- Policyuppdatering ska kunna göras utan full redeploy.
- Loggar ska kunna användas för kostnadsanalys och modelljämförelse.

## Scope för MVP

Ingår:

- API key authentication.
- `/v1/chat/completions` med streaming och non-streaming.
- Minst tre modellprofiler: cheap, balanced, premium.
- Modellregistry med provider, kostnad, latencyprofil och capabilities.
- Routingpolicy i YAML/JSON.
- Klassificering via deterministiska regler och snabb feature extraction.
- Eventlogg för request, decision, attempt och outcome.
- Enkel dashboard eller rapport för spend/latency.

Ingår inte i MVP:

- Full enterprise compliance.
- Automatisk modellträning.
- Avancerad multi-agent orchestration.
- Fullständig promptoptimering per modell.
- SLA-garantier.

## Användarresor

### 1. Utvecklare vill spara tokens

1. Användaren byter `OPENAI_BASE_URL` till router-endpoint.
2. Routern identifierar triviala jobb.
3. Billiga modeller används för triviala jobb.
4. Dyra modeller används endast vid hög risk eller komplexitet.
5. Dashboard visar sparad kostnad och routes.

### 2. Team vill undvika fel modell för känslig kod

1. Admin skapar policy: auth, payments, security och migrations kräver premium.
2. Agent skickar prompt med filkontext eller metadata.
3. Routern klassificerar risk.
4. Premium-modell eller verifier-cascade används.
5. Beslutet loggas med förklaring.

### 3. Provider har driftstörning

1. Primär provider får timeout.
2. Routern kontrollerar fallbackpolicy.
3. Request skickas till sekundär modell.
4. Logg markerar fallbackorsak.
5. Health-score sänks temporärt för primär provider.

## Prioriterade krav

| Prioritet | Krav |
|---|---|
| P0 | OpenAI-kompatibel proxy |
| P0 | Routingbeslut under definierad latencybudget |
| P0 | Modellregistry |
| P0 | Policy engine |
| P0 | Provider fallback |
| P0 | Request/decision logging |
| P1 | Dashboard |
| P1 | Outcome feedback |
| P1 | Evals |
| P1 | Per-project policies |
| P2 | Model-specific prompt adapters |
| P2 | Automatisk policyförbättring |

## Framgångsmått

- Kostnad per lyckad task minskar med minst 30 procent i beta.
- Fast path routing overhead p95 under 100 ms.
- Mindre än 1 procent routingfel där policy borde ha valt säkrare modell.
- Minst 90 procent av requestar har komplett decision log.
- Fallback fungerar för minst två providers.

## Risker

- Routerfel kan orsaka sämre kod än manuell modellstyrning.
- Provider-API:er skiljer sig åt, särskilt streaming och tool calls.
- Promptklassificering kan bli för långsam om den använder LLM för ofta.
- Kostnadsbesparing kan äta upps av extra verifiering.
- Team kan sakna tillit om beslut inte går att förklara.
