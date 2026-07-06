# Phase 06bis — Build and operate the Strangler facade

```
  ___| |_ _ _ __ _ _ _  __ _| |___ _ _
 (_-<  _| '_/ _` | ' \/ _` | / -_) '_|
 /__/\__|_| \__,_|_||_\__, |_\___|_|
                      |___/
  Phase 06bis — Same screen, different backend.
```

Estimated time: ~1h in breakout rooms, then a shared restitution.

```text
CP1-CP3   You mapped MIC, chose the Warehouse BC, built its domain layer
CP4-CP5   You gave it adapters, use cases, and a first HTTP API
CP6bis    Your Phase 04 dual-write goes live behind a facade you write   <-- you are here
CP7       Auth, request context, and correlation
```

> **Plenary opening (before you start):** we recap the Strangler Fig pattern, the facade,
> and the two dials of the migration on slides. Then you build and operate.

---

## Your mission

The Warehouse BC in this folder runs the **real migration wiring: the dual-write / single-read
decorator you built in Phase 04**, now in production. Its legacy adapter is an ACL over MIC's
actual god-table (`business_data`), its other adapter writes the new `warehouse_db`, and a
`READ_MODE` dial decides which store it trusts for reads.

What is missing is the piece in between: the **Strangler facade**, the service that decides,
request by request, *who answers*. Its routing decision is **one Go function, and you write
it** (Part 1). Then you operate the migration with it (Part 2), and, time permitting, you
migrate the first write (Part 3).

We migrate **operation by operation**, and eventually everything migrates. Today's subset:
the article **read** (the list) and the article **create**. Updates, deletes, item routes,
and every non-article screen are simply *not migrated yet*: they stay on the monolith.

```
MIC UI ──/api/articles──> YOUR facade ──"monolith"─────> PHP monolith ─────────────┐
                                    └──"warehouse-bc"──> adapter ──> Warehouse BC   │
                                                                    (your Phase 04  │
                                                                     dual-write)    │
                                                             ├── ACL ──> mic DB <───┘
                                                             └── BC adapter ──> warehouse_db
```

Note what the BC's position implies: once a route is migrated, the BC is in its path **even
when the data still lives in the legacy DB**. You are adding an availability dependency
without adding a data risk — and the facade is the lever that rolls it back in one move.

| Given / starter | Where |
|---|---|
| Local MIC monolith (UI + PHP + `business_data`) | `mic-monolith/` |
| Warehouse BC with your Phase 04 dual-write + legacy ACL | this folder (`repositories/`) |
| Facade skeleton, **routing decision stubbed** | `mic-integration/facade/main.go` |
| Facade test suite (ships red; Part 3 test ships skipped) | `mic-integration/facade/main_test.go` |
| Compatibility adapter (list + create translation) | `mic-integration/adapter/` |
| One-time warehouse backfill (same ids as legacy) | `mic-integration/backfill-articles.sql` |
| Integrated stack (MIC + DB + BC + adapter + facade + Adminer) | `mic-integration/docker-compose.strangler.yml` |

---

## The consegna

> This file is self-sufficient: everything you need for the hour is here. Part 1 needs no
> running stack (its tests are self-contained), so start the stack build in the background
> ([Run](#run), Steps 0-2) while you work on it. Part 3 is the flex: skip it if the clock
> is against you.

### ▸ Part 1 — Write the routing decision, route the read

Open `mic-integration/facade/main.go`. Everything is given (the two forwarding channels, the
HTTP plumbing, the `X-Strangler-Route` response header that names who answered) except the
decision:

```go
func decideUpstream(method, mode string) string {
    return upstreamMonolith // TODO: everything stays legacy until you implement the decision
}
```

The rules for Part 1 (`mode` comes from the `ROUTE_MODE` env var, fixed per process:
moving the dial is a restart):

1. `GET` (the article list): mode `warehouse-bc` sends it to the `warehouse-bc` upstream;
   mode `legacy`, or anything unexpected, keeps it on the monolith (fail safe, toward the
   old system).
2. Any other method (create, update, delete): monolith. Those operations are **not migrated
   yet** — Part 3 changes this rule for the create.

`mic-integration/facade/main_test.go` ships **red**: its Part 1 table is the specification
(the Part 3 test is there too, skipped for now). Write your own AI prompt from this checklist:

| Element | Must appear in the prompt |
|---|---|
| Modifiable file | `mic-integration/facade/main.go`, the `decideUpstream` function only |
| The rules | the two rules above |
| Forbidden | the tests, the rest of `main.go`, adapter, monolith, BC |
| Required AI output | show the diff |

Green when:

```bash
docker compose -f mic-integration/docker-compose.strangler.yml --profile test run --rm facade-test
```

Then **route the read**: recreate your facade with the dial on the new position:

```bash
ROUTE_MODE=warehouse-bc docker compose -f mic-integration/docker-compose.strangler.yml up -d --build strangler-facade
```

(PowerShell: set `$env:ROUTE_MODE = "warehouse-bc"` first, then run the same command.)
`curl.exe -i "http://localhost:8081/api/articles?limit=3"` now answers with
`X-Strangler-Route: warehouse-bc` — and the data is **identical and fresh**, because the BC's
`READ_MODE` is still `legacy`: it reads the same MIC table the monolith writes, through the
ACL. Pure traffic migration, zero data risk.

✅ **Green when:** facade suite green, and the list flips between `monolith` and
`warehouse-bc` with the `ROUTE_MODE` dial.

### ▸ Part 2 — Test it from the webapp, then break the truth

1. **Click around MIC** at `http://localhost:8081/#/articles` with DevTools Network open (or
   the facade logs streaming). The article **list** shows `X-Strangler-Route: warehouse-bc`;
   open an article's **detail**, or any other screen (clienti, fatture): `monolith`. Same app,
   two worlds, route by route.
