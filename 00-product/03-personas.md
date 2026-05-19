# Personas

## Persona 1: Solo-utvecklare

### Behov

- Vill använda AI ofta utan att bränna dyra tokens i onödan.
- Vill slippa byta modell manuellt.
- Vill förstå ungefär hur mycket en session kostar.

### Smärtpunkter

- Kör premium-modell på enkla kommandon.
- Glömmer att byta tillbaka till billig modell.
- Har ingen tydlig bild av vilka prompts som kostar mest.

### Produktvärde

- Automatisk modellrouting.
- Enkel spend-rapport.
- Konfigurerbar maxbudget per dag/projekt.

## Persona 2: AI-kodagentbyggare

### Behov

- Vill bygga en agent som kan använda olika modeller utan hårdkodade beslut.
- Vill kunna routea per steg i agentloopen.
- Vill mäta success rate per modell och tasktyp.

### Smärtpunkter

- En modell är inte bäst på allt.
- Agenten blir dyr när premium används för planering, commitmeddelanden och triviala edits.
- Fallback och retries blir snabbt komplexa.

### Produktvärde

- OpenAI-kompatibel proxy.
- Structured job descriptors.
- Fallback, escalation och verifiering.

## Persona 3: Team lead

### Behov

- Vill reducera AI-kostnader i teamet.
- Vill undvika att känslig kod går till fel provider.
- Vill ha audit trail.

### Smärtpunkter

- Ingen vet varför en viss modell användes.
- Kostnad sprids över många personliga API-nycklar.
- Säkerhetspolicies är svåra att upprätthålla.

### Produktvärde

- Central policy.
- Per-team budgets.
- Provider allow/deny.
- Beslutslogg och kostnadsrapport.

## Persona 4: Plattformsteam

### Behov

- Vill drifta en central AI-gateway.
- Vill integrera med observability, SSO och secrets.
- Vill ha SLO för proxy-lagret.

### Smärtpunkter

- Olika team använder olika providers.
- Svårt att incidenthantera modell- och providerproblem.
- Saknar standardiserad telemetry.

### Produktvärde

- OpenTelemetry.
- Health scoring.
- Multi-tenant API key management.
- Runbooks och policy-DLS.
