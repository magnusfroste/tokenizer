# ISSUE-058: Implementera trained lightweight classifier experiment

## Labels

- `epic: EPIC-03`
- `priority: P2`
- `type: data`
- `sprint: 08`

## Mål

Implementera implementera trained lightweight classifier experiment som del av model-router.

## Bakgrund

Detta issue stödjer målet att bygga en låg-latency prompt-router som kan välja modell automatiskt, förklarbart och säkert.

## Acceptanskriterier

- Dataset definierat.
- Baseline jämförs med rules.
- Ingen production rollout utan ADR.

## Tekniska noter

- Bevara fast path latencybudget om ändringen påverkar routing.
- Lägg till strukturerad logging där det är relevant.
- Lägg till eller uppdatera tester.

## Klar när

- Acceptanskriterierna är uppfyllda.
- Tester passerar.
- Dokumentation eller kontrakt är uppdaterade vid behov.
