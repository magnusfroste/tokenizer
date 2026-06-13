# Database

Local Postgres uses the `docker-compose.yml` defaults:

```bash
docker compose up -d postgres
make migrate
make seed
```

Both targets use `DATABASE_URL`, defaulting to:

```text
postgres://tokenizer:tokenizer@localhost:5432/tokenizer?sslmode=disable
```

The local seed stores only the SHA-256 hash of `LOCAL_API_KEY`; the plaintext local fixture remains in `.env.example`.

`make migrate` applies every `db/migrations/*.sql` file in sorted (numeric) order.
Migrations are idempotent (`CREATE TABLE IF NOT EXISTS`, wrapped in transactions),
so the target is safe to re-run; new migration files are picked up automatically.
