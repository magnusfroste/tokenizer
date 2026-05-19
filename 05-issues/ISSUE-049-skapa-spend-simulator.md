# ISSUE-049: Skapa spend simulator

## Labels

- `epic: EPIC-10`
- `priority: P1`
- `type: data`
- `sprint: 08`

## Mål

Implementera skapa spend simulator som del av model-router.

## Bakgrund

Detta issue stödjer målet att bygga en låg-latency prompt-router som kan välja modell automatiskt, förklarbart och säkert.

## Acceptanskriterier

- Simulerar baseline premium.
- Visar besparing.
- Visar riskjusterad besparing.

## Tekniska noter

- Bevara fast path latencybudget om ändringen påverkar routing.
- Lägg till strukturerad logging där det är relevant.
- Lägg till eller uppdatera tester.

## Klar när

- Acceptanskriterierna är uppfyllda.
- Tester passerar.
- Dokumentation eller kontrakt är uppdaterade vid behov.
