# Restitution — Phase 06: the full HTTP layer (Group 3)

> Versione italiana: [`restitution-IT.md`](./restitution-IT.md)
> Spoken restitution script for CP6 (~4 min). We compare the shape, not the syntax.
> ⚠️ **Invented phase**: the lab stops at `lezione-8-fase-5`. We built CP6 ourselves from the CP5
> README's description. **Auth is deliberately out** (that is CP7).

## Opening
We're Group 3. CP5 gave the BC use cases and a thin HTTP layer — but on an **in-memory** fake. CP6
makes it **real**: the same endpoints now persist through the CP4 **dual-write** repository (two
MySQL), gain **pagination**, and grow a **deprecation policy**. It lives in a new `phase-06-http/`
folder, assembled from CP5 (use cases + HTTP) + CP4 (repositories + the two databases).

## Turning the seam on — dual-write wiring
The composition root (`main.go`) opens the legacy and warehouse MySQL connections from env and injects
the CP4 `DualWriteArticleRepository` into **every** use case. So a `POST`/`PUT` from the HTTP layer now
lands in **both** stores (legacy first), and reads come from **one** store chosen by
`DUAL_WRITE_READ_MODE` (`legacy` today, `bc` after cutover). The in-memory fake is gone from the running
service — it stays only as a test double. The seam we designed across CP3→CP5 is now actually on.

## The sentinel unification (the CP6 wrinkle)
CP4 and CP5 each declared their own `ErrArticleNotFound` (one in `repositories`, one in `usecases`). Wired
together, a not-found from the **real** repo would not match the handler's `errors.Is` check → a missing
article would answer 400 instead of 404. CP6 fixes it the clean way: the sentinel moves to the **port**
(`interfaces.ErrArticleNotFound`); every adapter and the use cases **alias** it. Now not-found flows from
the dual-write repo all the way to a **404**. (This is exactly the "error-sentinel juggling" the CP5 README
deferred to CP6.)

## Pagination
`GET /articles?limit=&offset=` returns `{ "data": [...], "meta": { total, limit, offset } }` — the **same
envelope the MIC monolith used** back in CP1, a deliberate callback. Values are clamped (default 50, max
200, offset ≥ 0). Pagination is applied at the **use-case layer** (fetch-all + slice); DB-side
`LIMIT/OFFSET` would be a later optimization touching the port and every adapter.

## Deprecation policy (RFC 8594)
A small middleware stamps `Deprecation: true`, `Sunset: <date>`, and `Link: <successor>;
rel="successor-version"`. We wrap a **deprecated alias** `GET /v0/articles/:id` (successor:
`/articles/:id`) with it — a concrete demonstration of *how you retire a breaking-changed endpoint with
notice*, not just a comment. It shows up `deprecated: true` in the OpenAPI/Swagger.

## What we deliberately did NOT do
**No auth** (JWT/M2M) — that is CP7. We did not touch the aggregate, the port's method set, or the
dispatcher's contract. `phase-05-usecases/` stays the clean CP5 deliverable.

## How we proved it
- **Unit tests** (`go test ./...`, no DB): all CP3/CP4/CP5 tests carried over green + new pagination tests
  (paginates, defaults/clamp, offset-beyond-total).
- **End-to-end** against the two real MySQL: `POST`/`GET`/`PUT` persist; a missing id → **404 via the
  dual-write repo**; the article **survives a container restart** (real persistence, not in-memory);
  `GET /articles?limit=2&offset=0` returns the `{data,meta}` page; `GET /v0/articles/:id` carries the
  `Deprecation`/`Sunset`/`Link` headers.

## How to run it
```bash
cd phase-06-http
go test ./...                              # all packages green (no DB)
docker compose up --build -d               # 2 MySQL + app :8081  ->  http://localhost:8081/docs
#  POST /articles ; GET /articles?limit=&offset= ; GET /articles/:id (200/404) ; PUT price ; GET /v0/articles/:id
docker compose down                        # (no named volume -> data is fresh each up)
```

## Closing
The extraction is now a running service: HTTP in, dual-write to two MySQL, single-store reads,
paginated lists, and a deprecation policy for evolving the contract. The seam MIC's monolith exposed in
CP1 is now served by the Go BC end to end. Remaining lab-shaped work (auth, OPA policy, Hermes events,
observability) is CP7–CP10. Questions?

## Glossary
- **Dual-write wiring** — the HTTP use cases persist through the CP4 decorator (both stores, legacy first).
- **Read mode / cutover** — reads served from one store; flip `legacy → bc` when the new store is trusted.
- **Port sentinel** — a single `interfaces.ErrArticleNotFound` every layer matches (unifies CP4/CP5).
- **List envelope** — `{data, meta}`, the MIC monolith's shape, reused for pagination.
- **RFC 8594 sunset** — `Deprecation`/`Sunset`/`Link` headers announcing an endpoint's retirement.
