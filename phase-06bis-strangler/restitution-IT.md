# Restituzione — Fase 06bis: il facade Strangler (Gruppo 3)

> English version: [`restitution.md`](./restitution.md)
> Script parlato per la restituzione di CP6bis (~4 min) + risposte alle domande Q1–Q8.
> ⚠️ **Vincolo del lab rispettato**: l'unico codice cambiato è la funzione `decideUpstream`
> in `mic-integration/facade/main.go` (Part 1 + Part 3) e la rimozione di un `t.Skip` nel test.
> Nient'altro (adapter, monolite, BC, schemi DB, tabelle dei test) è stato toccato.

## Apertura
Siamo il Gruppo 3. Attenzione a non confondere questa fase con la nostra `phase-06-http`: là avevamo
reso **reale** il layer HTTP del BC (persistenza dual-write, paginazione, deprecation). Qui il BC e il
dual-write di CP4 sono **dati per scontati, in produzione**. CP6bis riguarda un problema diverso:
**migrare il traffico** dal monolite PHP di MIC al nuovo BC, una operazione alla volta, in modo sicuro e
reversibile — il pattern **Strangler Fig**.

Il pezzo mancante è il **facade**: un servizio che, richiesta per richiesta, decide *chi risponde*. Tutta
la meccanica (reverse proxy, header `X-Strangler-Route`, i due canali di forwarding) era già scritta.
Il nostro compito era **una sola funzione**: `decideUpstream(method, mode)`.

## Part 1 — La decisione di routing
La regola: solo la **lista articoli** (`GET /api/articles`) è migrata. Con `ROUTE_MODE=warehouse-bc` la
`GET` va all'upstream `warehouse-bc`; con `legacy` — o con **qualsiasi valore inatteso** — resta sul
monolite. Ogni altro metodo → monolite (non ancora migrato).

Il punto di design è il **fallthrough fail-safe**: un `ROUTE_MODE` scritto male non deve diventare un
outage. Un router che "fallisce verso il nuovo sistema" trasforma un typo in un incidente; noi falliamo
**verso il vecchio**. E poiché il `READ_MODE` del BC è ancora `legacy`, instradare la lettura è una
**migrazione di puro traffico**: la sorgente dati non cambia, l'unico rischio è di disponibilità, e la
manopola del facade è il suo rollback.

## Part 2 — Due mondi, una sola app; poi rompiamo la verità
Cliccando in MIC con la Network aperta: la **lista** risponde `X-Strangler-Route: warehouse-bc`, il
**dettaglio** di un articolo e ogni altra schermata (clienti, fatture) rispondono `monolith`. Stessa app,
due mondi, rotta per rotta.

Poi l'incidente: fermiamo il `warehouse-bc` → la lista va in **502** anche se il suo dato vive ancora nel
DB legacy. È la **dipendenza di disponibilità** che abbiamo aggiunto. Rollback con la sola manopola del
facade (`ROUTE_MODE=legacy`, ricreare) **mentre il BC è ancora giù**: la lista torna. Nessun deploy,
nessun tocco a MIC.

Infine rompiamo la verità: modifichiamo un articolo dalla UI (l'update **non** è migrato → finisce solo nel
DB legacy), poi giriamo la **seconda manopola** `READ_MODE=warehouse`. La lista ora legge `warehouse_db`
(riempito col backfill, che nessun write path alimenta): la modifica **non c'è**. Il dettaglio, non
migrato, legge il legacy: la modifica **c'è**. Stesso articolo, due verità, a un click di distanza.

## Part 3 — Il cutover della create
La cura alla staleness delle scritture è farle arrivare a **entrambi** gli store — ed è esattamente il
dual-write di CP4. Abbiamo esteso il dial: ora **GET e POST** seguono `ROUTE_MODE`; update e delete
restano sul monolite. Rimosso il `t.Skip`, l'intera suite è verde.

