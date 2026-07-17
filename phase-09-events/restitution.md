# Restitution — Phase 09: the fact leaves the BC (Group 3)

> Versione italiana: [`restitution-IT.md`](./restitution-IT.md)
> Spoken restitution script for CP9 (~4 min). We compare the shape, not the syntax.

## Opening
We're Group 3. CP7 wrapped an identity envelope around the BC. CP9 does something more drastic: it
**retires the CP4 dual-write**. Not because it was broken — for MIC's needs it worked — but because this
phase's subject is **event-driven integration**, and with the dual-write on there is no gap to observe.
So the scaffolding comes down: the BC writes **only its own store**, no longer knows the MIC database
exists, and now **mints its own article ids** (`/health` → `"write_mode": "warehouse-only"`).

Retirement opens a hole. The MIC order-entry screen was never migrated: it reads the legacy tables, and
nobody fills them anymore — an article **born in the BC is invisible** to MIC. We close that gap the
event-driven way: the BC publishes a **fact** (a Data Product payload validated against a JSON schema,
wrapped in a **CloudEvents** envelope), and a consumer inside MIC builds a **projection it owns**. The
pipeline — map → validate → envelope → keep — is **given** and already works for `InventoryAdjusted`:
that is our worked example. Two things were missing, both ours: the two ends of the contract, with the
schema in the middle.

## Part 1 — publish the fact (Go, `hermes_publisher.go`)
Two spots, pattern-matched from `InventoryAdjusted`. **`mapEventToSchema`** (Task 1a): map
`ArticleCreated` to the Data Product `warehouse.article.v1` — keys from the contract
(`article-entity-record-v1.json`: `article_id, sku, name, price_cents, currency, updated_at`), values from
the event, the timestamp as RFC3339Nano. The one trap: the event field is `ArticleName`, the contract
says `name`. **`eventMetadata`** (Task 1b): the envelope's `time` is the event's time, its `subject` is
`articles/<article id>`. We **observed the failure first**: before the fix, the create answered **`400`**
and the error *named our task* — "the publisher refuses to dispatch an event it has no contract for".
After the fix: **`201`**, and the fact is on the wire.

## Part 2 — consume the fact (PHP, `consumer.php`)
One function, `mapHermesArticleToMicProjection` — the polling loop and the idempotent upsert are given.
It adapts the CloudEvents record to the row MIC reads: `code` ← sku, `name` (fallback to sku),
`amount_1` = `centsToDecimal(price_cents)` — a **decimal string, never a float** (this is the CP4
Anti-Corruption Layer in reverse: the BC speaks integer cents, MIC speaks `DECIMAL`), plus MIC's defaults
(`text_1='WAREHOUSE-BC'`, `text_2='IVA22'`, `status='attivo'`). And a `payload_json` that carries the
**audit trail**: `projection_kind`, `article_id`, `currency`, and the envelope metadata
(`source_record_id`, `source_record_type`, `source`, `source_subject`, `source_time`) — every projected
row can name the exact Hermes record it came from. Crucially the row is
`record_type='warehouse_article_projection'`, **not** a faked legacy `'articolo'`: a read model MIC owns,
not a second source of truth.

## The thread: envelope vs payload
Read one record with the split in mind. The **envelope** (`specversion, id, source, type, subject, time`)
is uniform routing metadata the *platform* reads — to route, deduplicate, order, trace — without knowing
what an article is. The **payload** (`data`) is the *Data Product*, the domain, versioned as
`warehouse.article.v1`, which the *consumer's business logic* reads. Transport vs meaning. Our consumer
touched both: it filtered by envelope `type` and mapped from `data`.

## How we proved it
- **Tests** (`docker compose run --build --rm test`): all green in-container, including
  `TestHermesPublisher_*` (the `ArticleCreated` cases shipped red).
- **On the wire**: `/health` `warehouse-only`; create → **`201`**, the BC minted `art-bd9a5b0487fe2cf3`;
  `/debug/hermes/records` shows the envelope + payload. Order entry was **empty** before the consumer;
  after one pass the article appears with `"source": "hermes-projection"`, and the
  `warehouse_article_projection` row's `payload_json` points back to the exact Hermes record. With the
  loop consumer up, a **second** article flowed through **by itself** — and no duplicate rows: the upsert
  is idempotent by SKU.
- `git diff` touches **only** the two `ArticleCreated` branches and the one PHP function.

## The five questions (Q1–Q5)
**Q1 — the dual-write kept MIC perfectly consistent; why retire it? Name the costs it hid.** It was
consistent, but at a price. It **coupled** the BC to the legacy schema (tables, columns, `DECIMAL`, the
god-table minting ids) — the migration could never finish. It was **synchronous**: MIC slow or down
blocked BC writes; the producer's availability depended on the consumer's. It was **point-to-point via a
shared DB**: no published contract, so a second consumer meant a third write. And it **blurred
ownership**: two writers, no clear source of truth. Events replace all four with one versioned contract
that N consumers read, asynchronously. We retire it to make the gap — and the discipline that closes it —
visible.

