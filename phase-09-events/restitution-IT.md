# Restituzione — Fase 09: il fatto lascia il BC (Gruppo 3)

> English version: [`restitution.md`](./restitution.md)
> Script parlato per la restituzione di CP9 (~4 min). Confrontiamo la *forma*, non la sintassi.

## Apertura
Siamo il Gruppo 3. CP7 aveva messo una busta di identità attorno al BC. CP9 fa qualcosa di più drastico:
**ritira il dual-write di CP4**. Non perché fosse rotto — per i bisogni di MIC funzionava — ma perché il
tema di questa fase è l'**integrazione event-driven**, e con il dual-write acceso non c'è nessun gap da
osservare. Così lo scaffolding viene giù: il BC scrive **solo il proprio store**, non sa più che esiste il
database di MIC, e ora **conia da sé gli id degli articoli** (`/health` → `"write_mode": "warehouse-only"`).

Il ritiro apre un buco. La schermata di order entry di MIC non è mai stata migrata: legge le tabelle
legacy, e nessuno le riempie più — un articolo **nato nel BC è invisibile** a MIC. Chiudiamo il buco in
modo event-driven: il BC pubblica un **fatto** (un payload Data Product validato contro un JSON schema,
avvolto in una busta **CloudEvents**), e un consumer dentro MIC costruisce una **projection che possiede
lui**. Il pipeline — map → validate → envelope → keep — è **dato** e funziona già per `InventoryAdjusted`:
è il nostro esempio svolto. Due cose mancavano, entrambe nostre: i due lati del contratto, con lo schema
in mezzo.

## Parte 1 — pubblicare il fatto (Go, `hermes_publisher.go`)
Due punti, ricalcati da `InventoryAdjusted`. **`mapEventToSchema`** (Task 1a): mappa `ArticleCreated` sul
Data Product `warehouse.article.v1` — chiavi dal contratto (`article-entity-record-v1.json`:
`article_id, sku, name, price_cents, currency, updated_at`), valori dall'evento, il timestamp come
RFC3339Nano. L'unico tranello: il campo dell'evento è `ArticleName`, il contratto dice `name`.
**`eventMetadata`** (Task 1b): il `time` della busta è il tempo dell'evento, il suo `subject` è
`articles/<article id>`. Abbiamo **osservato prima il guasto**: senza la fix la create rispondeva **`400`**
e l'errore *nominava il nostro compito* — "il publisher rifiuta di dispatchare un evento di cui non ha
contratto". Dopo la fix: **`201`**, e il fatto è sul filo.

## Parte 2 — consumare il fatto (PHP, `consumer.php`)
Una funzione, `mapHermesArticleToMicProjection` — il loop di polling e l'upsert idempotente sono dati.
Adatta il record CloudEvents alla riga che MIC legge: `code` ← sku, `name` (fallback a sku), `amount_1` =
`centsToDecimal(price_cents)` — una **stringa decimale, mai un float** (è l'Anti-Corruption Layer di CP4 al
contrario: il BC parla in cent interi, MIC in `DECIMAL`), più i default di MIC (`text_1='WAREHOUSE-BC'`,
`text_2='IVA22'`, `status='attivo'`). E un `payload_json` che porta l'**audit trail**: `projection_kind`,
`article_id`, `currency`, e i metadati della busta (`source_record_id`, `source_record_type`, `source`,
`source_subject`, `source_time`) — ogni riga proiettata può nominare il record Hermes esatto da cui viene.
Punto chiave: la riga è `record_type='warehouse_article_projection'`, **non** una finta `'articolo'`
legacy: un read model che MIC possiede, non una seconda fonte di verità.

## Il filo: envelope vs payload
Leggi un record tenendo a mente la divisione. La **busta** (`specversion, id, source, type, subject,
time`) è metadato di routing uniforme che legge la *piattaforma* — per instradare, deduplicare, ordinare,
tracciare — senza sapere cosa sia un articolo. Il **payload** (`data`) è il *Data Product*, il dominio,
versionato come `warehouse.article.v1`, che legge la *logica di business del consumer*. Trasporto vs
significato. Il nostro consumer ha toccato entrambi: filtra per `type` della busta e mappa da `data`.

