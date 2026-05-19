# ISSUE-012: Implementera streaming skeleton

## Labels

- `epic: EPIC-06`
- `priority: P0`
- `type: backend`
- `sprint: 02`

## Mål

Implementera implementera streaming skeleton som del av model-router.

## Bakgrund

Detta issue stödjer målet att bygga en låg-latency prompt-router som kan välja modell automatiskt, förklarbart och säkert.

## Acceptanskriterier

- Streaming endpoint skickar chunks.
- First token timestamp mäts.
- Stream error loggas.

## Tekniska noter

- Bevara fast path latencybudget om ändringen påverkar routing.
- Lägg till strukturerad logging där det är relevant.
- Lägg till eller uppdatera tester.

## Klar när

- Acceptanskriterierna är uppfyllda.
- Tester passerar.
- Dokumentation eller kontrakt är uppdaterade vid behov.
