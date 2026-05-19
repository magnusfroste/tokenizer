# ISSUE-025: Implementera candidate filtering

## Labels
- `epic: EPIC-05`
- `priority: P0`
- `type: backend`
- `sprint: 05`
- `category: enhancement`
- `state: ready-for-agent`

## Mål

Implementera implementera candidate filtering som del av model-router.

## Bakgrund

Detta issue stödjer målet att bygga en låg-latency prompt-router som kan välja modell automatiskt, förklarbart och säkert.

## Acceptanskriterier

- Capabilities filtrerar kandidater.
- Policy constraints appliceras.
- Health kan exkludera provider.

## Tekniska noter

- Bevara fast path latencybudget om ändringen påverkar routing.
- Lägg till strukturerad logging där det är relevant.
- Lägg till eller uppdatera tester.

## Klar när

- Acceptanskriterierna är uppfyllda.
- Tester passerar.
- Dokumentation eller kontrakt är uppdaterade vid behov.
