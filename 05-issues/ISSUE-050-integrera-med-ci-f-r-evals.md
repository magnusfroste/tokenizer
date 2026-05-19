# ISSUE-050: Integrera med CI för evals

## Labels

- `epic: EPIC-08`
- `priority: P1`
- `type: infra`
- `sprint: 08`

## Mål

Implementera integrera med ci för evals som del av model-router.

## Bakgrund

Detta issue stödjer målet att bygga en låg-latency prompt-router som kan välja modell automatiskt, förklarbart och säkert.

## Acceptanskriterier

- Eval smoke test körs i CI.
- Policyändringar testas.
- Rapport artifact skapas.

## Tekniska noter

- Bevara fast path latencybudget om ändringen påverkar routing.
- Lägg till strukturerad logging där det är relevant.
- Lägg till eller uppdatera tester.

## Klar när

- Acceptanskriterierna är uppfyllda.
- Tester passerar.
- Dokumentation eller kontrakt är uppdaterade vid behov.
