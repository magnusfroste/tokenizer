# ISSUE-054: Implementera A/B policy simulation

## Labels

- `epic: EPIC-08`
- `priority: P2`
- `type: backend`
- `sprint: 08`

## Mål

Implementera implementera a/b policy simulation som del av model-router.

## Bakgrund

Detta issue stödjer målet att bygga en låg-latency prompt-router som kan välja modell automatiskt, förklarbart och säkert.

## Acceptanskriterier

- Två policies kan jämföras offline.
- Cost/route diff rapporteras.
- Ingen produktionspåverkan.

## Tekniska noter

- Bevara fast path latencybudget om ändringen påverkar routing.
- Lägg till strukturerad logging där det är relevant.
- Lägg till eller uppdatera tester.

## Klar när

- Acceptanskriterierna är uppfyllda.
- Tester passerar.
- Dokumentation eller kontrakt är uppdaterade vid behov.