**Q2 — envelope vs payload: which fields does the platform read, which the consumer's business logic, and
why does the split matter?** The **platform** reads the **envelope** — `specversion, id, source, type,
subject, time` — to route, deduplicate, order and trace, uniformly across every event type, without
parsing domain content. The **consumer's business logic** reads the **payload** (`data`): the Data
Product's domain fields. The split separates **transport from meaning**: routing needs no coupling to the
domain, and the domain (`warehouse.article.v1`) can be versioned and evolve without touching routing.

**Q3 — why `record_type='warehouse_article_projection'` instead of a faked `'articolo'` row that would
need no adapter?** Because the BC is now the single source of truth. A faked `'articolo'` row would be a
second, unlabeled write model masquerading as legacy master data — the exact ambiguity the dual-write
had — and MIC logic could edit it and silently diverge. Labeling it a projection tells the truth: a
**read model MIC owns**, derived from the fact, eventually consistent, never edited in place, and
**rebuildable by replaying the events**. The adapter is the price of that honesty.

**Q4 — kill the consumer, create three articles: what stays consistent, what lags, what happens on
restart?** **Consistent**: the BC and its store — the three are saved and the three facts are published;
creates never block on the consumer. **Lags**: the MIC projection — order entry won't see them yet. This
is **eventual consistency**: a bounded staleness window, not data loss. **On restart**: the consumer
polls, reads the records it missed, and the idempotent upsert brings the projection current — the three
appear, with no duplicates. The lag closes.

**Q5 — the failed create still left a row in `warehouse_db` (save ran, publish refused): what does that
half-write tell you, and what would production need?** It exposes the **dual-write problem of event
publishing**: "save" and "publish" are two operations with no shared transaction. If the save commits and
the publish fails — or the process dies between them — the state changed but the fact never traveled, and
the consumer diverges **silently** (worse than our visible `400`, since in production the publish might
fail *after* a `201`). Production needs the **outbox**: in the same transaction as the state change, write
the event to an `outbox` table; a relay publishes from there with at-least-once delivery. Atomicity is
restored, delivery is guaranteed, and idempotent consumers (like ours) absorb the duplicates. In our
exercise the publish is in-process, so the half-write is only a teaching artifact — but it names exactly
why real systems need an outbox.

## How to run it
```bash
cd phase-09-events
docker compose up -d --build                                   # MIC + facade + IAM + BC + consumer + Adminer
docker compose --profile tools run --rm backfill-articles      # the stock the BC took ownership of
docker compose run --build --rm test                           # green suite (incl. TestHermesPublisher_*)
# Task 1 on the wire:
TOKEN=$(curl -s -X POST http://localhost:9001/oauth/token \
  -d "grant_type=password&username=alice&password=demo&client_id=11111111-2222-4333-8444-555555555555&scope=openid profile" \
  | sed -E 's/.*"access_token":"([^"]+)".*/\1/')
curl -s -X POST http://localhost:8083/articles -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"sku":"ART-GAP-002","name":"Born in the BC","price_cents":990,"currency":"EUR"}'   # 201
curl -s "http://localhost:8083/debug/hermes/records?type=warehouse.article.v1"
# Task 2:
docker compose --profile tools run --rm mic-hermes-consumer-once    # one pass; order entry now finds it
docker compose --profile consumer up -d mic-hermes-consumer         # continuous: a new article appears by itself
docker compose --profile consumer down
```
The map: `8081` MIC UI (facade) · `8082` Adminer (server `integration-mysql`, root/root) · `8083`
Warehouse BC · `9001` IAM mock.

## Closing
The dual-write retired, and events took its place: the BC publishes a versioned fact and owns its store;
MIC reads a contract and owns its projection. Same shape as the earlier phases — a contract in the middle,
a default that fails safe — one level up, and now asynchronous. The one debt we named is the outbox; that
is what makes "publish after save" safe in production.

## Glossary
- **Data Product** — the payload (`data`) as a versioned, schema-validated contract (`warehouse.article.v1`).
- **CloudEvents envelope** — uniform routing metadata (`specversion, id, source, type, subject, time`) around any payload.
- **Projection / read model** — a consumer-owned, derived, read-only copy of data; here `record_type='warehouse_article_projection'`.
- **Eventual consistency** — the projection lags the source by a bounded window, then catches up; not data loss.
- **Idempotent upsert** — re-applying the same record changes nothing; keyed by SKU, so repeated polling makes no duplicates.
- **Outbox** — write the event in the same transaction as the state change; a relay publishes it — the fix for the half-write.
- **Anti-Corruption Layer (reverse)** — `centsToDecimal`: the BC's integer cents translated to MIC's `DECIMAL`, at the boundary.