Una create dalla UI ora viaggia facade → adapter → BC → dual-write: **legacy per primo** (che conia l'id),
poi warehouse **con lo stesso id**. Il nuovo articolo appare subito nella lista migrata (niente staleness
sulle create) e in Adminer si vede lo stesso id in `mic.business_data` **e** in `warehouse_db.articles`.

## Risposte per la restituzione (Q1–Q8)
- **Q1 — Quale URL continua a chiamare la UI?** Sempre `/api/articles` (vedi `mic-monolith/php-app/public/js/articles.js`). È questo contratto stabile del browser a rendere sicura la mossa Strangler.
- **Q2 — Cosa decide ciascuna delle due manopole?** `ROUTE_MODE` (facade) = *chi serve la rotta* (monolite o BC). `READ_MODE` (BC) = *quale store è la verità* (`mic` legacy via ACL o `warehouse_db`). Sono due layer indipendenti: traffico e dati.
- **Q3 — Cosa abbiamo strangolato finora?** Una operazione alla volta: la **lettura lista** (Part 1) e la **create** (Part 3). Non una schermata, non un modulo CRUD: una coppia rotta+metodo per volta.
- **Q4 — Dove è finita la modifica, e perché lista e dettaglio non concordano?** Solo nel DB legacy (l'update non è migrato). La lista migrata legge `warehouse_db`, che nessun write path alimenta ancora; il dettaglio legge il legacy. Due store, due verità: la cura è migrare le scritture, oppure tenere `READ_MODE=legacy` finché non lo si fa.
- **Q5 — Cosa si rompe quando il BC è giù?** Ogni operazione migrata (la lista; dopo Part 3 anche la create). Tutto ciò che appartiene al monolite continua a funzionare: il blast radius è **esattamente** il sottoinsieme migrato — ecco l'argomento per tagliare piccolo.
- **Q6 — Perché il rollback non ha richiesto di toccare MIC?** Perché il contratto del browser non è mai cambiato: MIC chiama sempre `/api/articles`, e l'unica cosa che si è spostata è la decisione del facade. Ritirarsi è una mossa di manopola, non un deploy — e funziona proprio quando il nuovo sistema è in fiamme.
- **Q7 — Perché il dual-write scrive prima il legacy?** Perché il legacy è il system of record e non deve mai restare indietro; e concretamente **conia l'identità** (l'id è l'autoincrement di `business_data`, assegnato sull'insert legacy e poi riusato per la riga warehouse). Non si può scrivere prima il warehouse: non se ne conosce ancora l'id.
- **Q8 — Le create sono sicure: possiamo lasciare `READ_MODE=warehouse`?** Non ancora. Update e delete fluiscono ancora solo verso il legacy: ogni edit riapre il gap. O si migrano anche quelle scritture, o si tengono le letture su `legacy`. Una migrazione è finita per una risorsa solo quando **ogni** write path è migrato.

## Come l'abbiamo dimostrato
- **Test del facade** (`go test ./...` in `mic-integration/facade`, self-contained, senza Docker): verdi
  la tabella di Part 1, la tabella di Part 3 (skip rimosso) e il test di scope HTTP `X-Strangler-Route`.
- **Diff minimale**: `git diff` tocca solo `decideUpstream` e la riga `t.Skip` rimossa — come richiede la checklist del lab.
- **Runtime (Part 2/3)**: le demo con lo stack Docker (flip delle manopole, incidente+rollback, staleness,
  same-id in Adminer) si eseguono col `docker-compose.strangler.yml` — vanno fatte in aula dove Docker è disponibile.

## Come si esegue
```bash
cd phase-06bis-strangler/mic-integration
# codice (senza Docker):
( cd facade && go test ./... )                                                   # suite verde
# stack completo (in aula, con Docker):
docker compose -f docker-compose.strangler.yml up -d --build
docker compose -f docker-compose.strangler.yml --profile tools run --rm backfill-articles
docker compose -f docker-compose.strangler.yml --profile test run --rm facade-test
ROUTE_MODE=warehouse-bc docker compose -f docker-compose.strangler.yml up -d --build strangler-facade
curl -i "http://localhost:8081/api/articles?limit=3"                             # X-Strangler-Route: warehouse-bc
docker compose -f docker-compose.strangler.yml down
```

## Chiusura
Abbiamo aggiunto il punto di controllo della migrazione **tra il client e i backend**. La lista, e poi la
create, attraversano ora il BC; il resto resta sul monolite finché non arriva il suo turno. La ritirata è
una manopola, non un deploy. Il lavoro rimanente (auth, request context, correlation) è CP7. Domande?

## Glossario
- **Strangler Fig** — si avvolge il legacy con un facade e si migra rotta per rotta, finché il vecchio è vuoto e si può spegnere.
- **Facade** — il servizio che decide, per ogni richiesta, chi risponde; unico punto in cui vive la decisione di cutover.
- **ROUTE_MODE / READ_MODE** — le due manopole: *chi serve la rotta* vs *quale store è la verità*.
- **Fail-safe** — un `ROUTE_MODE` inatteso instrada verso il vecchio sistema: un typo non diventa un outage.
- **Staleness** — leggere da uno store che nessun write path aggiorna: lista e dettaglio divergono.
- **Same-id dual-write** — il legacy conia l'id sull'insert, il warehouse riusa lo stesso id: una sola identità in due store.
