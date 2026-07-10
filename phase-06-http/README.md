# Phase 06 — Warehouse BC: the full HTTP layer

> Italian version: [`README-IT.md`](./README-IT.md)

```
 _   _ _____ _____ ____
| | | |_   _|_   _|  _ \
| |_| | | |   | | | |_) |
|  _  | | |   | | |  __/
|_| |_| |_|   |_| |_|

  Phase 06 — The same endpoints, now real: persistence, lists, deprecation.
```

> ⚠️ **Invented phase (beyond the lab scope).** The instructor's lab stops at `lezione-8-fase-5`: there
> is **no** official scaffolding or tests for Phase 06. We (Group 3) built it from the *"Next phase"*
> section of the CP5 README. **Auth is deliberately out** — it belongs to CP7.

```text
CP3    Domain layer + repository port
CP4    Real adapters + dual-write
CP5    Use cases + a thin HTTP API (in-memory)
CP6    The full HTTP layer: dual-write persistence, list+pagination, deprecation   <-- you are here
CP7+   Auth (JWT / M2M tokens), OPA policy, Hermes events, observability
```

---

## The mission

CP5 gave the BC use cases and a thin HTTP layer — but on an **in-memory** fake. Phase 06 makes it
**real**: the same endpoints now persist through the CP4 **dual-write** repository (two MySQL), gain a
paginated **`List`**, and grow a **deprecation policy** for evolving the contract.

The domain is untouched: the aggregate, events, ports, and dispatcher stay exactly as in CP3/CP4/CP5.
What changes is the **wiring** (from fakes to real DBs) and the **HTTP edge** (list, pagination,
deprecation headers).

```
COMPOSITION ROOT
  main.go                       opens both MySQL connections from env, builds the dual-write repo,
                                injects it into ALL use cases; mounts /health, /docs, the routes

USE CASES (from CP5 + the new List)
  usecases/create_article.go        CreateArticleUseCase
  usecases/get_article.go           GetArticleUseCase
  usecases/change_article_price.go  ChangeArticlePriceUseCase
  usecases/list_articles.go         ListArticlesUseCase          <-- new in CP6 (pagination)

HTTP LAYER
  handlers/article_handler.go   Create / Get / ChangePrice / List
  handlers/router.go            routes + deprecated /v0 group
  handlers/deprecation.go       RFC 8594 middleware (Deprecation/Sunset/Link headers)   <-- new
  handlers/openapi.go           OpenAPI spec -> Swagger UI at /docs

PERSISTENCE (from CP4)
  repositories/legacy_article_repository.go       ACL over the legacy DB (DECIMAL <-> cents)
  repositories/dual_write_article_repository.go   writes to both, reads from one (ReadMode)
  repositories/article_repository.go              warehouse MySQL adapter
  repositories/in_memory_article_repository.go    fake, now only a test double
  legacy-init.sql, warehouse-init.sql             the two databases' schemas

DOMAIN + PORTS (from CP3, unchanged)
  entities/  events/  dispatcher/
  interfaces/repository.go      the port + the single ErrArticleNotFound sentinel   <-- unified in CP6
```

---

## The three sub-steps

Each step follows the lab's rhythm: **implement → green Go tests → verify live → commit**.

### ▸ Step 6a — Real dual-write wiring + sentinel unification

- **Composition root** (`main.go`): opens the `legacy` and `warehouse` MySQL connections from env
  (`LEGACY_DB_*`, `WAREHOUSE_DB_*`, `DUAL_WRITE_READ_MODE`), builds
  `DualWriteArticleRepository(legacy, bc, mode)`, and injects it into **every** use case. The in-memory
  fake is gone from the running service — it stays only as a test double.
- **Single sentinel** (the CP5 README listed this as CP6 work): CP4 and CP5 each declared their own
  `ErrArticleNotFound`. Wired together, a not-found from the **real** repo would not match the handler's
  `errors.Is` check → a missing article would answer **400 instead of 404**. The sentinel moves to the
  **port** (`interfaces.ErrArticleNotFound`); every adapter and the use cases **alias** it. Now not-found
  flows from the dual-write repo all the way to a **404**.
- **Proof**: an article created via `POST` **survives a container restart** (real persistence, no longer
  in-memory); a `GET` of a missing id → **404** through the dual-write repo.

