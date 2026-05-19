# ISSUE-051: Implementera budget caps

## Labels

- `epic: EPIC-09`
- `priority: P1`
- `type: backend`
- `sprint: 08`

## Mål

Implementera implementera budget caps som del av model-router.

## Bakgrund

Detta issue stödjer målet att bygga en låg-latency prompt-router som kan välja modell automatiskt, förklarbart och säkert.

## Acceptanskriterier

- Budget per tenant/project.
- Threshold warning.
- Block eller downgrade enligt policy.

## Tekniska noter

- Bevara fast path latencybudget om ändringen påverkar routing.
- Lägg till strukturerad logging där det är relevant.
- Lägg till eller uppdatera tester.

## Klar när

- Acceptanskriterierna är uppfyllda.
- Tester passerar.
- Dokumentation eller kontrakt är uppdaterade vid behov.
