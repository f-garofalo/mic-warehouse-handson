# Fase 06 — Warehouse BC: il layer HTTP completo

> Versione inglese: [`README.md`](./README.md)

```
 _   _ _____ _____ ____
| | | |_   _|_   _|  _ \
| |_| | | |   | | | |_) |
|  _  | | |   | | |  __/
|_| |_| |_|   |_| |_|

  Fase 06 — Gli stessi endpoint, ma reali: persistenza, liste, deprecation.
```

> ⚠️ **Fase inventata (oltre lo scope del lab).** Il lab del docente si ferma a `lezione-8-fase-5`:
> per la Fase 06 **non esiste** uno scaffolding né dei test ufficiali. L'abbiamo costruita noi (Gruppo 3)
> a partire dalla sezione *"Fase successiva"* del README di CP5. **L'auth è esclusa di proposito** — è
> più propriamente CP7.

```text
CP3    Domain layer + porta del repository
CP4    Adapter reali + dual-write
CP5    Use case + una sottile API HTTP (in memoria)
CP6    Il layer HTTP completo: persistenza dual-write, list+paginazione, deprecation   <-- sei qui
CP7+   Auth (JWT / token M2M), policy OPA, eventi Hermes, observability
```

---

## La missione

CP5 ha dato al BC gli use case e un sottile layer HTTP — ma sopra un fake **in memoria**. La Fase 06 lo
rende **reale**: gli stessi endpoint ora persistono attraverso il repository **dual-write** di CP4 (due
MySQL), guadagnano una **`List` paginata** e una **policy di deprecation** per far evolvere il contratto.

Non si tocca il dominio: l'aggregate, gli eventi, le porte e il dispatcher restano quelli di CP3/CP4/CP5.
Cambia solo il **cablaggio** (dai fake ai DB reali) e il **bordo HTTP** (list, paginazione, header di
deprecation).

```
COMPOSITION ROOT
  main.go                       apre le due connessioni MySQL da env, costruisce il dual-write,
                                lo inietta in TUTTI gli use case; monta /health, /docs, le rotte

USE CASE (da CP5 + il nuovo List)
  usecases/create_article.go        CreateArticleUseCase
  usecases/get_article.go           GetArticleUseCase
  usecases/change_article_price.go  ChangeArticlePriceUseCase
  usecases/list_articles.go         ListArticlesUseCase          <-- nuovo in CP6 (paginazione)

LAYER HTTP
  handlers/article_handler.go   Create / Get / ChangePrice / List
  handlers/router.go            rotte + gruppo /v0 deprecato
  handlers/deprecation.go       middleware RFC 8594 (header Deprecation/Sunset/Link)   <-- nuovo
  handlers/openapi.go           spec OpenAPI -> Swagger UI su /docs

PERSISTENZA (da CP4)
  repositories/legacy_article_repository.go       ACL sul DB legacy (DECIMAL <-> cents)
  repositories/dual_write_article_repository.go   scrive su entrambi, legge da uno (ReadMode)
  repositories/article_repository.go              adapter MySQL del warehouse
  repositories/in_memory_article_repository.go    fake, ora solo come test double
  legacy-init.sql, warehouse-init.sql             schema dei due database

DOMINIO + PORTE (da CP3, invariato)
  entities/  events/  dispatcher/
  interfaces/repository.go      la porta + il sentinel unico ErrArticleNotFound   <-- unificato in CP6
```

---

## I tre sotto-step

Ogni step segue lo stesso ritmo del lab: **implemento → test Go verdi → verifico dal vivo → commit**.

### ▸ Step 6a — Wiring dual-write reale + unificazione del sentinel

- **Composition root** (`main.go`): apre le connessioni `legacy` e `warehouse` MySQL da env
  (`LEGACY_DB_*`, `WAREHOUSE_DB_*`, `DUAL_WRITE_READ_MODE`), costruisce
  `DualWriteArticleRepository(legacy, bc, mode)` e lo inietta in **tutti** gli use case. Il fake in
  memoria sparisce dal servizio in esecuzione — resta solo come test double.
- **Sentinel unico** (il README di CP5 lo elencava come lavoro di CP6): CP4 e CP5 dichiaravano ciascuna
  il proprio `ErrArticleNotFound`. Collegati, un not-found dal repo **reale** non avrebbe matchato il
  controllo `errors.Is` dell'handler → un articolo mancante avrebbe risposto **400 invece di 404**. Il
  sentinel si sposta nella **porta** (`interfaces.ErrArticleNotFound`); ogni adapter e gli use case lo
  **aliasano**. Ora il not-found risale dal dual-write fino a un **404**.
- **Prova**: un articolo creato via `POST` **sopravvive al restart del container** (persistenza reale,
  non più in memoria); `GET` di un id inesistente → **404** attraverso il dual-write.