## Come l'abbiamo dimostrato
- **Test** (`docker compose run --build --rm test`): tutti verdi in-container, inclusi
  `TestHermesPublisher_*` (i casi `ArticleCreated` partivano rossi).
- **Sul filo**: `/health` `warehouse-only`; create → **`201`**, il BC ha coniato `art-bd9a5b0487fe2cf3`;
  `/debug/hermes/records` mostra busta + payload. L'order entry era **vuoto** prima del consumer; dopo un
  passaggio l'articolo compare con `"source": "hermes-projection"`, e il `payload_json` della riga
  `warehouse_article_projection` punta al record Hermes esatto. Col loop consumer acceso, un **secondo**
  articolo è passato **da solo** — e nessuna riga duplicata: l'upsert è idempotente per SKU.
- `git diff` tocca **solo** i due branch `ArticleCreated` e l'unica funzione PHP.

## Le cinque domande (Q1–Q5)
**Q1 — il dual-write teneva MIC perfettamente consistente; perché ritirarlo? Nomina i costi che
nascondeva.** Era consistente, ma a un prezzo. **Accoppiava** il BC allo schema legacy (tabelle, colonne,
`DECIMAL`, la god-table che coniava gli id) — la migrazione non poteva mai finire. Era **sincrono**: MIC
lento o giù bloccava le scritture del BC; la disponibilità del produttore dipendeva da quella del
consumatore. Era **punto-a-punto via DB condiviso**: nessun contratto pubblicato, quindi un secondo
consumer significava una terza scrittura. E **confondeva l'ownership**: due scrittori, nessuna fonte di
verità chiara. Gli eventi sostituiscono tutti e quattro con un unico contratto versionato che N consumer
leggono, in modo asincrono. Lo ritiriamo per rendere visibile il gap — e la disciplina che lo chiude.

**Q2 — envelope vs payload: quali campi legge la piattaforma, quali la logica di business del consumer, e
perché conta la divisione?** La **piattaforma** legge la **busta** — `specversion, id, source, type,
subject, time` — per instradare, deduplicare, ordinare e tracciare, in modo uniforme su ogni tipo di
evento, senza interpretare il contenuto di dominio. La **logica di business del consumer** legge il
**payload** (`data`): i campi di dominio del Data Product. La divisione separa **trasporto e significato**:
il routing non ha bisogno di accoppiarsi al dominio, e il dominio (`warehouse.article.v1`) può essere
versionato ed evolvere senza toccare il routing.

**Q3 — perché `record_type='warehouse_article_projection'` invece di una finta riga `'articolo'` che non
richiederebbe adapter?** Perché il BC ora è l'unica fonte di verità. Una finta `'articolo'` sarebbe un
secondo write model non etichettato, travestito da dati master legacy — la stessa ambiguità del
dual-write — e la logica di MIC potrebbe modificarlo e divergere in silenzio. Etichettarla projection dice
la verità: un **read model che MIC possiede**, derivato dal fatto, eventualmente consistente, mai
modificato sul posto, e **ricostruibile rigiocando gli eventi**. L'adapter è il prezzo di questa onestà.

**Q4 — uccidi il consumer, crea tre articoli: cosa resta consistente, cosa lagga, cosa succede al
riavvio?** **Consistente**: il BC e il suo store — i tre sono salvati e i tre fatti pubblicati; le create
non si bloccano mai sul consumer. **Lagga**: la projection di MIC — l'order entry non li vede ancora. È
**consistenza eventuale**: una finestra di staleness limitata, non perdita di dati. **Al riavvio**: il
consumer fa polling, legge i record che aveva perso, e l'upsert idempotente porta la projection al passo —
i tre compaiono, senza duplicati. Il lag si chiude.

