# Restituzione — Fase 05: use case e una sottile API HTTP (Gruppo 3)

> English version: [`restitution.md`](./restitution.md)
> Script parlato per la restituzione di CP5 (~4 min). Confrontiamo la *forma* dei layer, non la sintassi.
> Codice in `usecases/get_article.go` + `change_article_price.go`, `handlers/article_handler.go` + `router.go`.

## Apertura
Siamo il Gruppo 3. CP4 ha dato la persistenza al BC. CP5 aggiunge il layer sopra e lo **espone via
HTTP**: gli **use case** (workflow applicativi) più un **sottile layer HTTP** che mappa le richieste su
di essi. `CreateArticle` era dato end-to-end come esempio svolto; noi abbiamo portato **GetArticle**
(Loop A) e **ChangeArticlePrice** (Loop B) per tutto il giro.

## La vertical slice
Ogni feature è una fetta sottile attraverso i layer, ognuno con **un solo** compito:

```text
handler HTTP (bind del JSON, chiama lo use case, mappa il risultato)
  → use case (valida, chiede all'aggregate, Save, Dispatch)
  → aggregate (regola di business)  → porte (repository + dispatcher)  → JSON
```

L'handler traduce solo l'HTTP; lo use case orchestra; l'**aggregate possiede le regole**.

## Loop A — GetArticle · `GET /articles/:id`
Workflow in sola lettura. Lo use case valida l'id, chiama `repo.FindByID` e **avvolge l'errore con
`%w`** — così l'handler riconosce `ErrArticleNotFound` via `errors.Is` e risponde **404** (contro
**400** per ogni altro errore, **200** in caso di successo). Il punto: il sentinel viaggia *avvolto*, e
il layer che conosce gli status code HTTP è l'handler, non lo use case.

## Loop B — ChangeArticlePrice · `PUT /articles/:id/price`
Workflow che muta lo stato, stessa forma di `CreateArticle`: `FindByID` → `NewMoney` → chiede
all'aggregate di fare `ChangePrice`. La **regola vive nell'aggregate** (stessa valuta, prezzo > 0); lo
use case orchestra soltanto. Due cose da difendere:
- **No-op**: se il nuovo prezzo è uguale a quello attuale, l'aggregate non cambia nulla e non registra
  eventi — quindi lo use case **non** fa `Save` e **non** fa `Dispatch`. Idempotente per design.
- **Save prima di Dispatch, i fallimenti non dispatchano mai**: costruisci il canonico
  `ArticlePriceChanged`, `Save`, poi `Dispatch`, poi `ClearPendingEvents`. Se find/save/dispatch
  fallisce, l'errore emerge e nessun evento esce.

## L'angolo EDA — e un gap onesto
L'aggregate **registra** i fatti; lo use case **dispatcha** l'evento canonico solo **dopo che Save è
riuscito** — record-then-publish, lo spirito dell'*outbox*. I BC a valle reagiscono in **eventual
consistency**. C'è un gap deliberato: se `Save` riesce ma `Dispatch` fallisce, lo stato è cambiato ma
l'evento è perso. L'abbiamo notato, non l'abbiamo nascosto — si chiude più avanti col lavoro su
transactional outbox / Hermes (CP9).

## Clean architecture
**Dependency inversion** ovunque: lo use case dipende dalle **porte** `ArticleRepository` e
`EventDispatcher`, non dalle implementazioni concrete; qui sono fake in memoria, e CP6 le collega al
repository dual-write reale. Il dominio non vede mai JSON; l'handler non vede mai SQL.

## Cosa abbiamo lasciato FUORI di proposito
Questo layer HTTP è una versione **ridotta** di CP6: niente auth/JWT, niente middleware oltre
logger+recover, niente `List`/paginazione, niente wiring dual-write/MySQL, niente framework di
validazione. Abbiamo toccato solo i due use case, i loro handler e il router — **non** il dominio, le
porte, il dispatcher, il dato `openapi.go` o `main.go`.

## Come l'abbiamo dimostrato
- **Test unitari (fake in memoria, senza server):** GetArticle ritorna / fa 404 / rifiuta id vuoto;
  ChangePrice cambia+salva+dispatcha esattamente un evento, è no-op sullo stesso prezzo, rifiuta il
  cambio valuta e fa emergere i fallimenti find/save/dispatch senza dispatchare. Tutti verdi.
- **API dal vivo via Swagger su `/docs`** (o curl): `POST /articles` → 201; `GET` → 200 / 404;
  `PUT .../price` → 200, stesso prezzo → 200 (no-op), cambio valuta → 400.

## Come si esegue
```bash
cd phase-05-usecases
go test ./...                              # use case + dispatcher verdi
docker compose up --build                  # :8081  ->  http://localhost:8081/docs (Swagger UI)
#   POST /articles ; GET /articles/:id ; PUT /articles/:id/price
docker compose down
```

## Chiusura
Il BC ora ha workflow applicativi e una superficie HTTP funzionante: una richiesta scorre handler →
use case → aggregate → porte e torna come JSON, con gli eventi **registrati** e **dispatchati dopo
Save**. Prossimo passo (CP6): il layer HTTP completo — `List` + paginazione, wiring al repository
dual-write reale, auth e la policy di deprecation dell'API. Domande?

## Glossario
- **Use case** — un workflow applicativo; orchestra aggregate + porte, non possiede regole di business.
- **Vertical slice** — una feature attraversata da handler → use case → aggregate → porte.
- **Porta (port)** — un'interfaccia da cui dipende lo use case (`ArticleRepository`, `EventDispatcher`); gli adapter si innestano.
- **Dependency inversion** — i layer interni definiscono le porte; i layer esterni le implementano.
- **Dispatcher** — propaga gli eventi di dominio ai subscriber dopo `Save` (record-then-publish / outbox).
- **No-op idempotente** — reinviare lo stesso prezzo non cambia nulla e non emette eventi.
- **Eventual consistency** — i BC a valle reagiscono agli eventi in modo asincrono.