### ▸ Step 6b — Paginazione su `GET /articles`

- `ListArticlesUseCase`: input `{Limit, Offset}`, output `{Articles, Total, Limit, Offset}`; carica
  `repo.List`, clampa (`limit` default 50, max 200; `offset ≥ 0`), fa lo slice.
- Handler `ListArticles`: legge `?limit=&offset=` e risponde con l'**envelope del monolite MIC**
  `{ "data": [...], "meta": { total, limit, offset } }` — un callback voluto a CP1.
- La paginazione è a livello **use case** (fetch-all + slice): corretta per la demo; il `LIMIT/OFFSET`
  lato SQL sarebbe un'ottimizzazione successiva che toccherebbe la porta e ogni adapter.

### ▸ Step 6c — Policy di deprecation (RFC 8594)

- Middleware `Deprecation(sunset, successor)` che stampa gli header `Deprecation: true`,
  `Sunset: <data RFC 1123>`, `Link: <successore>; rel="successor-version"`.
- Rotta **alias deprecata** `GET /v0/articles/:id` (successore `/articles/:id`) avvolta dal middleware:
  una dimostrazione concreta di *come si annuncia il ritiro di un endpoint con breaking change*, non un
  semplice commento. Compare `deprecated: true` in OpenAPI/Swagger.

---

## Esecuzione e test

```bash
cd phase-06-http
go test ./...                              # tutti i package verdi (senza DB)
docker compose up --build -d               # due MySQL + app :8081 + adminer :8082
```

**Testala dal browser** — apri **http://localhost:8081/docs** (Swagger UI): scegli un endpoint,
**Try it out**, **Execute**, leggi risposta e status code. Cose da provare:

- `POST /articles` → crea e **persiste su entrambi i MySQL** (dual-write, legacy per primo).
- `GET /articles?limit=2&offset=0` → envelope `{data, meta:{total,limit,offset}}`.
- `GET /articles/:id` → 200 se esiste, **404** altrimenti.
- `PUT /articles/:id/price` → 200; stesso prezzo = no-op; valuta diversa = 400.
- `GET /v0/articles/:id` → 200 con header `Deprecation`/`Sunset`/`Link` (in Swagger appare barrato).

**Ispeziona i due database** su **http://localhost:8082** (Adminer) — vedi l'ACL al lavoro: nel legacy
`price` è `DECIMAL` (`29.99`), nel warehouse `price_cents` (`2999`) + `currency` (`EUR`).

| Campo | Legacy DB | Warehouse DB |
|---|---|---|
| System | `MySQL` | `MySQL` |
| Server | `legacy-mysql` | `warehouse-mysql` |
| Username | `legacy_user` | `warehouse_user` |
| Password | `legacy_pass` | `warehouse_pass` |
| Database | `legacy_db` | `warehouse_db` |

> Nel campo **Server** usa il nome del container (non `localhost`): Adminer gira dentro la rete Docker.

Con quale store legge il servizio lo decide `DUAL_WRITE_READ_MODE` (`legacy` oggi, `bc` dopo il cutover);
`GET /health` riporta il mode attivo. **Nessun volume nominato** → i dati sono freschi a ogni `up`.

```bash
docker compose down                        # ferma tutto; i dati non persistono tra un up e l'altro
```

---

## Scope: cosa abbiamo lasciato FUORI di proposito

| Lasciato fuori qui | Arriva in |
|---|---|
| Auth / JWT / token M2M | CP7 |
| Policy OPA / autorizzazione | CP7+ |
| Eventi verso Hermes / transactional outbox reale | CP9 |
| `LIMIT/OFFSET` lato SQL (qui fetch-all + slice) | ottimizzazione futura |
| Observability (tracing/metrics) | CP10 |

Non abbiamo toccato l'aggregate, l'insieme di metodi della porta, né il contratto del dispatcher.
`phase-05-usecases/` resta il deliverable CP5 pulito e in memoria.

---

## Restituzione

Lo script parlato per la restituzione (~4 min) è in [`restitution-IT.md`](./restitution-IT.md)
(inglese: [`restitution.md`](./restitution.md)): wiring dual-write, unificazione del sentinel,
paginazione `{data,meta}`, deprecation RFC 8594, e il disclaimer di fase inventata.

---

## Fase successiva

→ **CP7** aggiunge l'**auth**: JWT / token M2M sul bordo HTTP, e la policy di autorizzazione (OPA). Da lì
CP8–CP10 portano eventi verso Hermes, transactional outbox reale e observability.

---

*MIC v0.1 - by Mic (Michele Mondora) - 2026 · Fase 06 inventata dal Gruppo 3*
