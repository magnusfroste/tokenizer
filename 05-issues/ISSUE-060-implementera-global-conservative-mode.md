# ISSUE-060: Implementera global conservative mode

## Labels

- `epic: EPIC-05`
- `priority: P1`
- `type: backend`
- `sprint: 08`

## Mål

Implementera implementera global conservative mode som del av model-router.

## Bakgrund

Detta issue stödjer målet att bygga en låg-latency prompt-router som kan välja modell automatiskt, förklarbart och säkert.

## Acceptanskriterier

- Feature flag finns.
- Okända tasks routeas premium/balanced enligt policy.
- Incidentrunbook refererar till flaggan.

## Tekniska noter

- Bevara fast path latencybudget om ändringen påverkar routing.
- Lägg till strukturerad logging där det är relevant.
- Lägg till eller uppdatera tester.

## Klar när

- Acceptanskriterierna är uppfyllda.
- Tester passerar.
- Dokumentation eller kontrakt är uppdaterade vid behov.
