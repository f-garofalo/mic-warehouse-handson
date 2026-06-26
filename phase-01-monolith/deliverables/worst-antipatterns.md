# MIC — Worst-of List (Top 3 Antipatterns)

> The three architectural choices that hurt the most, why each hurts, and what it costs the
> team to keep living with it. Ordered by blast radius. Evidence cited inline.

---

## #1 — The "God Table": one generic schema for the whole business

**What it is.** All ~18 business entity types share **one** table, `business_data`, discriminated only by a `record_type` string; all relationships share **one** table, `business_relations`, discriminated by `relation_type` (`database/schema.sql`). The columns are meaningless on their own — `amount_1..4`, `text_1..5`, `date_1..3`, plus a `payload_json` overflow. The *meaning* of every column exists only in PHP, in each controller's `toDto()`/`fromPayload()` (e.g. for an `articolo`, `amount_1`=price and `text_2`=VAT code — `ArticleController.php:20-55`). There are **no foreign keys**, **no per-entity NOT NULL/CHECK constraints**, and **no typed columns**.

**Why it hurts.**
- The database can enforce **nothing**: not "an order has a customer", not "a price is positive", not even that a `target_id` points at a real row. Every invariant is optional and lives in scattered PHP.
- You **cannot read the data without the code.** A row `{record_type:'articolo', amount_1:12.5, text_2:'IVA22'}` is undecodable unless you know `ArticleController`. Onboarding, debugging, and reporting all require tribal knowledge.
- **No boundaries to extract along.** Every domain is physically interleaved in the same table, so there is no seam to cut — which is exactly why an extraction needs a deliberate plan.
- Indexes and queries are generic (`idx_type`, `idx_type_status`); they can't be tuned per entity, and every query begins with `WHERE record_type = …`.

**What it costs to keep.** Permanent onboarding tax (everyone must memorise the column-meaning map); a steady stream of "wrong column / wrong type" bugs; data you can't trust for finance or inventory; and a migration that is *harder than it should be* because the storage layer encodes no boundaries. This is the single design choice that makes MIC hardest to change safely — **fixing it (giving each bounded context its own typed store) is the whole point of the lab.**

---

## #2 — No domain layer: business logic reaches across domains by reading raw columns

**What it is.** There is no model layer and there are no contracts between domains. Business rules live inside HTTP controllers that read **other domains' raw rows/columns** directly:
- `OrderController::addRiga()` computes pricing/VAT by reading the article's `amount_1` (base price) and `text_2` (VAT code), then querying the `aliquota_iva` rows by hand (`OrderController.php:131-143`).
- `DashboardController::kpi()` hardcodes `record_type='fattura'|'ordine'|'articolo'|'cliente'|'audit_log'` and joins across them in raw SQL (`DashboardController.php:18-100`).
- `CustomerController` instantiates `OrderController`/`InvoiceController` and calls their `dto()` to render its tabs (`CustomerController.php:64,78`).

**Why it hurts.**
- **High blast radius.** A change to how Article stores price or VAT silently breaks Orders; a change to any entity's storage breaks the Dashboard. Nothing tells you at compile time.
- **No anti-corruption boundary.** You cannot extract Catalog/Warehouse without breaking Orders, because Orders depends on Catalog's *physical columns*, not on a stable interface.
- **Duplicated rules.** The VAT default (`'IVA22'` / `22.0`) is hardcoded in more than one place (`OrderController.php:137,142`); pricing logic is wherever someone needed it.

**What it costs to keep.** Every cross-domain change becomes a careful, repo-wide grep-and-pray. The Strangler extraction is forced to build anti-corruption layers exactly here (Orders↔Catalog) before anything can move — the work this lab sequences over the next phases.

---

## #3 — Untrusted numbers: derived/duplicated financial & stock data, with fabricated values and no invariants

**What it is.** Money and stock have **no single source of truth** and, in places, are simply made up:
- **Stock is derived, never stored**, as `SUM(movimento.amount)` via a 4-way join (`MagazzinoController.php:46-62`) — and it **can go negative**: the UI literally has a red style for `giacenza < 0` (`magazzino.js:76`). No reservation, no lock, nothing prevents overselling.
- **Order totals are both stored and recomputed**: each line keeps `imponibile/iva/iva_perc` in `payload_json` *and* in `amount_*`, and `recalcOrdine()` re-derives the order header (`OrderController.php:146-198`) — three copies that can disagree.
- **Price is duplicated** across `articolo.amount_1`, `voce_listino.amount_1`, and the `articolo_di_voce` relation `amount` (`ListinoController.php:72-81`).
- **Fabricated numbers**: "Genera fattura" splits a total as `imponibile=totale*0.82`, `iva=totale*0.18` (`orders.js:147-148`); the Dashboard's "articoli sotto scorta" KPI is `floor(count*0.18)` — **pure fiction** (`DashboardController.php:41-42`).
- **`Money` has no currency** — every amount is a bare `DECIMAL` (`schema.sql`), so the model can't represent or guard currency.

**Why it hurts.** The same business fact (a total, a stock level, a price) gives different answers depending on where you read it. For an *invoicing and inventory* product, that is a correctness problem, not a cosmetic one: wrong VAT, oversold stock, KPIs nobody can trust. Concurrency makes the derived stock unsafe.

**What it costs to keep.** Constant reconciliation and "why don't these numbers match?" investigations; a real risk of shipping incorrect invoices or negative inventory; and a blocker for extraction, because a clean service must first define a single owner and an invariant for each value (e.g. modelling reservations as an explicit counter, per the Phase-02 target).

---

## Honourable mentions (not in the top 3, but worth flagging)

- **No platform layer / no auth.** A hardcoded "admin" user in the topbar (`index.html:48-54`), placeholder password hashes (`UserController.php:29`), roles enforced nowhere. Fine for a teaching monolith; dangerous if ever mistaken for production.
- **Route table rebuilt on every request** (`index.php:84-135`) and **CDN-loaded Chart.js** (`index.html:9`) — minor, but a runtime coupling to an external host.
- **Orphan domains in the data but not the flow** — `fornitore`, `sconto`, `contatto` exist with seed data but no real process uses them.

## Tie-in to the exit questions
- *Single design choice that makes MIC hardest to change safely?* **#1, the God Table** — it removes every boundary and every DB-enforced invariant.
- *What would you do about it?* Extract bounded contexts one at a time (Strangler Fig), giving each its own typed store and a contract at the seam — starting with the most self-sufficient slice (**Warehouse**), and breaking the **Orders→Article price/VAT** read first.
