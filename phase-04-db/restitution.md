# Restitution — Phase 04: Warehouse BC adapters & dual-write (Group 3)

> Versione italiana: [`restitution-IT.md`](./restitution-IT.md)
> Spoken restitution script for CP4 (~4 min). We compare the shape of the seam, not the syntax.
> Design in `repositories/legacy_article_repository.go` + `dual_write_article_repository.go`.

## Opening
We're Group 3. In CP3 we built the Warehouse domain and its `ArticleRepository` **port** — a contract
with no implementation. CP4 is where that port stops being abstract: we give it **real adapters** and
run them **beside the legacy database**. This is the **Strangler Fig** turning on.

## The migration scenario — why an ACL
The Warehouse BC must run **side by side** with the legacy DB during migration: every write lands in
**both** stores so nothing breaks while traffic is still on the monolith. But the two stores model
price differently **on purpose**:

| | `legacy_db` | `warehouse_db` |
|---|---|---|
| price | `DECIMAL(10,2)` — `29.99` | `price_cents BIGINT` — `2999` |
| currency | none (EUR assumed) | explicit `CHAR(3)` — `EUR` |

An **Anti-Corruption Layer (ACL)** contains that mess so it never reaches the clean domain.

## Task 1 — the legacy ACL adapter
`LegacyMySQLArticleRepository` implements the port against `legacy_db`. The heart is the price
translation: `centsToDecimal` (`2999 → "29.99"`) and `decimalToCents` (`"29.99" → 2999`), done with
**integer/string math — never `float64`**, because a float silently corrupts money. On read it
**invents `currency = EUR`** (legacy has no column); on write it **drops it**. `Save` is an idempotent
upsert (`INSERT ... ON DUPLICATE KEY UPDATE`); `Find`/`List` **rehydrate through the domain factories**
(`NewSKU`, `NewMoney`), so `entities.Article` always sees a valid `Money` even though the legacy table
can't represent one. The domain never learns legacy details — that's the ACL's whole job.

## Task 2 — the dual-write / single-read decorator
`DualWriteArticleRepository` is a **decorator**: from the outside it *is* an `ArticleRepository`;
inside it holds the legacy + BC adapters. **Writes** (`Save`/`Delete`) go to **both** stores, **legacy
first**: if legacy fails we stop and don't touch BC; if BC fails we return the error (legacy keeps the
article — no rollback in this exercise). **Reads** go to exactly **one** store, chosen by a `ReadMode`
flag (`ReadFromLegacy` by default → `ReadFromBC` after cutover). No comparison, no reconciliation.
Per ADR-013 this decorator is **transitional**: once cutover completes, it is deleted.

## What we deliberately did NOT do
We touched only the two adapter files. We did **not** change the domain, the port, the BC adapter, or
the in-memory fake — those are fixed. Persisting `Article.Inventories` is out of scope this phase
(only the article row). **No `InventoryRepository`** — that would break the aggregate boundary.

## How we proved it
- **Unit tests (fast, no DB):** the pure conversion tests (`2999 ↔ "29.99"`, rounding, round-trip)
  and the dual-write tests against in-memory fakes (write-to-both; legacy-fails → BC untouched;
  BC-fails → legacy keeps; read-routes-by-mode) — all green.
- **End-to-end against the two real MySQL**, via the `seed` CLI (which runs the same
  `dual.Save`/`dual.FindByID` a handler will call in CP6):
  - `seed write` → `✓ legacy write OK` / `✓ BC write OK`; `legacy.price=29.99, bc.price_cents=2999 EUR`
  - `seed read` → `price=2999 EUR` (mode=legacy)
  - `seed compare` → `OK: stores aligned`

Same concept, two shapes, one clean domain view — that is the ACL working.

## How to run it
```bash
cd phase-04-db
docker compose run --build --rm test                                        # all packages green
docker compose --profile demo run --build --rm seed write demo-1 ABC-001 "Widget" 2999 EUR
docker compose --profile demo run --rm seed read    demo-1                  # price=2999 EUR
docker compose --profile demo run --rm seed compare demo-1                  # OK: stores aligned
docker compose down                                                         # stop the stack
```
Unit tests without Docker: `go test ./...`.

## Closing
The port from CP3 now has real adapters and the seam is **live**: every write lands in both stores,
reads come from one, and the ACL keeps the legacy DECIMAL/no-currency mess out of the domain. Next
(CP5): use cases + an event dispatcher that drains the domain events, on top of these repositories.
Questions?

## Glossary
- **Adapter** — a concrete implementation of a port against one store.
- **Anti-Corruption Layer (ACL)** — an adapter whose job is to contain legacy mess so it never reaches
  the clean domain (here: DECIMAL↔cents, invent/drop currency).
- **Port** — the `ArticleRepository` interface (no implementation) the domain depends on.
- **Dual-write** — write to both stores on every mutation (legacy first).
- **Single-read** — reads served by exactly one store, chosen by mode.
- **Cutover** — the switch from `ReadFromLegacy` to `ReadFromBC` once the new store is trusted.
- **Idempotent upsert** — save-twice-same-state = one row, no error, no duplicate.
- **Strangler Fig** — extract a BC incrementally while the monolith keeps serving.
