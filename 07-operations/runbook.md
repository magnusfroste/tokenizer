# Runbook

## Provider error rate ökar

1. Kontrollera dashboard för provider error rate.
2. Kontrollera om felet är rate limit, timeout eller 5xx.
3. Sätt provider health till degraded om automatiken inte gjort det.
4. Kontrollera att fallback fungerar.
5. Informera användare om påverkan vid behov.
6. Efter incident: skapa regressionstest om routing påverkades.

## Router latency p95 över 100 ms

1. Kontrollera traces för var tid spenderas.
2. Leta efter DB reads i fast path.
3. Kontrollera event queue backlog.
4. Kontrollera policy reload eller registry lock contention.
5. Skala API-noder om CPU-bound.
6. Rollbacka senaste change om latency ökade efter release.

## Fel modell väljs

1. Hämta request id.
2. Läs JobDescriptor och matched policy rules.
3. Kontrollera policyversion och registryversion.
4. Kör `/router/decision` dry-run med samma input.
5. Lägg till regressionstest.
6. Uppdatera policy eller classifier.

## Budgetincident

1. Identifiera tenant/project.
2. Kontrollera spend per modell och taskklass.
3. Kontrollera om fallback/retry loop orsakat kostnad.
4. Aktivera budget cap eller conservative downgrade för låg-risk tasks.
5. Informera admin.

## Missklassificering / osäker routing under incident

Om classifiern routar osäkra (låg-confidence) eller okända tasks för billigt under
en incident: aktivera **global conservative mode** (ISSUE-060).

1. Sätt feature-flaggan `ROUTER_CONSERVATIVE_MODE=true` (eller anropa
   `Engine.SetConservative(true)` i drift) för att höja golvet.
2. Effekt: okända/låg-confidence klassificeringar routas till minst `balanced`
   (policy- och task-forcing till `premium` gäller fortfarande — flaggan sänker
   aldrig säkerhetsnivån).
3. Verifiera via `/router/decision` att osäkra prompts nu väljer ≥ balanced.
4. Stäng av flaggan när classifiern/incidenten är åtgärdad.

## Policy reload failure

1. Kontrollera policy validator logs.
2. Behåll senaste fungerande policy.
3. Kör policy tests lokalt/staging.
4. Aktivera korrigerad policy.
5. Logga audit event.
