# Restitution — Phase 06bis: the Strangler facade (Group 3)

> Versione italiana: [`restitution-IT.md`](./restitution-IT.md)
> Spoken restitution script for CP6bis (~4 min) + answers to Q1–Q8.
> ⚠️ **Lab constraint honored**: the only code changed is the `decideUpstream` function in
> `mic-integration/facade/main.go` (Part 1 + Part 3) and the removal of one `t.Skip` line in the test.
> Nothing else (adapter, monolith, BC, DB schemas, test tables) was touched.

## Opening
We are Group 3. Don't confuse this phase with our `phase-06-http`: there we made the BC's HTTP layer
**real** (dual-write persistence, pagination, deprecation). Here the BC and the CP4 dual-write are
**given, in production**. CP6bis is a different problem: **migrating traffic** from MIC's PHP monolith
to the new BC, one operation at a time, safely and reversibly — the **Strangler Fig** pattern.

The missing piece is the **facade**: a service that decides, request by request, *who answers*. All the
plumbing (reverse proxy, `X-Strangler-Route` header, the two forwarding channels) was already written.
Our job was **a single function**: `decideUpstream(method, mode)`.

## Part 1 — The routing decision
The rule: only the **article list** (`GET /api/articles`) is migrated. With `ROUTE_MODE=warehouse-bc` the
`GET` goes to the `warehouse-bc` upstream; with `legacy` — or **any unexpected value** — it stays on the
monolith. Every other method → monolith (not migrated yet).

The design point is the **fail-safe fallthrough**: a mistyped `ROUTE_MODE` must not become an outage.
A router that fails *toward the new system* turns a typo into an incident; we fail **toward the old one**.
And because the BC's `READ_MODE` is still `legacy`, routing the read is a **pure traffic migration**: the
data source doesn't change, the only risk is availability, and the facade dial is its rollback.

## Part 2 — Two worlds, one app; then break the truth
Clicking around MIC with the Network tab open: the **list** answers `X-Strangler-Route: warehouse-bc`, an
article's **detail** and every other screen (customers, invoices) answer `monolith`. Same app, two worlds,
route by route.

Then the incident: stop `warehouse-bc` → the list returns **502** even though its data still lives in the
legacy DB. That is the **availability dependency** we added. Roll back with the facade dial alone
(`ROUTE_MODE=legacy`, recreate) **while the BC is still down**: the list is back. No deploy, no touching MIC.

Finally, break the truth: edit an article from the UI (the update is **not** migrated → it lands only in
the legacy DB), then turn the **second dial** `READ_MODE=warehouse`. The list now reads `warehouse_db`
(backfilled, fed by no write path): the edit **isn't there**. The detail, not migrated, reads legacy: the
edit **is there**. Same article, two truths, one click apart.

## Part 3 — The create cutover
The cure for write staleness is to make writes reach **both** stores — exactly the CP4 dual-write. We
extended the dial: now **GET and POST** follow `ROUTE_MODE`; update and delete stay on the monolith.
With the `t.Skip` removed, the whole suite is green.

A create from the UI now travels facade → adapter → BC → dual-write: **legacy first** (which mints the id),
then warehouse **with the same id**. The new article appears in the migrated list immediately (no staleness
on creates), and in Adminer you see the same id in `mic.business_data` **and** `warehouse_db.articles`.

## Restitution answers (Q1–Q8)
- **Q1 — Which URL does the UI keep calling?** Always `/api/articles` (see `mic-monolith/php-app/public/js/articles.js`). This stable browser contract is what makes the Strangler move safe.
- **Q2 — What does each dial decide?** `ROUTE_MODE` (facade) = *who serves the route* (monolith or BC). `READ_MODE` (BC) = *which store is the truth* (`mic` legacy via ACL or `warehouse_db`). Two independent layers: traffic and data.
- **Q3 — What did we strangle so far?** One operation at a time: the **list read** (Part 1) and the **create** (Part 3). Not a screen, not a CRUD module: one route+method pair at a time.
- **Q4 — Where did the edit go, and why do list and detail disagree?** Into the legacy DB only (the update isn't migrated). The migrated list reads `warehouse_db`, which no write path feeds yet; the detail reads legacy. Two stores, two truths: the cure is migrating the writes, or keeping `READ_MODE=legacy` until you do.
- **Q5 — What breaks when the BC is down?** Every migrated operation (the list; after Part 3, the create too). Everything monolith-owned keeps working: the blast radius equals **exactly** the migrated subset — the argument for cutting small.
- **Q6 — Why did rollback not require touching MIC?** Because the browser contract never changed: MIC still calls `/api/articles`, and the only thing that moved is the facade decision. Retreat is a dial move, not a deploy — and it works precisely when the new system is on fire.
- **Q7 — Why does the dual-write write legacy first?** Because legacy is the system of record and must never be left behind; and concretely it **mints the identity** (the id is the `business_data` autoincrement, assigned on the legacy insert and then reused for the warehouse row). You can't write the warehouse first: you don't know its id yet.
- **Q8 — Creates are safe now: can we leave `READ_MODE=warehouse`?** Not yet. Updates and deletes still flow to legacy only: every edit re-opens the gap. Either migrate those writes too, or keep reads on `legacy`. A migration is finished for a resource only when **every** write path is migrated.

## How we proved it
- **Facade tests** (`go test ./...` in `mic-integration/facade`, self-contained, no Docker): green for the
  Part 1 table, the Part 3 table (skip removed), and the `X-Strangler-Route` HTTP scope test.
- **Minimal diff**: `git diff` touches only `decideUpstream` and the removed `t.Skip` line — as the lab checklist requires.
- **Runtime (Part 2/3)**: the Docker-stack demos (dial flips, incident+rollback, staleness, same-id in
  Adminer) run via `docker-compose.strangler.yml` — done in class where Docker is available.

## How to run
```bash
cd phase-06bis-strangler/mic-integration
# code (no Docker):
( cd facade && go test ./... )                                                   # suite green
# full stack (in class, with Docker):
docker compose -f docker-compose.strangler.yml up -d --build
docker compose -f docker-compose.strangler.yml --profile tools run --rm backfill-articles
docker compose -f docker-compose.strangler.yml --profile test run --rm facade-test
ROUTE_MODE=warehouse-bc docker compose -f docker-compose.strangler.yml up -d --build strangler-facade
curl -i "http://localhost:8081/api/articles?limit=3"                             # X-Strangler-Route: warehouse-bc
docker compose -f docker-compose.strangler.yml down
```

## Closing
We added the migration's control point **between the client and the backends**. The list, then the create,
now cross the BC; the rest stays on the monolith until its turn. Retreat is a dial, not a deploy. The
remaining work (auth, request context, correlation) is CP7. Questions?

## Glossary
- **Strangler Fig** — wrap the legacy in a facade and migrate route by route, until the old system is empty and can be switched off.
- **Facade** — the service that decides, per request, who answers; the single place the cutover decision lives.
- **ROUTE_MODE / READ_MODE** — the two dials: *who serves the route* vs *which store is the truth*.
- **Fail-safe** — an unexpected `ROUTE_MODE` routes toward the old system: a typo doesn't become an outage.
- **Staleness** — reading from a store no write path updates: list and detail disagree.
- **Same-id dual-write** — legacy mints the id on insert, warehouse reuses it: one identity across two stores.
