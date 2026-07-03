# Restituzione — Fase 06: il layer HTTP completo (Gruppo 3)

> English version: [`restitution.md`](./restitution.md)
> Script parlato per la restituzione di CP6 (~4 min). Confrontiamo la *forma*, non la sintassi.
> ⚠️ **Fase inventata**: il lab si ferma a `lezione-8-fase-5`. CP6 l'abbiamo costruita noi dalla
> descrizione del README di CP5. **Auth esclusa di proposito** (è CP7).

## Apertura
Siamo il Gruppo 3. CP5 ha dato al BC gli use case e un sottile layer HTTP — ma su un fake **in memoria**.
CP6 lo rende **reale**: gli stessi endpoint ora persistono attraverso il repository **dual-write** di CP4
(due MySQL), guadagnano la **paginazione** e una **policy di deprecation**. Vive in una nuova cartella
`phase-06-http/`, assemblata da CP5 (use case + HTTP) + CP4 (repositories + i due database).

## Accendere la cucitura — wiring dual-write
Il composition root (`main.go`) apre le connessioni legacy e warehouse MySQL da env e inietta il
`DualWriteArticleRepository` di CP4 in **tutti** gli use case. Così un `POST`/`PUT` dal layer HTTP ora
atterra in **entrambi** gli store (legacy per primo), e le letture arrivano da **uno** store scelto da
`DUAL_WRITE_READ_MODE` (`legacy` oggi, `bc` dopo il cutover). Il fake in memoria sparisce dal servizio in
esecuzione — resta solo come test double. La cucitura progettata da CP3 a CP5 è ora davvero attiva.

## L'unificazione del sentinel (la ruga di CP6)
CP4 e CP5 dichiaravano ciascuna il proprio `ErrArticleNotFound` (uno in `repositories`, uno in
`usecases`). Collegati, un not-found dal repo **reale** non avrebbe matchato il controllo `errors.Is`
dell'handler → un articolo mancante avrebbe risposto 400 invece di 404. CP6 lo risolve in modo pulito: il
sentinel si sposta nella **porta** (`interfaces.ErrArticleNotFound`); ogni adapter e gli use case lo
**aliasano**. Ora il not-found risale dal dual-write fino a un **404**. (È esattamente l'"error-sentinel
juggling" che il README di CP5 rimandava a CP6.)

## Paginazione
`GET /articles?limit=&offset=` restituisce `{ "data": [...], "meta": { total, limit, offset } }` — lo
**stesso envelope del monolite MIC** di CP1, un callback voluto. I valori sono clampati (default 50, max
200, offset ≥ 0). La paginazione è applicata a livello **use case** (fetch-all + slice); il
`LIMIT/OFFSET` lato SQL sarebbe un'ottimizzazione successiva che toccherebbe la porta e ogni adapter.

## Policy di deprecation (RFC 8594)
Un piccolo middleware stampa `Deprecation: true`, `Sunset: <data>` e `Link: <successore>;
rel="successor-version"`. Lo avvolgiamo attorno a un **alias deprecato** `GET /v0/articles/:id`
(successore: `/articles/:id`) — una dimostrazione concreta di *come si ritira un endpoint con breaking
change dando preavviso*, non un semplice commento. Compare `deprecated: true` in OpenAPI/Swagger.

## Cosa NON abbiamo fatto di proposito
**Niente auth** (JWT/M2M) — è CP7. Non abbiamo toccato l'aggregate, l'insieme di metodi della porta, né
il contratto del dispatcher. `phase-05-usecases/` resta il deliverable CP5 pulito.

## Come l'abbiamo dimostrato
- **Test unitari** (`go test ./...`, senza DB): tutti i test CP3/CP4/CP5 riportati verdi + i nuovi test di
  paginazione (paginates, defaults/clamp, offset-beyond-total).
- **End-to-end** contro i due MySQL reali: `POST`/`GET`/`PUT` persistono; id mancante → **404 via
  dual-write**; l'articolo **sopravvive al restart del container** (persistenza reale, non in memoria);
  `GET /articles?limit=2&offset=0` ritorna la pagina `{data,meta}`; `GET /v0/articles/:id` porta gli header
  `Deprecation`/`Sunset`/`Link`.

## Come si esegue
```bash
cd phase-06-http
go test ./...                              # tutti i package verdi (senza DB)
docker compose up --build -d               # 2 MySQL + app :8081  ->  http://localhost:8081/docs
#  POST /articles ; GET /articles?limit=&offset= ; GET /articles/:id (200/404) ; PUT price ; GET /v0/articles/:id
docker compose down                        # (nessun volume nominato -> i dati sono freschi a ogni up)
```

## Chiusura
L'estrazione è ora un servizio in esecuzione: HTTP in ingresso, dual-write su due MySQL, letture da un
solo store, liste paginate e una policy di deprecation per far evolvere il contratto. La cucitura che il
monolite MIC esponeva in CP1 è ora servita end-to-end dal BC in Go. Il lavoro "da lab" rimanente (auth,
policy OPA, eventi Hermes, observability) è CP7–CP10. Domande?

## Glossario
- **Wiring dual-write** — gli use case HTTP persistono tramite il decorator di CP4 (entrambi gli store, legacy per primo).
- **Read mode / cutover** — letture da uno store; si passa `legacy → bc` quando il nuovo store è affidabile.
- **Sentinel di porta** — un unico `interfaces.ErrArticleNotFound` che ogni layer matcha (unifica CP4/CP5).
- **Envelope della lista** — `{data, meta}`, la forma del monolite MIC, riusata per la paginazione.
- **Sunset RFC 8594** — header `Deprecation`/`Sunset`/`Link` che annunciano il ritiro di un endpoint.
