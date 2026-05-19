# ISSUE-019: Mät feature extraction latency

## Labels

- `epic: EPIC-03`
- `priority: P0`
- `type: test`
- `sprint: 03`

## Mål

Implementera mät feature extraction latency som del av model-router.

## Bakgrund

Detta issue stödjer målet att bygga en låg-latency prompt-router som kan välja modell automatiskt, förklarbart och säkert.

## Acceptanskriterier

- Benchmark finns.
- p95 rapporteras.
- Failar om budget överskrids.

## Tekniska noter

- Bevara fast path latencybudget om ändringen påverkar routing.
- Lägg till strukturerad logging där det är relevant.
- Lägg till eller uppdatera tester.

## Klar när

- Acceptanskriterierna är uppfyllda.
- Tester passerar.
- Dokumentation eller kontrakt är uppdaterade vid behov.
