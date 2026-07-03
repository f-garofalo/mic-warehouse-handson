# Fase 05 — Warehouse BC: use case e una prima API HTTP

> Versione inglese: [`README.md`](./README.md)

```
 _   _            ____
| | | |___  ___  / ___|__ _ ___  ___  ___
| | | / __|/ _ \| |   / _` / __|/ _ \/ __|
| |_| \__ \  __/| |__| (_| \__ \  __/\__ \
 \___/|___/\___| \____\__,_|___/\___||___/

  Fase 05 — I workflow applicativi, esposti via HTTP.
```

Tempo stimato: ~1h nelle breakout room (un loop completo; il secondo è opzionale), poi una restituzione condivisa.

```text
CP3    Domain layer + porta del repository
CP4    Adapter reali + dual-write
CP5    Use case + una sottile API HTTP sopra   <-- sei qui
CP6+   Il layer HTTP completo: auth, list, wiring dual-write, deprecation
```

---

## La tua missione

CP4 ha dato la persistenza al BC. Ora costruisci il layer sopra e lo **esponi via HTTP**: gli **use
case** (workflow applicativi) più un sottile **layer HTTP** che mappa le richieste su di essi. Alla fine
hai un'API funzionante che puoi testare dal browser.

Ci sono **tre use case. Esattamente uno è dato** end-to-end come esempio svolto (`CreateArticle`, da
`POST /articles` fino al dominio). Tu porti **un altro use case per tutto il giro del loop** — e, solo se
hai tempo, un **secondo (opzionale)**.

```
I TRE USE CASE
  create_article.go        CreateArticleUseCase        ✔ DATO    (l'unico esempio svolto)
  get_article.go           GetArticleUseCase           → LOOP A  (fai questo per intero)
  change_article_price.go  ChangeArticlePriceUseCase   → LOOP B  (opzionale, se hai tempo)

LAYER HTTP
  handlers/article_handler.go   handler CreateArticle DATO; handler Get / ChangePrice → COSTRUISCI
  handlers/router.go            rotta POST /articles DATA; due rotte → COSTRUISCI (decommenta)
  handlers/openapi.go           DATO — serve la Swagger UI su /docs
  main.go                       composition root, cabla già tutto

SCAFFOLDING (dato, non toccare)
  usecases/in_memory_article_repository.go, usecases/mocks.go   fake repository di test
  dispatcher/, entities/, events/, interfaces/                  porte + dominio da CP3/CP4
```

### Il pattern della slice (leggi prima l'esempio dato)

Apri `CreateArticle` end-to-end: è esattamente la forma che segue il tuo loop.

```text
richiesta HTTP  ->  handler (bind del JSON, chiama lo use case, mappa il risultato)
                ->  use case (valida, chiede all'aggregate, Save, Dispatch)
                ->  aggregate (regola di business)  ->  porte repository + dispatcher  ->  risposta JSON
```

Ogni layer fa **un** lavoro: l'handler traduce solo l'HTTP, lo use case orchestra, l'aggregate possiede
le regole.

---

## La consegna — un loop completo, poi (opzionalmente) il secondo

Porta **un use case per tutto questo loop in quattro passi prima di iniziare il successivo**:

| Passo | Cosa | Perché |
|---|---|---|
| **1. Codice** | implementa lo use case (`…Execute`) | il workflow vero e proprio |
| **2. Test Go** | fai diventare verde il suo `*_test.go` | **testa SUBITO** — feedback istantaneo mentre scrivi, senza server |
| **3. API** | implementa l'handler + decommenta la rotta | esponilo via HTTP |
| **4. Test API** | prova l'endpoint dal vivo su **`/docs`** | **black-box** — chiunque può farlo, senza saper leggere Go |

### ▸ Loop A — GetArticle · `GET /articles/:id`  (lo fanno tutti)

1. **Codice** — `GetArticleUseCase.Execute` in `usecases/get_article.go`: rifiuta un `ID` vuoto, `repo.FindByID`, avvolgi con `fmt.Errorf("GetArticle: %w", err)`.
2. **Test Go** — `usecases/get_article_test.go` verde.
3. **API** — `ArticleHandler.GetArticle`: leggi `c.Param("id")`, chiama lo use case, **404** su `usecases.ErrArticleNotFound`, altrimenti **200**. Poi decommenta `GET /articles/:id` in `router.go`.
4. **Test API** — su `/docs`: crea un articolo (POST), poi fai GET (atteso 200 + l'articolo); GET di un id inesistente (atteso 404).

### ▸ Loop B — ChangeArticlePrice · `PUT /articles/:id/price`  (opzionale, se hai tempo)

1. **Codice** — `ChangeArticlePriceUseCase.Execute`: valida, `FindByID`, `NewMoney`, `Article.ChangePrice`, **no-op se il prezzo non è cambiato**, altrimenti `Save`, dispatcha `events.ArticlePriceChanged`, `ClearPendingEvents`. Le regole stanno nell'aggregate; i fallimenti non dispatchano mai.
2. **Test Go** — `usecases/change_article_price_test.go` verde.
3. **API** — `ArticleHandler.ChangeArticlePrice`: bind di `ChangePriceRequest`, chiama con `ArticleID = c.Param("id")`, **200 / 400**. Poi decommenta `PUT /articles/:id/price`.
4. **Test API** — su `/docs`: cambia un prezzo (atteso 200 + il nuovo prezzo); reinvia lo **stesso** prezzo (ancora 200, no-op).

### ▸ Fatto quando

- `docker compose run --rm test` è verde per lo/gli use case che hai completato, **e**
- l'endpoint funziona da `/docs` end-to-end. Il Loop A da solo è un risultato completo: un'API con
  `POST /articles` e `GET /articles/:id`.

---

## Esecuzione e test dell'API

```bash
cd ../phase-04-db && docker compose down          # libera la porta 8081
cd ../phase-05-usecases
docker compose up --build                          # serve su :8081
```

**Testala dal browser — niente curl, niente Go:** apri **http://localhost:8081/docs**. È una Swagger UI
(come il `/docs` di FastAPI): scegli un endpoint, clicca **Try it out**, compila il body, **Execute**,
leggi risposta e status code. `POST /articles` funziona da subito; `GET` e `PUT` si accendono man mano
che li costruisci.

<details>
<summary>Preferisci il terminale? Le stesse tre chiamate con <code>curl</code></summary>

```bash
curl -X POST localhost:8081/articles -H 'Content-Type: application/json' \
  -d '{"id":"a1","sku":"ABC-001","name":"Widget","price_cents":1000,"currency":"EUR"}'
curl localhost:8081/articles/a1
curl -X PUT localhost:8081/articles/a1/price -H 'Content-Type: application/json' \
  -d '{"price_cents":1500,"currency":"EUR"}'
```

</details>

I dati sono in memoria: vivono per la durata del run e si azzerano al riavvio (la persistenza era il
compito di CP4).

---

## Scope: cosa abbiamo lasciato FUORI dal layer HTTP di proposito

Questo layer HTTP è una versione **ridotta** di quello di CP6. Di proposito **non** ha — e tu **non** devi
aggiungere:

| Lasciato fuori qui | Arriva in |
|---|---|
| Auth / JWT / token M2M | CP6–CP7 |
| Middleware oltre logger + recover | CP6 |
| `List` + paginazione | CP6 |
| Wiring dual-write / MySQL (qui solo in memoria) | già CP4; cablato all'HTTP in CP6 |
| Framework di validazione, gestione dei sentinel error tra repository | CP6 |

Inoltre non toccare `entities/`, `events/`, `interfaces/`, `dispatcher/` o il file dato
`handlers/openapi.go`. Se l'AI mette mano al dominio o a una porta mentre completi un loop, **fermati** —
è il layer sbagliato.

---

## Come lavorare

Piccole breakout room, l'agente AI come motore, **un use case per tutto il giro prima del successivo**:

```text
codice -> test Go (verde) -> handler -> rotta -> test API su /docs
```

Tieni ogni diff confinato al file che stai completando (`git diff`). Quando il Go non è chiaro, chiedi
all'AI di rispiegartelo in un linguaggio che conosci invece di scrivere altro codice.

> **Prima Save, poi dispatch; i fallimenti non dispatchano.** Nel Loop B, se `Save` riesce ma `Dispatch`
> fallisce, lo stato è cambiato ma l'evento è perso: un gap reale lasciato aperto di proposito, chiuso più
> avanti dal lavoro su transactional outbox / Hermes (CP9). Notalo; non risolverlo qui.

---

## Soluzioni

Implementazioni di riferimento in [`solutions/`](./solutions/): `get_article.expected.go.txt`,
`change_article_price.expected.go.txt`, `article_handler.expected.go.txt`, `router.expected.go.txt`,
con la spiegazione in [`solutions/README-IT.md`](./solutions/README-IT.md). **Aprile solo dopo che il tuo
loop è verde** — checklist, non sorgente da copiare.

---

## Fase successiva

→ La **Fase 06** trasforma questo nel **layer HTTP completo**: `List` + paginazione, cablaggio degli
handler sul repository dual-write reale, auth, e la policy di deprecation per i breaking change dell'API.

---

*MIC v0.1 - by Mic (Michele Mondora) - 2026*
