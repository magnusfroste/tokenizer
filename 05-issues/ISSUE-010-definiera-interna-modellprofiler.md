# ISSUE-010: Definiera interna modellprofiler

## Labels

- `epic: EPIC-02`
- `priority: P0`
- `type: product`
- `sprint: 02`

## Mål

Implementera definiera interna modellprofiler som del av model-router.

## Bakgrund

Detta issue stödjer målet att bygga en låg-latency prompt-router som kan välja modell automatiskt, förklarbart och säkert.

## Acceptanskriterier

- cheap, balanced och premium definieras.
- Profil mappar till provider model id.
- Policy använder profiler.

## Tekniska noter

- Bevara fast path latencybudget om ändringen påverkar routing.
- Lägg till strukturerad logging där det är relevant.
- Lägg till eller uppdatera tester.

## Klar när

- Acceptanskriterierna är uppfyllda.
- Tester passerar.
- Dokumentation eller kontrakt är uppdaterade vid behov.
