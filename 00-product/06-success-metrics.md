# Success metrics

## North Star Metric

**Cost per successful task**.

Detta är bättre än ren tokenkostnad eftersom billig modell inte är värdefull om den producerar misslyckade svar.

## Primära produktmått

| Mått | Definition | Mål för beta |
|---|---|---|
| Routing overhead p95 | Tid från request in till provider request out | < 100 ms |
| Routing overhead p99 | Samma, p99 | < 250 ms |
| Cost reduction | Jämfört med premium på alla requests | > 30 % |
| Escalation precision | Andel high-risk tasks som väljer säkrare modell | > 95 % |
| Fallback success | Fallback som ger användbar response | > 90 % |
| Decision log coverage | Requests med komplett routinglogg | > 90 % |

## Kvalitetsmått

- Success rate per taskklass.
- Human acceptance rate.
- Test pass rate för kodtasks.
- Retry count per request.
- Verifier failure rate.
- Andel requests som eskaleras.
- Andel requests där användaren manuellt overridear routern.

## Kostnadsmått

- Spend per tenant.
- Spend per provider.
- Spend per modell.
- Spend per tasktyp.
- Estimerad besparing mot baseline.
- Kostnad per accepterad response.

## Latencymått

- Router ingress latency.
- Feature extraction latency.
- Policy evaluation latency.
- Provider selection latency.
- Provider first-token latency.
- Provider total latency.
- Streaming start delay.

## Säkerhetsmått

- Antal blockerade requests.
- Antal secret masking events.
- Antal policy violations.
- Provider denylist hits.
- Audit log completeness.