**Q5 — la create fallita ha comunque lasciato una riga in `warehouse_db` (il save è andato, la publish ha
rifiutato): cosa ti dice quel mezzo-write, e cosa servirebbe in produzione?** Espone il **problema del
dual-write della pubblicazione di eventi**: "salvare" e "pubblicare" sono due operazioni senza transazione
condivisa. Se il save committa e la publish fallisce — o il processo muore in mezzo — lo stato è cambiato
ma il fatto non è viaggiato, e il consumer diverge **in silenzio** (peggio del nostro `400` visibile,
perché in produzione la publish potrebbe fallire *dopo* un `201`). La produzione ha bisogno dell'**outbox**:
nella stessa transazione del cambio di stato, scrivi l'evento in una tabella `outbox`; un relay pubblica da
lì con consegna at-least-once. L'atomicità è ripristinata, la consegna garantita, e i consumer idempotenti
(come il nostro) assorbono i duplicati. Nel nostro esercizio la publish è in-process, quindi il mezzo-write
è solo un artefatto didattico — ma nomina esattamente perché i sistemi reali hanno bisogno di un outbox.

## Come si esegue
```bash
cd phase-09-events
docker compose up -d --build                                   # MIC + facade + IAM + BC + consumer + Adminer
docker compose --profile tools run --rm backfill-articles      # lo stock di cui il BC ha preso possesso
docker compose run --build --rm test                           # suite verde (incl. TestHermesPublisher_*)
# Task 1 sul filo:
TOKEN=$(curl -s -X POST http://localhost:9001/oauth/token \
  -d "grant_type=password&username=alice&password=demo&client_id=11111111-2222-4333-8444-555555555555&scope=openid profile" \
  | sed -E 's/.*"access_token":"([^"]+)".*/\1/')
curl -s -X POST http://localhost:8083/articles -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"sku":"ART-GAP-002","name":"Born in the BC","price_cents":990,"currency":"EUR"}'   # 201
curl -s "http://localhost:8083/debug/hermes/records?type=warehouse.article.v1"
# Task 2:
docker compose --profile tools run --rm mic-hermes-consumer-once    # un passaggio; l'order entry ora lo trova
docker compose --profile consumer up -d mic-hermes-consumer         # continuo: un nuovo articolo appare da solo
docker compose --profile consumer down
```
La mappa: `8081` UI MIC (facade) · `8082` Adminer (server `integration-mysql`, root/root) · `8083`
Warehouse BC · `9001` IAM mock.

## Chiusura
Il dual-write ritirato, e gli eventi al suo posto: il BC pubblica un fatto versionato e possiede il suo
store; MIC legge un contratto e possiede la sua projection. Stessa forma delle fasi precedenti — un
contratto in mezzo, un default che fallisce sicuro — un livello più su, e ora asincrona. L'unico debito
che abbiamo nominato è l'outbox; è ciò che rende sicuro il "pubblica dopo aver salvato" in produzione.

## Glossario
- **Data Product** — il payload (`data`) come contratto versionato e validato da schema (`warehouse.article.v1`).
- **Busta CloudEvents** — metadato di routing uniforme (`specversion, id, source, type, subject, time`) attorno a qualsiasi payload.
- **Projection / read model** — copia derivata, read-only, posseduta dal consumer; qui `record_type='warehouse_article_projection'`.
- **Consistenza eventuale** — la projection insegue la fonte con una finestra limitata, poi la raggiunge; non è perdita di dati.
- **Upsert idempotente** — riapplicare lo stesso record non cambia nulla; per SKU, quindi il polling ripetuto non crea duplicati.
- **Outbox** — scrivi l'evento nella stessa transazione del cambio di stato; un relay lo pubblica — la fix del mezzo-write.
- **Anti-Corruption Layer (al contrario)** — `centsToDecimal`: i cent interi del BC tradotti nel `DECIMAL` di MIC, al confine.
