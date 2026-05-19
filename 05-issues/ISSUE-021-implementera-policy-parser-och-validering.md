# ISSUE-021: Implementera policy parser och validering

## Labels
- `epic: EPIC-04`
- `priority: P0`
- `type: backend`
- `sprint: 04`
- `category: enhancement`
- `state: ready-for-agent`

## Mål

Implementera implementera policy parser och validering som del av model-router.

## Bakgrund

Detta issue stödjer målet att bygga en låg-latency prompt-router som kan välja modell automatiskt, förklarbart och säkert.

## Acceptanskriterier

- Ogiltig policy stoppas.
- Okända modeller upptäcks.
- Tester finns.

## Tekniska noter

- Bevara fast path latencybudget om ändringen påverkar routing.
- Lägg till strukturerad logging där det är relevant.
- Lägg till eller uppdatera tester.

## Klar när

- Acceptanskriterierna är uppfyllda.
- Tester passerar.
- Dokumentation eller kontrakt är uppdaterade vid behov.
