# Phase 01 — Deliverables (Breakout 1: understand MIC)

> Versione italiana: [`README-IT.md`](./README-IT.md)

The five artifacts required by `../README.md`. Each is grounded in the source
(code + `database/schema.sql` + `database/seed.sql`), with `file:line` evidence inline.

| # | Artifact | Answers |
|---|---|---|
| 1 | [`page-map.md`](./page-map.md) | Every screen and what it's for (grouped by domain). |
| 2 | [`architecture.md`](./architecture.md) | Running components + how a request flows browser → DB → back. |
| 3 | [`repo-guide.md`](./repo-guide.md) | What each file/folder is, and where to look to change things. |
| 4 | [`coupling-map.md`](./coupling-map.md) | How data is really stored, who reads/writes whom, couplings to break. |
| 5 | [`worst-antipatterns.md`](./worst-antipatterns.md) | Top 3 antipatterns: what, why it hurts, cost to keep. |

## The two findings that converge

1. **The storage.** All ~18 entity types live in **two generic tables** (`business_data` + `business_relations`), told apart only by a type column; the meaning of every generic column lives in PHP controllers, not the schema. Ten tidy screens, zero boundaries underneath.
2. **The coupling.** **Orders reaches straight into the article rows** for price (`articolo.amount_1`) and VAT (`articolo.text_2` → `aliquota_iva`) in `OrderController::addRiga` — the dependency a Warehouse extraction must break first.

> Live run not captured: the Docker engine wasn't running in this environment.
> To verify against a live stack: `docker compose up --build` from `phase-01-monolith/`,
> then the two smoke checks in `../README.md` (§ Step 3).
