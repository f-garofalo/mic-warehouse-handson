# Phase 09 — Events: the fact leaves the BC


```
  _______     _______ _   _ _____ ____
 | ____\ \   / / ____| \ | |_   _/ ___|
 |  _|  \ \ / /|  _| |  \| | | | \___ \
 | |___  \ V / | |___| |\  | | |  ___) |
 |_____|  \_/  |_____|_| \_| |_| |____/

  Phase 09 — the dual-write retires; events take its place.
```

Estimated time: ~45min in breakout rooms, then a shared restitution.

```text
CP1-CP5   Domain, adapters, use cases, first HTTP API
CP6bis    The Strangler facade: reads and creates flow through the BC
CP7       Login and the first policy: the identity envelope
CP9       The dual-write retires; the fact leaves the BC as an event   <-- you are here
```

> **Plenary opening (before you start):** on slides we walk why the dual-write
> is scaffolding and not the end state, what a Data Product and a CloudEvents
> envelope are, and why the consumer owns its projection. Then you build.

---

## Your mission

The Phase 04 dual-write is **retired** in this phase. To be clear: it was not
broken, and for MIC's internal needs it could have kept working. We switch it
off because this phase's subject is **event-driven integration**, the general
way facts travel between contexts, and with the dual-write on there would be
no gap to close and nothing to observe. So the scaffolding comes down: the BC
writes only to its own store, no longer knows the MIC database exists, and
even mints its own article ids (the legacy god-table used to do that). Check
`/health`: `"write_mode": "warehouse-only"`.

Retirement opens a hole. The MIC screens that were never migrated still read
the legacy tables, and nobody fills those tables anymore: an article born in
the BC is now **invisible to order entry**. Today you close that gap the
event-driven way: the BC publishes a **fact** (a Hermes record: a Data
Product payload validated against a JSON schema, wrapped in a CloudEvents
envelope), and a consumer inside MIC builds a **projection it owns**
(`record_type='warehouse_article_projection'`). Not a second write model:
a read model, eventually consistent, clearly labeled.

You write **both ends of the contract**; the schema sits in the middle:

1. **the publisher side** (Go, Task 1): map the `ArticleCreated` domain event
   to the `warehouse.article.v1` Data Product. The pipeline is given and
   already works for `InventoryAdjusted`: that mapping is your worked example;
2. **the consumer side** (PHP, Task 2): map the Hermes record into the shape
   MIC can read. The polling loop and the idempotent upsert are given.

| Given / starter | Where |
|---|---|
| The Warehouse BC of Phase 07 (auth + your policy), dual-write retired | root of this phase (given) |
| Hermes publisher pipeline: validate → envelope → keep, `/debug/hermes/records` | `events/hermes_publisher.go` (given), `main.go` |
| The Data Product contracts | `schemas/*.json` (given) |
| The `ArticleCreated` mapping, payload + envelope metadata | `events/hermes_publisher.go` — **STARTER, Task 1 (two TODOs)** |
| Test suite (Task 1 ships red; the worked example is green) | `events/hermes_publisher_test.go` |
| MIC consumer: polling, filtering, idempotent upsert | `mic-integration/consumer/consumer.php` (given) |
| The Hermes record → MIC projection mapping | same file, `mapHermesArticleToMicProjection` — **STARTER, Task 2** |
| Projection adapter: merges legacy list + projection rows for order entry | `mic-integration/adapter/` (given) |
| Integrated stack: MIC + facade + IAM + BC + consumer + Adminer | `docker-compose.yml` |

---

## The consegna

