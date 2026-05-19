# Decision log

Använd denna fil för korta produkt-/teknikbeslut som inte kräver full ADR.

| Datum | Beslut | Ägare | Kommentar |
|---|---|---|---|
| 2026-05-19 | Starta med OpenAI-kompatibel proxy | TBD | Sänker adoptionströskel |
| 2026-05-19 | Fast path ska inte använda LLM-classifier | TBD | Skyddar latency och kostnad |
| 2026-05-19 | Implementeringsspråk: Go 1.22 | Magnus | Latency, native streaming, OTel-stöd. LiteLLM (Python) skrotad. |
| 2026-05-19 | Stack sprint 1: stdlib net/http (1.22 ServeMux), log/slog, google/uuid | Magnus | Minimera externa deps tills routing/policy motiverar mer |
| 2026-05-19 | Go-modul `github.com/magnusfroste/tokenizer` (initialt `tokenix`, rebrand samma dag — ISSUE-061) | Magnus | Satt vid första push; repo omdöpt till `tokenizer` senare samma dag. |
| 2026-05-19 | `cmd/worker` är no-op stub i sprint 1 | Magnus | Acceptanskriterium ISSUE-001 kräver "worker"; riktig event-queue i sprint 6 (EPIC-07). |
