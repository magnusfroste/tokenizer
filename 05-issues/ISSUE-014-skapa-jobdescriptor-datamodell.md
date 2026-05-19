# ISSUE-014: Skapa JobDescriptor datamodell

## Labels

- `epic: EPIC-03`
- `priority: P0`
- `type: backend`
- `sprint: 03`

## Mål

Implementera skapa jobdescriptor datamodell som del av model-router.

## Bakgrund

Detta issue stödjer målet att bygga en låg-latency prompt-router som kan välja modell automatiskt, förklarbart och säkert.

## Acceptanskriterier

- Schema implementerat.
- Metadata mappas.
- Descriptor loggas utan prompttext.

## Tekniska noter

- Bevara fast path latencybudget om ändringen påverkar routing.
- Lägg till strukturerad logging där det är relevant.
- Lägg till eller uppdatera tester.

## Klar när

- Acceptanskriterierna är uppfyllda.
- Tester passerar.
- Dokumentation eller kontrakt är uppdaterade vid behov.
