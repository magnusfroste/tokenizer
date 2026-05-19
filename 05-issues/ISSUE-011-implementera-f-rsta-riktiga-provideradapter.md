# ISSUE-011: Implementera första riktiga provideradapter

## Labels

- `epic: EPIC-06`
- `priority: P0`
- `type: backend`
- `sprint: 02`

## Mål

Implementera implementera första riktiga provideradapter som del av model-router.

## Bakgrund

Detta issue stödjer målet att bygga en låg-latency prompt-router som kan välja modell automatiskt, förklarbart och säkert.

## Acceptanskriterier

- Adapter kan skicka chat request.
- Usage normaliseras.
- Fel normaliseras.

## Tekniska noter

- Bevara fast path latencybudget om ändringen påverkar routing.
- Lägg till strukturerad logging där det är relevant.
- Lägg till eller uppdatera tester.

## Klar när

- Acceptanskriterierna är uppfyllda.
- Tester passerar.
- Dokumentation eller kontrakt är uppdaterade vid behov.
