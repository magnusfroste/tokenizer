# ISSUE-015: Implementera snabb tokenestimering

## Labels

- `epic: EPIC-03`
- `priority: P0`
- `type: backend`
- `sprint: 03`

## Mål

Implementera implementera snabb tokenestimering som del av model-router.

## Bakgrund

Detta issue stödjer målet att bygga en låg-latency prompt-router som kan välja modell automatiskt, förklarbart och säkert.

## Acceptanskriterier

- Char-baserad approximation finns.
- Estimat returneras i descriptor.
- Latencytest finns.

## Tekniska noter

- Bevara fast path latencybudget om ändringen påverkar routing.
- Lägg till strukturerad logging där det är relevant.
- Lägg till eller uppdatera tester.

## Klar när

- Acceptanskriterierna är uppfyllda.
- Tester passerar.
- Dokumentation eller kontrakt är uppdaterade vid behov.