### ▸ Step 6b — Pagination on `GET /articles`

- `ListArticlesUseCase`: input `{Limit, Offset}`, output `{Articles, Total, Limit, Offset}`; loads
  `repo.List`, clamps (`limit` default 50, max 200; `offset ≥ 0`), slices.
- `ListArticles` handler: reads `?limit=&offset=` and answers with the **MIC monolith's envelope**
  `{ "data": [...], "meta": { total, limit, offset } }` — a deliberate callback to CP1.
- Pagination is at the **use-case layer** (fetch-all + slice): correct for the demo; DB-side
  `LIMIT/OFFSET` would be a later optimization touching the port and every adapter.

### ▸ Step 6c — Deprecation policy (RFC 8594)

- `Deprecation(sunset, successor)` middleware that stamps `Deprecation: true`,
  `Sunset: <RFC 1123 date>`, `Link: <successor>; rel="successor-version"`.
- A **deprecated alias** route `GET /v0/articles/:id` (successor `/articles/:id`) wrapped by the
  middleware: a concrete demonstration of *how you announce a breaking-changed endpoint's retirement*,
  not just a comment. It shows up `deprecated: true` in OpenAPI/Swagger.

---

## Run and test

```bash
cd phase-06-http
go test ./...                              # all packages green (no DB)
docker compose up --build -d               # two MySQL + app :8081 + adminer :8082
```

**Test it from the browser** — open **http://localhost:8081/docs** (Swagger UI): pick an endpoint,
**Try it out**, **Execute**, read response and status code. Things to try:

- `POST /articles` → creates and **persists to both MySQL** (dual-write, legacy first).
- `GET /articles?limit=2&offset=0` → `{data, meta:{total,limit,offset}}` envelope.
- `GET /articles/:id` → 200 if it exists, **404** otherwise.
- `PUT /articles/:id/price` → 200; same price = no-op; different currency = 400.
- `GET /v0/articles/:id` → 200 with `Deprecation`/`Sunset`/`Link` headers (shown struck-through in Swagger).

**Inspect the two databases** at **http://localhost:8082** (Adminer) — see the ACL at work: in the legacy
DB `price` is `DECIMAL` (`29.99`), in the warehouse DB it is `price_cents` (`2999`) + `currency` (`EUR`).

| Field | Legacy DB | Warehouse DB |
|---|---|---|
| System | `MySQL` | `MySQL` |
| Server | `legacy-mysql` | `warehouse-mysql` |
| Username | `legacy_user` | `warehouse_user` |
| Password | `legacy_pass` | `warehouse_pass` |
| Database | `legacy_db` | `warehouse_db` |

> In the **Server** field use the container name (not `localhost`): Adminer runs inside the Docker network.

Which store the service reads from is decided by `DUAL_WRITE_READ_MODE` (`legacy` today, `bc` after
cutover); `GET /health` reports the active mode. **No named volume** → data is fresh on each `up`.

```bash
docker compose down                        # stops everything; data does not persist between ups
```

---

## Scope: what we deliberately left OUT

| Left out here | Arrives in |
|---|---|
| Auth / JWT / M2M tokens | CP7 |
| OPA policy / authorization | CP7+ |
| Hermes events / real transactional outbox | CP9 |
| DB-side `LIMIT/OFFSET` (fetch-all + slice here) | future optimization |
| Observability (tracing/metrics) | CP10 |

We did not touch the aggregate, the port's method set, or the dispatcher's contract.
`phase-05-usecases/` stays the clean, in-memory CP5 deliverable.

---

## Restitution

The spoken restitution script (~4 min) is in [`restitution.md`](./restitution.md)
(Italian: [`restitution-IT.md`](./restitution-IT.md)): dual-write wiring, sentinel unification,
`{data,meta}` pagination, RFC 8594 deprecation, and the invented-phase disclaimer.

---

## Next phase

→ **CP7** adds **auth**: JWT / M2M tokens at the HTTP edge, and the authorization policy (OPA). From
there CP8–CP10 bring Hermes events, a real transactional outbox, and observability.

---

*MIC v0.1 - by Mic (Michele Mondora) - 2026 · Phase 06 invented by Group 3*
