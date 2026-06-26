# Restitution — Breakout 2: il domain layer del Warehouse BC (Gruppo 3)

> Script parlato per la restitution di CP3 (~4 min). Confrontiamo la *forma* del modello, non la sintassi.
> Le decisioni dettagliate stanno in [`design-decisions.md`](./design-decisions.md) / [`-IT.md`](./design-decisions-IT.md).

## Apertura
Siamo il Gruppo 3. In CP1 abbiamo fatto **knowledge crunching** sul monolite e ne abbiamo mappato gli accoppiamenti; in CP2 abbiamo scelto il **Bounded Context** Warehouse come prima fetta della **Strangler Fig**. Qui presentiamo il suo **domain layer** in Go: **DDD tattico** puro, niente infrastruttura. La sintassi è dettaglio — difendiamo il *modello*.

## Cosa abbiamo costruito — DDD tattico
Abbiamo mappato la **Ubiquitous Language** del Warehouse sui building block tattici:
- **Article** = **Aggregate Root**, unico entry point e unico **consistency boundary**;
- **SKU** e **Money** = **Value Object** (immutabili, *equality by value*);
- **InventoryLevel** = **Entity** interna all'aggregato;
- i **Domain Event** in past tense;
- un solo **Repository port**.

Tre package — `entities`, `events`, `interfaces` — e nient'altro.

## Gli invarianti dentro il boundary
L'aggregato è il posto dove gli **invarianti** vivono e vengono fatti rispettare *a ogni istante*: `reserved ≤ quantity`, `quantity ≥ 0`, prezzo > 0, valuta stabile dopo la creazione. Nel monolite la giacenza era una `SUM` di movimenti che poteva andare negativa — **nessun invariante**. Qui lo stato illegale è *irrappresentabile*: i **Value Object** validano in una **factory** *fail-fast*, e l'`InventoryLevel` si tocca **solo** tramite l'Article. E `Money` è in **centesimi interi**, non float: niente drift di arrotondamento.

## L'angolo EDA — record, don't publish
Sul fronte **event-driven**: l'aggregato **registra** i fatti (`ArticleCreated`, `ArticlePriceChanged`, `InventoryAdjusted`, `StockReserved`) su una lista pending, e un layer esterno li drena con `PullEvents` **dopo** che il `Save` è andato a buon fine. È il pattern **transactional outbox** in nuce: separi la mutazione di stato dalla pubblicazione, così eviti il **dual-write problem** tra DB e broker. Attenzione: **non è event sourcing** — lo stato resta la *fonte di verità*, gli eventi sono fatti da propagare. Da lì i BC a valle (Ordini, Pricing) reagiscono in **eventual consistency**, in stile **EDA**. Punto chiave: gli eventi si *registrano*, non si *pubblicano* dentro il dominio.

## Clean architecture / hexagonal
Architetturalmente è **Ports & Adapters**: `ArticleRepository` è una **porta** definita in termini di dominio, con **dependency inversion** — l'adapter MySQL la implementerà in CP4 senza che il dominio lo sappia. Un solo port a livello di aggregato e **nessun `InventoryRepository`** (sarebbe una violazione dell'aggregate boundary). Il dominio non importa né web framework né driver DB; l'unico tocco di "fuori" è un `/health`, isolato in `main.go`.

## Due decisioni non ovvie
- **ID iniettati**, non generati nel dominio → dominio *puro e deterministico*, test riproducibili.
- **Creazione vs reconstitution**: `NewArticle` emette `ArticleCreated`, ma `RehydrateArticle` (rehydrate dell'aggregato dal repository) **non** ri-emette eventi. Caricare non è un fatto di business.

## Come l'abbiamo dimostrato
Gli invarianti *sono* la specifica. Per ognuno c'è uno **unit test** sul dominio (senza adapter, regola clean-arch): boundary di SKU/Money, fail-fast della factory, currency-change rifiutato, `reserved ≤ quantity`, eventi in ordine e drenati una volta, Save idempotente. **`go test ./...` tutto verde** — *green means the rules hold*.

## Come eseguire i test
Serve solo Go installato (1.22+); niente Docker, niente DB — il dominio si testa in isolamento.

```bash
cd phase-03-skeleton

go test ./...                                  # tutti i test (sintetico)
go test ./... -v                               # verboso: ogni invariante, uno a uno
go test ./entities/ -run TestReserveStock -v   # una singola regola (reserved <= quantity)
go test ./... -cover                           # copertura

go vet ./... && gofmt -l .                     # niente warning, niente file non formattati

go run .                                       # avvia il servizio, poi in un altro terminale:
curl -s localhost:8081/health                  # -> {"status":"ok"}
```

Output atteso di `go test ./...`:
```
ok      warehouse.local/core/entities     0.9s
?       warehouse.local/core/events       [no test files]
ok      warehouse.local/core/interfaces   0.6s
```

> Il README di CP3 cita anche `docker compose run --rm test`, ma quel servizio `test` è nel compose della *soluzione di riferimento*. Nel nostro repo abbiamo lasciato il `docker-compose.yml` del monolite PHP e **verifichiamo con Go locale**; il wiring Docker del servizio Go è rimandato (CP4+).

## Aggancio a CP1 + chiusura
Si chiude il cerchio: questi invarianti sono la *risposta uno-a-uno* agli antipattern del monolite — `Money` con valuta, giacenza non più negativa, dominio senza logica sparsa nei controller. L'accoppiamento **Ordini→Articolo** trovato in CP1 si romperà con un **Anti-Corruption Layer** e una **Published Language** alla cucitura (context mapping). Tutto è su GitHub con un `design-decisions.md` (EN+IT) che motiva ogni scelta. Prossimo step: persistenza + **dual-write** — la cucitura si accende.

## Glossario lampo
- **Aggregate Root** — unità di consistenza, unico entry point.
- **Value Object** — identità per valore, immutabile, validato in factory.
- **Entity** — identità propria, mutata solo via l'aggregato.
- **Domain Event** — fatto al passato, *registrato* non pubblicato.
- **Repository (port)** — porta di persistenza dell'aggregato (Ports & Adapters).
- **record-then-publish** — pattern *transactional outbox* (evita il dual-write).
- **EDA / eventual consistency** — i BC a valle reagiscono agli eventi in modo asincrono.
- **Strangler Fig** — estrazione incrementale del BC dal monolite.
- **ACL / Published Language** — come si rompe l'accoppiamento alla cucitura tra BC.
