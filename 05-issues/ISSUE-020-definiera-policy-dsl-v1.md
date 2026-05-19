# ISSUE-020: Definiera policy DSL v1

## Labels

- `epic: EPIC-04`
- `priority: P0`
- `type: backend`
- `sprint: 04`

## Mål

Implementera definiera policy dsl v1 som del av model-router.

## Bakgrund

Detta issue stödjer målet att bygga en låg-latency prompt-router som kan välja modell automatiskt, förklarbart och säkert.

## Acceptanskriterier

- YAML/JSON-format dokumenterat.
- Block/force/constraints stöds.
- Exempelpolicy finns.

## Tekniska noter

- Bevara fast path latencybudget om ändringen påverkar routing.
- Lägg till strukturerad logging där det är relevant.
- Lägg till eller uppdatera tester.

## Klar när

- Acceptanskriterierna är uppfyllda.
- Tester passerar.
- Dokumentation eller kontrakt är uppdaterade vid behov.
