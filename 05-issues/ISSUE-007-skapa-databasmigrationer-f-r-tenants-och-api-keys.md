# ISSUE-007: Skapa databasmigrationer för tenants och api_keys

## Labels

- `epic: EPIC-09`
- `priority: P0`
- `type: data`
- `sprint: 01`

## Mål

Implementera skapa databasmigrationer för tenants och api_keys som del av model-router.

## Bakgrund

Detta issue stödjer målet att bygga en låg-latency prompt-router som kan välja modell automatiskt, förklarbart och säkert.

## Acceptanskriterier

- Tabeller skapas.
- Keys lagras hashade.
- Seed för lokal tenant finns.

## Tekniska noter

- Bevara fast path latencybudget om ändringen påverkar routing.
- Lägg till strukturerad logging där det är relevant.
- Lägg till eller uppdatera tester.

## Klar när

- Acceptanskriterierna är uppfyllda.
- Tester passerar.
- Dokumentation eller kontrakt är uppdaterade vid behov.