2. **Simulate a BC incident.** `docker compose -f mic-integration/docker-compose.strangler.yml stop warehouse-bc`:
   the list breaks (`502`) even though its data lives in the legacy DB — that is the
   availability dependency you added. Everything else still works. **Roll back with the
   facade** (`ROUTE_MODE=legacy`, recreate) while the BC is still down: the list is back.
   Then restart the BC and put the facade dial back on `warehouse-bc`.
3. **Now break the truth.** First make a change the old way: in the MIC UI, **edit an
   article** (change its price or name — the update is not migrated, so it lands only in the
   legacy DB). Then flip the **second dial**, the BC's read mode:

   ```bash
   READ_MODE=warehouse docker compose -f mic-integration/docker-compose.strangler.yml up -d warehouse-bc
   ```

   Reload the list: your edit **is not there** (the list now reads `warehouse_db`, backfilled
   at setup, which no write path feeds). Open the same article's detail: the edit **is
   there** (detail is not migrated: monolith → legacy DB). Same article, two truths, one
   click apart. That is staleness, made physical.

Hold for restitution: **Q1** which URL does the UI keep calling, and where do you see it?
**Q2** two dials exist now: what does `ROUTE_MODE` decide, and what does `READ_MODE` decide?
**Q3** what exactly did you strangle so far? **Q4** where did your edit go, and why do list
and detail disagree? **Q5** what breaks when the BC is down, and what should still work?
**Q6** why did rollback not require touching MIC?

### ▸ Part 3 — Migrate the create (skip if behind schedule)

The staleness has one cure: writes must reach **both** stores, and the owner of that job is
the dual-write you built in Phase 04. Route the create to it:

1. In `mic-integration/facade/main_test.go`, remove the `t.Skip` line from
   `TestDecideUpstream_part3CreateCutover`: the table below it is the new specification
   (the create follows the dial; updates still do not).
2. Update **your** `decideUpstream` until the whole suite is green — migration is changing
   the rules of your own dial over time. Then rebuild the facade
   (`ROUTE_MODE=warehouse-bc ... up -d --build strangler-facade`).
3. **Create an article from the MIC UI** (Articoli → new). The create now travels
   facade → adapter → BC → dual-write: legacy first (which **mints the id**), then
   warehouse with the **same id**. The new article appears in the migrated list immediately
   (no staleness for creates anymore), and in Adminer you can see the same id in
   `mic.business_data` *and* `warehouse_db.articles` — with the legacy row's non-BC fields
   filled by defaults (`categoria = WAREHOUSE`...): the write-side seam.

Hold for restitution: **Q7** why does the dual-write write legacy first — what does the
legacy store mint? **Q8** creates are now safe, updates are not: can we leave `READ_MODE`
on `warehouse`?

### ▸ Done when

