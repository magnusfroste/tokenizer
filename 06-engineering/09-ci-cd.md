# CI/CD

## CI-pipeline

Steg:

1. Lint.
2. Unit tests.
3. Integration tests med mock provider.
4. Policy tests.
5. Eval smoke test.
6. Build image.
7. Security scan.
8. Deploy preview.

## Policy CI

Policyändringar ska testas separat:

- Syntaxvalidering.
- Kompilering.
- Golden cases.
- No blocked model references.
- No orphan providers.

## Release

Release ska innehålla:

- Router version.
- Policy version.
- Registry version.
- Migration version.

## Rollback

Rollback måste kunna ske för:

- Kod.
- Policy.
- Registry.

Policy rollback ska vara snabbast och kunna göras utan koddeploy.

## Environments

- `local`
- `dev`
- `staging`
- `production`

## Deployment checklist

- Migrations körda.
- Policy aktiverad.
- Registry validerat.
- Provider keys tillgängliga.
- Smoke test passerat.
- Metrics syns.
- Alerts aktiva.
