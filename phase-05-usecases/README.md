# Phase 05 — Warehouse BC: use cases & a first HTTP API

> Italian version: [`README-IT.md`](./README-IT.md)

```
 _   _            ____
| | | |___  ___  / ___|__ _ ___  ___  ___
| | | / __|/ _ \| |   / _` / __|/ _ \/ __|
| |_| \__ \  __/| |__| (_| \__ \  __/\__ \
 \___/|___/\___| \____\__,_|___/\___||___/

  Phase 05 — Application workflows, exposed over HTTP.
```

Estimated time: ~1h in breakout rooms (one full loop; a second is optional), then a shared restitution.

```text
CP3    Domain layer + repository port
CP4    Real adapters + dual-write
CP5    Use cases + a thin HTTP API on top   <-- you are here
CP6+   The full HTTP layer: auth, list, dual-write wiring, deprecation
```

---

## Your mission

CP4 gave the BC persistence. Now build the layer above it and **expose it over HTTP**: **use cases**
(application workflows) plus a thin **HTTP layer** that maps requests to them. By the end you have a
running API you can test from the browser.

There are **three use cases. Exactly one is given** end to end as the worked example
(`CreateArticle`, from `POST /articles` down to the domain). You take **one more use case all the way
around the loop** — and, only if you have time, a **second (optional)** one.

```
THE THREE USE CASES
  create_article.go        CreateArticleUseCase        ✔ GIVEN   (the one worked example)
  get_article.go           GetArticleUseCase           → LOOP A  (do this one fully)
  change_article_price.go  ChangeArticlePriceUseCase   → LOOP B  (optional, if time)

HTTP LAYER
  handlers/article_handler.go   CreateArticle handler GIVEN; Get / ChangePrice handlers → BUILD
  handlers/router.go            POST /articles route GIVEN; two routes → BUILD (uncomment)
  handlers/openapi.go           GIVEN — serves Swagger UI at /docs
  main.go                       composition root, already wires everything

SCAFFOLDING (given, do not touch)
  usecases/in_memory_article_repository.go, usecases/mocks.go   test fake repository
  dispatcher/, entities/, events/, interfaces/                  ports + domain from CP3/CP4
```

### The slice pattern (read the given example first)

Open `CreateArticle` end to end — this is the exact shape your loop follows:

```text
HTTP request  ->  handler (bind JSON, call the use case, map the result)
              ->  use case (validate, ask the aggregate, Save, Dispatch)
              ->  aggregate (business rule)  ->  repository + dispatcher ports  ->  JSON response
```

Each layer does **one** job: the handler only translates HTTP, the use case orchestrates, the
aggregate owns the rules.

---

## The consegna — one full loop, then (optionally) the second

Take **one use case all the way around this four-step loop before starting the next**:

| Step | What | Why |
|---|---|---|
| **1. Code** | implement the use case (`…Execute`) | the workflow itself |
| **2. Go tests** | make its `*_test.go` green | **test ASAP** — instant feedback while you code, no server needed |
| **3. API** | implement the handler + uncomment the route | expose it over HTTP |
| **4. API tests** | exercise the live endpoint at **`/docs`** | **black-box** — anyone can do it, no Go needed |

### ▸ Loop A — GetArticle · `GET /articles/:id`  (everyone does this one)

1. **Code** — `GetArticleUseCase.Execute` in `usecases/get_article.go`: reject an empty `ID`, `repo.FindByID`, wrap with `fmt.Errorf("GetArticle: %w", err)`.
2. **Go tests** — `usecases/get_article_test.go` green.
3. **API** — `ArticleHandler.GetArticle`: read `c.Param("id")`, call the use case, **404** on `usecases.ErrArticleNotFound`, else **200**. Then uncomment `GET /articles/:id` in `router.go`.
4. **API tests** — at `/docs`: create an article (POST), then GET it (expect 200 + the article); GET a missing id (expect 404).

### ▸ Loop B — ChangeArticlePrice · `PUT /articles/:id/price`  (optional, if time)

1. **Code** — `ChangeArticlePriceUseCase.Execute`: validate, `FindByID`, `NewMoney`, `Article.ChangePrice`, **no-op if the price did not change**, else `Save`, dispatch `events.ArticlePriceChanged`, `ClearPendingEvents`. Rules stay in the aggregate; failures never dispatch.
2. **Go tests** — `usecases/change_article_price_test.go` green.
3. **API** — `ArticleHandler.ChangeArticlePrice`: bind `ChangePriceRequest`, call with `ArticleID = c.Param("id")`, **200 / 400**. Then uncomment `PUT /articles/:id/price`.
4. **API tests** — at `/docs`: change a price (expect 200 + the new price); send the **same** price again (still 200, no-op).

### ▸ Done when

- `docker compose run --rm test` is green for the use case(s) you completed, **and**
- the endpoint works from `/docs` end to end. Loop A alone is a complete result: an API with
  `POST /articles` and `GET /articles/:id`.

---

## Run and test the API

```bash
cd ../phase-04-db && docker compose down          # free port 8081
cd ../phase-05-usecases
docker compose up --build                          # serves on :8081
```

**Test it from the browser — no curl, no Go:** open **http://localhost:8081/docs**. It is a Swagger UI
(like FastAPI's `/docs`): pick an endpoint, click **Try it out**, fill the body, **Execute**, read the
response and status code. `POST /articles` works from the start; `GET` and `PUT` light up as you build
them.

<details>
<summary>Prefer the terminal? The same three calls with <code>curl</code></summary>

```bash
curl -X POST localhost:8081/articles -H 'Content-Type: application/json' \
  -d '{"id":"a1","sku":"ABC-001","name":"Widget","price_cents":1000,"currency":"EUR"}'
curl localhost:8081/articles/a1
curl -X PUT localhost:8081/articles/a1/price -H 'Content-Type: application/json' \
  -d '{"price_cents":1500,"currency":"EUR"}'
```

</details>

Data is in-memory: it lives for the run and resets on restart (persistence was CP4's job).

---

## Scope: what we deliberately left OUT of the HTTP layer

This HTTP layer is a **reduced** version of the CP6 one. On purpose, it does **not** have — and you
should **not** add:

| Left out here | Arrives in |
|---|---|
| Auth / JWT / M2M tokens | CP6–CP7 |
| Middleware beyond logger + recover | CP6 |
| `List` + pagination | CP6 |
| Dual-write / MySQL wiring (in-memory only here) | already CP4; wired to HTTP in CP6 |
| Request-validation framework, error-sentinel juggling across repos | CP6 |

Also don't touch `entities/`, `events/`, `interfaces/`, `dispatcher/`, or the given `handlers/openapi.go`.
If the AI reaches into the domain or a port while you complete a loop, **stop** — it is the wrong layer.

---

## How to work

Small breakout rooms, your AI agent as the engine, **one use case fully around the loop before the
next**:

```text
code -> Go test (green) -> handler -> route -> API test at /docs
```

Keep each diff to the file you are completing (`git diff`). When the Go is unclear, ask the AI to
explain it back in a language you know rather than to write more code.

> **Save first, dispatch after; failures never dispatch.** In Loop B, if `Save` succeeds but `Dispatch`
> fails, the state changed but the event is lost — a real gap left open on purpose, closed later by the
> transactional outbox / Hermes work (CP9). Notice it; don't fix it here.

---

## Solutions

Reference implementations in [`solutions/`](./solutions/): `get_article.expected.go.txt`,
`change_article_price.expected.go.txt`, `article_handler.expected.go.txt`, `router.expected.go.txt`,
with the walkthrough in [`solutions/README.md`](./solutions/README.md). **Open them only after your
loop is green** — checklist, not copy source.

---

## Next phase

→ **Phase 06** turns this into the **full HTTP layer**: `List` + pagination, wiring the handlers onto the
real dual-write repository, auth, and the deprecation policy for breaking API changes.

---

*MIC v0.1 - by Mic (Michele Mondora) - 2026*