- [ ] Facade suite green; `git diff` touches only `decideUpstream` (and, in Part 3, the removed skip).
- [ ] The list flips `monolith` ↔ `warehouse-bc` with `ROUTE_MODE`; detail and other screens always `monolith`.
- [ ] With `READ_MODE=legacy` the migrated list is identical and fresh; the BC incident breaks only the list; the facade rolls it back with the BC still down.
- [ ] After an edit + `READ_MODE=warehouse`: list stale, detail fresh, and you can say why.
- [ ] Part 3 (if done): a create from the UI lands in **both** stores with the **same id**, and the migrated list shows it.
- [ ] You can answer Q1-Q6 (and Q7-Q8 if you did Part 3).

---

## Run

> Prerequisites: Rancher Desktop with the Docker engine. No local Go needed (build and tests
> run in containers).

**Step 0 — free the ports** (this phase binds 8081, 8082):

```bash
docker ps --filter "publish=8081" --filter "publish=8082"   # expect no rows
```

If a container shows up, `docker compose down` in the folder of the phase that owns it.

**Step 1 — start the integrated stack:**

```bash
docker compose -f mic-integration/docker-compose.strangler.yml up -d --build
docker compose -f mic-integration/docker-compose.strangler.yml ps
```

Expected: `integration-mysql` healthy; `mic-app`, `warehouse-bc`, `mic-compat-adapter`,
`strangler-facade`, `adminer` up.

**Step 2 — backfill the warehouse** (one-time copy of the article history, same ids):

```bash
docker compose -f mic-integration/docker-compose.strangler.yml --profile tools run --rm backfill-articles
```

Useful while you work:

```bash
docker compose -f mic-integration/docker-compose.strangler.yml logs -f strangler-facade      # who answers, live
docker compose -f mic-integration/docker-compose.strangler.yml exec warehouse-bc wget -qO- http://localhost:8081/health   # {"status":"ok","read_mode":"..."}
```

> **Corporate proxy (Netskope):** on `x509: certificate signed by unknown authority` build
> errors, set `NETSKOPE_CA_BUNDLE` to your `netskope-cert-bundle.pem` path and append
> `-f mic-integration/docker-compose.netskope.yml` to build/up commands.

Shut down at the end with `docker compose -f mic-integration/docker-compose.strangler.yml down`
(add `-v` only to delete demo data).

---

## Scope & how to work

The only code you may change is the `decideUpstream` function in
`mic-integration/facade/main.go` (plus removing one `t.Skip` line in Part 3). Do **not**
touch the rest of the facade, the tests' tables, MIC JavaScript, the adapter, the BC, the
database schemas. If the AI proposes touching any of those, **stop**: that change misses the
point of the lab.

Use the AI as a **migration coach, not a code generator**: ask it to explain the request path
from UI to facade to adapter to BC to the two stores, to review your diff before rebuilding,
to predict what breaks if the BC is down, or to explain why list and detail disagree.

### Two hints while you work

> 💡 **Watch both stores in Adminer** (`http://localhost:8082`, server `integration-mysql`,
> user `root`/`root`): the articles live in `mic.business_data` (`record_type='articolo'`,
> the god-table) and in `warehouse_db.articles` (the clean schema). Every claim in this lab
> (freshness, staleness, dual-write, same id) can be checked there in one query.

> 💡 **Two dials, two questions.** `ROUTE_MODE` (facade) answers *who serves the route*;
> `READ_MODE` (BC) answers *which store is the truth*. When something looks wrong, check
> both dials before blaming the code: `/facade/health` and the BC `/health` tell you where
> they point.

---

## Solutions

Reference implementations for Part 1 and Part 3 of the decision:
[`solutions/facade_decide.expected.go.txt`](./solutions/facade_decide.expected.go.txt).
Reasoned answers to Q1-Q8 are in [`solutions/README.md`](./solutions/README.md). **Open them
only after your own green and your own cutover** — the value of the lab is noticing the two
truths before reading the answer.

---

## Next phase

→ **Phase 07**: once traffic crosses a facade, routing is no longer just a code problem.
The BC gets an **identity envelope** — JWT validation, M2M tokens, correlation headers — and
the cut you just made breaks on purpose until MIC learns to authenticate.

---

*MIC v0.1 - by Mic (Michele Mondora) - 2026*
