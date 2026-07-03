# Restitution — Phase 05: use cases & a thin HTTP API (Group 3)

> Versione italiana: [`restitution-IT.md`](./restitution-IT.md)
> Spoken restitution script for CP5 (~4 min). We compare the shape of the layering, not the syntax.
> Code in `usecases/get_article.go` + `change_article_price.go`, `handlers/article_handler.go` + `router.go`.

## Opening
We're Group 3. CP4 gave the BC persistence. CP5 adds the layer above and **exposes it over HTTP**: the
**use cases** (application workflows) plus a **thin HTTP layer** that maps requests onto them.
`CreateArticle` was given end-to-end as the worked example; we carried **GetArticle** (Loop A) and
**ChangeArticlePrice** (Loop B) all the way round.

## The vertical slice
Every feature is one thin slice through the layers, each doing exactly **one** job:

```text
HTTP handler (bind JSON, call use case, map result)
  → use case (validate, ask the aggregate, Save, Dispatch)
  → aggregate (business rule)  → ports (repository + dispatcher)  → JSON
```

The handler only translates HTTP; the use case orchestrates; the **aggregate owns the rules**.

## Loop A — GetArticle · `GET /articles/:id`
A read-only workflow. The use case validates the id, calls `repo.FindByID`, and **wraps the error with
`%w`** — so the handler can still detect `ErrArticleNotFound` via `errors.Is` and answer **404** (vs
**400** for any other error, **200** on success). The point: the sentinel travels *wrapped*, and the
layer that knows HTTP status codes is the handler, not the use case.

## Loop B — ChangeArticlePrice · `PUT /articles/:id/price`
A mutating workflow, same shape as `CreateArticle`: `FindByID` → `NewMoney` → ask the aggregate to
`ChangePrice`. The **rule lives in the aggregate** (same currency, price > 0); the use case only
orchestrates. Two things worth defending:
- **No-op**: if the new price equals the current one, the aggregate changes nothing and records no
  event — so the use case does **not** `Save` and does **not** `Dispatch`. Idempotent by design.
- **Save before Dispatch, failures never dispatch**: build the canonical `ArticlePriceChanged`,
  `Save`, then `Dispatch`, then `ClearPendingEvents`. If find/save/dispatch fails, we surface the
  error and no event goes out.

## The EDA angle — and an honest gap
The aggregate **records** facts; the use case **dispatches** the canonical event only **after Save
succeeds** — record-then-publish, the *outbox* spirit. Downstream BCs react in **eventual
consistency**. There is a deliberate gap: if `Save` succeeds but `Dispatch` fails, the state changed
but the event is lost. We noted it, we did not paper over it — it is closed later by the transactional
outbox / Hermes work (CP9).

## Clean architecture
**Dependency inversion** throughout: the use case depends on the `ArticleRepository` and
`EventDispatcher` **ports**, not concretes; here they're in-memory fakes, and CP6 wires them to the
real dual-write repository. The domain never sees JSON; the handler never sees SQL.

## What we deliberately left out
This HTTP layer is a **reduced** CP6: no auth/JWT, no middleware beyond logger+recover, no
List/pagination, no dual-write/MySQL wiring, no validation framework. We touched only the two use
cases, their handlers, and the router — **not** the domain, the ports, the dispatcher, the given
`openapi.go`, or `main.go`.

## How we proved it
- **Unit tests (in-memory fakes, no server):** GetArticle returns / 404s / rejects an empty id;
  ChangePrice changes+saves+dispatches exactly one event, no-ops on same price, rejects a currency
  change, and surfaces find/save/dispatch failures without dispatching. All green.
- **Live API via Swagger on `/docs`** (or curl): `POST /articles` → 201; `GET` → 200 / 404;
  `PUT .../price` → 200, same price → 200 (no-op), currency change → 400.

## How to run it
```bash
cd phase-05-usecases
go test ./...                              # use cases + dispatcher green
docker compose up --build                  # :8081  ->  http://localhost:8081/docs (Swagger UI)
#   POST /articles ; GET /articles/:id ; PUT /articles/:id/price
docker compose down
```

## Closing
The BC now has application workflows and a working HTTP surface: a request flows handler → use case →
aggregate → ports and back as JSON, with events **recorded** and **dispatched after Save**. Next
(CP6): the full HTTP layer — `List` + pagination, wiring to the real dual-write repository, auth, and
the API-deprecation policy. Questions?

## Glossary
- **Use case** — an application workflow; orchestrates aggregate + ports, owns no business rule.
- **Vertical slice** — one feature threaded through handler → use case → aggregate → ports.
- **Port** — an interface the use case depends on (`ArticleRepository`, `EventDispatcher`); adapters plug in.
- **Dependency inversion** — inner layers define ports; outer layers implement them.
- **Dispatcher** — fans domain events to subscribers after `Save` (record-then-publish / outbox).
- **No-op idempotency** — re-sending the same price changes nothing and emits no event.
- **Eventual consistency** — downstream BCs react to events asynchronously.
