# ISSUE-008: Skapa model registry schema

## Labels

- `epic: EPIC-02`
- `priority: P0`
- `type: data`
- `sprint: 02`

## Mål

Implementera skapa model registry schema som del av model-router.

## Bakgrund

Detta issue stödjer målet att bygga en låg-latency prompt-router som kan välja modell automatiskt, förklarbart och säkert.

## Acceptanskriterier

- Models och providers kan lagras.
- Capabilities och cost metadata finns.
- Registry version kan anges.

## Tekniska noter

- Bevara fast path latencybudget om ändringen påverkar routing.
- Lägg till strukturerad logging där det är relevant.
- Lägg till eller uppdatera tester.

## Klar när

- Acceptanskriterierna är uppfyllda.
- Tester passerar.
- Dokumentation eller kontrakt är uppdaterade vid behov.
