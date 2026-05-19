# ISSUE-044: Implementera audit log

## Labels

- `epic: EPIC-09`
- `priority: P1`
- `type: security`
- `sprint: 08`

## Mål

Implementera implementera audit log som del av model-router.

## Bakgrund

Detta issue stödjer målet att bygga en låg-latency prompt-router som kan välja modell automatiskt, förklarbart och säkert.

## Acceptanskriterier

- Policy changes loggas.
- API key changes loggas.
- Blockerade requests loggas.

## Tekniska noter

- Bevara fast path latencybudget om ändringen påverkar routing.
- Lägg till strukturerad logging där det är relevant.
- Lägg till eller uppdatera tester.

## Klar när

- Acceptanskriterierna är uppfyllda.
- Tester passerar.
- Dokumentation eller kontrakt är uppdaterade vid behov.
