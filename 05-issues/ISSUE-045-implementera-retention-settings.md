# ISSUE-045: Implementera retention settings

## Labels
- `epic: EPIC-09`
- `priority: P1`
- `type: data`
- `sprint: 08`
- `category: enhancement`
- `state: ready-for-agent`

## Mål

Implementera implementera retention settings som del av model-router.

## Bakgrund

Detta issue stödjer målet att bygga en låg-latency prompt-router som kan välja modell automatiskt, förklarbart och säkert.

## Acceptanskriterier

- Promptlogging kan stängas av.
- Retention per tenant.
- Cleanup job finns.

## Tekniska noter

- Bevara fast path latencybudget om ändringen påverkar routing.
- Lägg till strukturerad logging där det är relevant.
- Lägg till eller uppdatera tester.

## Klar när

- Acceptanskriterierna är uppfyllda.
- Tester passerar.
- Dokumentation eller kontrakt är uppdaterade vid behov.