> This file is self-sufficient: everything you need for the hour is here.
> Start the stack first ([Run](#run): it compiles Go, C# and PHP images, it
> takes minutes) — the Task 1 tests need no stack, so you can start there
> while it builds.

### ▸ Task 1 — Publish the fact (`events/hermes_publisher.go`)

1. **Observe the failure.** With the stack up, get Alice's token and try to
   create an article in the BC:

   ```bash
   TOKEN=$(curl -s -X POST http://localhost:9001/oauth/token \
     -d "grant_type=password&username=alice&password=demo&client_id=11111111-2222-4333-8444-555555555555&scope=openid profile" \
     | sed -E 's/.*"access_token":"([^"]+)".*/\1/')

   curl -s -X POST http://localhost:8083/articles \
     -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" \
     -d '{"sku":"ART-GAP-001","name":"Born in the BC","price_cents":990,"currency":"EUR"}'
   ```

   Expected: `400`, and the error **names your task**. The publisher refuses
   to dispatch an event it has no contract for, so the create fails loudly.
   (Meanwhile `POST .../inventory/adjust` already publishes fine: that path
   is the worked example.)
2. **Complete the two TODOs**: the `ArticleCreated` branch in
   `mapEventToSchema` (the Data Product payload: the contract is
   `schemas/article-entity-record-v1.json`) and in `eventMetadata` (the
   envelope's time and subject). Pattern-match the `InventoryAdjusted`
   branches right above each TODO. The specification is
   `events/hermes_publisher_test.go`:

   ```bash
   docker compose run --build --rm test
   ```

3. **Verify on the wire.** Rebuild the BC
   (`docker compose up -d --build warehouse-bc`), repeat step 1 with a
   **fresh SKU** (`ART-GAP-002`): now `201`, and the fact is out:

   ```bash
   curl -s "http://localhost:8083/debug/hermes/records?type=warehouse.article.v1"
   ```

   Read the record with the split in mind: envelope fields
   (`specversion`, `id`, `source`, `type`, `subject`, `time`) are routing
   metadata for any consumer; everything under `data` is your Data Product,
   exactly as the schema declares it.

✅ **Green when:** the suite is green, the create answers `201`, and the
record is visible in the Hermes mock with envelope and payload both correct.

### ▸ Task 2 — Consume the fact (`mic-integration/consumer/consumer.php`)

MIC still cannot see the article: open `http://localhost:8081/`, go to
`Ordini`, open an order and search `ART-GAP-002` in the article selector.
Nothing. The fact exists, but nobody on the MIC side reads it yet.

1. **Complete `mapHermesArticleToMicProjection`** (only that function): the
   consumer-side adapter from the CloudEvents record to the row MIC expects.
   The TODO in the file spells out every key. Syntax check without the stack:

   ```bash
   docker compose --profile tools run --rm --no-deps mic-hermes-consumer-once php -l /consumer/consumer.php
   ```

2. **Run one pass** and watch the projection land:

   ```bash
   docker compose --profile tools run --rm mic-hermes-consumer-once
   ```

   Repeat the order entry search: the article is there, and
   `GET http://localhost:8081/api/articles?q=ART-GAP-002` shows it with
   `"source": "hermes-projection"`. In Adminer, the row sits in
   `mic.business_data` with `record_type='warehouse_article_projection'` and
   a `payload_json` that names the exact Hermes record it came from.
3. **Go continuous.** Start the loop consumer, then create one more article
   in the BC and touch nothing else:

   ```bash
   docker compose --profile consumer up -d mic-hermes-consumer
   ```

   Within a few seconds the new article appears in MIC by itself. Note what
   repeated polling does NOT do: no duplicate rows: the upsert is idempotent
   by SKU.

✅ **Green when:** the projection row exists, order entry finds the article,
and a second article flows through with the consumer running.

### ▸ Flex task — break the contract (skip if behind schedule)

The schema is not documentation: it is an enforced, versioned contract, and
both of your mappings depend on it. Prove it:

1. Remove `"sku"` from the `required` array in
   `schemas/article-entity-record-v1.json`, run the suite: which tests fail,
   and on which side of the contract?
2. Restore it, then rename `"sku"` to `"stock_keeping_unit"` in the schema
   only: an innocent-looking rename. Who breaks now, the producer or the
   consumer, and who would break **silently** in production?
3. Restore the schema before the restitution.

### ▸ Done when

- [ ] `docker compose run --build --rm test` fully green; Go diff touches only the two `ArticleCreated` branches.
- [ ] A fresh article answers `201` and its record shows in `/debug/hermes/records`.
- [ ] Order entry finds the BC-born article; the row is `record_type='warehouse_article_projection'`.
- [ ] With the loop consumer up, a new article appears in MIC with no manual step.
- [ ] You can answer Q1-Q5 below.

---

## Run

> Prerequisites: Rancher Desktop with the Docker engine. No Go, no .NET, no
> PHP needed locally.

**Step 0 — free the ports** (this phase binds 8081, 8082, 8083, 9001):

```bash
cd ../phase-07-auth && docker compose -f mic-integration/docker-compose.auth.yml down
cd ../phase-09-events
docker ps --filter "publish=8081" --filter "publish=8083" --filter "publish=9001"   # expect no rows
```

**Step 1 — start the stack:**

```bash
docker compose up -d --build
docker compose ps
```

Expected up: `integration-mysql` (healthy), `iam-mock`, `mic-app`,
`warehouse-bc`, `mic-projection-adapter`, `strangler-facade`, `adminer`.

**Step 2 — load the warehouse** (one-time copy of the legacy articles, same
ids: the stock the BC took ownership of during the migration):

```bash
docker compose --profile tools run --rm backfill-articles
```

The map: `8081` MIC UI behind the facade · `8082` Adminer (server
`integration-mysql`, root/root) · `8083` Warehouse BC · `9001` IAM mock.

> **Corporate proxy (Netskope):** on `x509: certificate signed by unknown
> authority` build errors, set `NETSKOPE_CA_BUNDLE` to your
> `netskope-cert-bundle.pem` path and append
> `-f docker-compose.netskope.yml` to build/up commands.

Shut down at the end with `docker compose down` (add `--profile consumer` if
the loop consumer is running).

---

## Scope & how to work

You may modify exactly two things: the two `ArticleCreated` TODO branches in
`events/hermes_publisher.go` (Task 1) and `mapHermesArticleToMicProjection`
in `mic-integration/consumer/consumer.php` (Task 2). Everything else —
`Dispatch`, the schemas, the polling loop, the upsert, the adapter, the BC —
is given and off limits. If the AI proposes touching them, **stop**.

### Two hints while you work

> 💡 **Watch the fact travel.** Three observation points, in order:
> `/debug/hermes/records` (did the fact leave the BC?), the consumer logs
> (`docker compose --profile consumer logs -f mic-hermes-consumer`: did MIC
> read it?), Adminer on `mic.business_data` (did the projection land?). When
> something is missing, the first dark point names the guilty side.

> 💡 **Turn the code into a language you know.** The publisher is Go and the
> consumer is PHP — if either is opaque, ask the AI for an equivalent snippet
> in C#, Java, TypeScript, or pseudocode first, understand it there, then
> come back.

Hold for restitution: **Q1** the dual-write kept MIC perfectly consistent —
why retire it instead of keeping it forever? Name the costs it hid. **Q2**
envelope versus payload: which fields does the *platform* read, and which
does the *consumer's business logic* read? Why does the split matter? **Q3**
why does the consumer write `record_type='warehouse_article_projection'`
instead of faking a legacy `'articolo'` row, which would have needed no
adapter? **Q4** kill the consumer and create three articles: what stays
consistent, what lags, and what happens when the consumer comes back? **Q5**
in Task 1 step 1, the failed create still left a row in `warehouse_db` (the
save ran, the publish refused): what does that half-write tell you about
publishing events after saving, and what would production need? (One word:
outbox.)

---

## Solutions

Reference implementations:
[`solutions/hermes_publisher.expected.go.txt`](./solutions/hermes_publisher.expected.go.txt)
(Task 1, full file),
[`solutions/consumer_map.expected.php.txt`](./solutions/consumer_map.expected.php.txt)
(Task 2, the function), walkthrough and Q1-Q5 answers in
[`solutions/README.md`](./solutions/README.md). **Open them only after your
own green** — use as a checklist, not a copy source.

---

## Next phase

→ **Phase 11**: a new kind of consumer walks up to the BC: an AI agent,
through MCP. Different protocol, same discipline: a published contract, a
machine identity, and your policy deciding what it may do.

---

*MIC v0.1 - by Mic (Michele Mondora) - 2026*
