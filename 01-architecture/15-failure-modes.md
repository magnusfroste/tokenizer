# Failure modes

## Provider timeout

### Symptom

- Provider svarar inte inom timeout.
- First token latency ökar.

### Hantering

- Avbryt attempt.
- Välj fallback om före första token.
- Sänk health score.
- Logga `provider_timeout`.

## Provider rate limit

### Symptom

- 429 eller quota error.

### Hantering

- Fallback till annan provider.
- Markera rate limit i health cache.
- Respektera retry-after om tillgängligt.

## Policy reload failure

### Symptom

- Ny policy kan inte kompileras.

### Hantering

- Behåll senaste fungerande policy.
- Larma.
- Blockera aktivering.

## Modell saknar capability

### Symptom

- Request kräver tool calls eller JSON schema men vald modell stöder inte det.

### Hantering

- Capabilityfilter ska förhindra detta.
- Om det ändå händer: fallback och incidentlogg.

## Event queue backlog

### Symptom

- Loggar släpar.
- Cost dashboard blir stale.

### Hantering

- API får fortsätta om kritisk audit inte krävs.
- Skala workers.
- Sätt backpressure om backlog når risknivå.

## Redis nere

### Symptom

- Health/cache/budget påverkas.

### Hantering

- Använd in-memory fallback för registry/policy.
- Budget kan gå i conservative mode.
- Health antas neutral eller degraded enligt policy.

## Postgres nere

### Symptom

- Auth lookup, policy fetch och logging kan påverkas.

### Hantering

- Fortsätt endast för cached keys om policy tillåter.
- Buffer event logs kortvarigt.
- Adminändringar stoppas.

## Router bug väljer fel modell

### Symptom

- Ökad rejection.
- Ökad under-routing.
- Policy violation.

### Hantering

- Feature flag rollback.
- Aktivera konservativ global policy.
- Kör eval regression.
- Postmortem.

## Streaming bryts

### Symptom

- Klient får partial response.

### Hantering

- Logga stream interruption.
- Fallback endast om klient stödjer restart.
- Annars returnera korrekt error.
