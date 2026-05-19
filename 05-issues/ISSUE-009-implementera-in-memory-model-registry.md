# ISSUE-009: Implementera in-memory model registry

## Labels

- `epic: EPIC-02`
- `priority: P0`
- `type: backend`
- `sprint: 02`

## Mål

Implementera implementera in-memory model registry som del av model-router.

## Bakgrund

Detta issue stödjer målet att bygga en låg-latency prompt-router som kan välja modell automatiskt, förklarbart och säkert.

## Acceptanskriterier

- Registry laddas vid startup.
- Registry kan hämtas utan DB per request.
- Reload-funktion finns.

## Tekniska noter

- Bevara fast path latencybudget om ändringen påverkar routing.
- Lägg till strukturerad logging där det är relevant.
- Lägg till eller uppdatera tester.

## Klar när

- Acceptanskriterierna är uppfyllda.
- Tester passerar.
- Dokumentation eller kontrakt är uppdaterade vid behov.
