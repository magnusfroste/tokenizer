# ISSUE-048: Skapa SDK metadata helpers

## Labels

- `epic: EPIC-10`
- `priority: P1`
- `type: backend`
- `sprint: 08`

## Mål

Implementera skapa sdk metadata helpers som del av model-router.

## Bakgrund

Detta issue stödjer målet att bygga en låg-latency prompt-router som kan välja modell automatiskt, förklarbart och säkert.

## Acceptanskriterier

- SDK kan skicka project/task/risk.
- Exempel finns.
- Backward compatible.

## Tekniska noter

- Bevara fast path latencybudget om ändringen påverkar routing.
- Lägg till strukturerad logging där det är relevant.
- Lägg till eller uppdatera tester.

## Klar när

- Acceptanskriterierna är uppfyllda.
- Tester passerar.
- Dokumentation eller kontrakt är uppdaterade vid behov.
