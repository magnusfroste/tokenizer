# Roadmap

## Fas 0: Discovery och teknisk grund

Mål:

- Fastställa målpersona och primärt arbetsflöde.
- Definiera första policyformatet.
- Välja providerabstraktion.
- Bygga minimal OpenAI-kompatibel proxy.

Leverabler:

- Arkitekturdokument.
- ADR:er.
- Första API-spec.
- Lokal devmiljö.

## Fas 1: MVP

Mål:

- Routea promptar mellan tre modellprofiler.
- Logga beslut, kostnad och latency.
- Ha deterministisk fast path.
- Stödja fallbackkedja.

Leverabler:

- `/v1/chat/completions`.
- Modellregistry.
- Policy engine.
- Feature extractor.
- Decision log.
- Enkel spend/latency dashboard.

## Fas 2: Beta

Mål:

- Multi-tenant stöd.
- API key management.
- Outcome feedback.
- Offline evals.
- Health scoring.
- Per-project policies.

Leverabler:

- Admin UI.
- Evals CLI.
- Outcome webhook/API.
- Model health monitor.
- Secrets masking.

## Fas 3: Teamprodukt

Mål:

- Team-budgetar.
- Audit och policy governance.
- Mer avancerad routing.
- Integrering med kodagenter, CI och dashboards.

Leverabler:

- RBAC.
- Policy versions.
- CI integration.
- Routing simulator.
- Model-specific prompt adapters.

## Fas 4: Avancerad router

Mål:

- Lära routing från outcome-data.
- Automatisk policyrekommendation.
- Bandit/cascade-routing.
- Quality prediction per tasktyp.

Leverabler:

- Tränad routermodell.
- A/B-testing.
- Cost-quality frontier.
- Autopilot med guardrails.
