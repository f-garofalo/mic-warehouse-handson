# Warehouse BC — Decisioni di design (CP3)

> English version: [`design-decisions.md`](./design-decisions.md)

Il domain layer Go codifica scelte architetturali, per lo più implicite. Ognuna qui sotto indica la
**decisione**, la **regola/confine che protegge**, l'**alternativa scartata** e **come si allinea al
target clean-architecture** di CP2. Difendiamo questo design, non la sintassi.

Scope: solo il domain layer (`entities/`, `events/`, `interfaces/`) + una health probe priva di
dominio. Persistenza, use case, la superficie HTTP `/articles`, auth, ecc. arrivano nelle fasi
successive.

## 1. `Article` è l'unico aggregate root
- **Decisione.** `Article` è il solo aggregate root; `InventoryLevel` vive *dentro* di lui ed è
  creato/mutato solo tramite i metodi di `Article` (`AdjustInventory`, `ReserveStock`).
- **Regola che protegge.** `reserved ≤ quantity`, `quantity ≥ 0`, "una giacenza per location" —
  invarianti che attraversano articolo e giacenze devono stare in un unico confine di consistenza.
- **Scartata.** Tre aggregati separati (Article, StockReservation, StockMovement). L'Event Storming
  li ha fatti emergere, ma separarli ora spargerebbe gli invarianti di stock su più aggregati senza
  alcuna garanzia transazionale tra loro.
- **Fit clean-arch.** L'aggregate root *è* l'unità di consistenza che use case e persistenza
  caricheranno e salveranno per intero.

## 2. `SKU` e `Money` sono value object senza identità
- **Decisione.** `SKU` e `Money` sono struct immutabili con campi non esportati, confrontati **per
  valore** (`Equals`), costruiti solo tramite factory (`NewSKU`, `NewMoney`) che **falliscono rumorose**.
- **Regola che protegge.** "SKU/Money valido per costruzione" — la validità è garantita al confine,
  così i punti d'uso non rivalidano. `SKU` matcha `^[A-Z0-9-]{3,32}$`; `Money` è `cents ≥ 0` +
  valuta ISO-4217.
- **Scartata.** (a) Validazione nel chiamante — la restitution segnala esplicitamente "validation in
  the caller, not the factory". (b) Identità/ID su SKU o Money — non hanno ciclo di vita; due valori
  uguali sono la stessa cosa.
- **Fit clean-arch.** I value object sono lo strato più interno; tutto il resto si fida di loro.

## 3. `Money` è in centesimi interi, mai un float
- **Decisione.** `Money` è `int64 amountCents` + `string currency`, mai un importo a virgola mobile.
- **Regola che protegge.** Esattezza monetaria — niente deriva di arrotondamento floating su
  prezzi/totali.
- **Scartata.** Prezzo `float64` (l'anti-segnale "price as a floating-point number" della restitution).
- **Fit clean-arch.** Risolve uno dei peggiori antipattern trovati in CP1 (il `DECIMAL` nudo senza
  valuta del monolite): ora il denaro porta con sé la valuta e una rappresentazione esatta.

## 4. Esattamente un repository port; **nessun** `InventoryRepository`
- **Decisione.** `interfaces.ArticleRepository` è l'unico port, con operazioni a livello di aggregato
  (`Save`, `FindByID`, `FindBySKU`, `List`, `Delete`). `Save` è documentato come upsert idempotente
  che **non** pubblica eventi.
- **Regola che protegge.** Il confine dell'aggregato: la giacenza è persistita come parte del suo
  `Article`.
- **Scartata.** Un `InventoryRepository` (o `SKURepository`/`MoneyRepository`). La restitution segnala
  direttamente "an InventoryRepository (breaks the boundary)".
- **Fit clean-arch.** I port puntano verso l'interno, al dominio; l'adapter MySQL (CP4) implementa
  questa interfaccia senza che il dominio ne sappia nulla.

## 5. Gli eventi di dominio sono **registrati, non pubblicati**
- **Decisione.** L'aggregato accoda gli eventi in una lista privata `pendingEvents`; `PullEvents()`
  li drena (restituisce e svuota). Niente nel dominio pubblica o serializza.
- **Regola che protegge.** Un dominio puro e senza side effect: un'operazione muta solo l'aggregato e
  registra un fatto. Ordine/transazionalità della pubblicazione sono una preoccupazione esterna.
- **Scartata.** Pubblicare l'istante in cui l'evento accade (l'anti-segnale "events published the
  moment they happen") — accoppia il dominio a un broker e rompe il "Save poi pubblica".
- **Fit clean-arch.** Gli use case di CP5 faranno `Save` poi draineranno `PullEvents()` e
  dispatcheranno — la cucitura è già qui.

## 6. Gli ID sono iniettati; l'id di `InventoryLevel` è un composito deterministico
- **Decisione.** `NewArticle(id, …)` riceve l'id in input (nessuna generazione nel dominio). L'id di
  `InventoryLevel` è derivato deterministicamente come `articleID + ":" + locationCode`.
- **Regola che protegge.** Un dominio puro e deterministico — niente `crypto/rand`, niente ID guidati
  da clock nascosti; i test sono riproducibili. "Una giacenza per (articolo, location)" cade fuori
  dalla chiave composita.
- **Scartata.** Generare UUID dentro la factory — porterebbe un side effect (casualità) nel dominio e
  fisserebbe prematuramente la strategia di ID; la generazione di UUID spetta allo strato use-case di CP5.
- **Fit clean-arch.** Un UUID assegnato dalla persistenza può sostituire l'id composito in CP4 senza
  cambiare l'API di dominio.

## 7. Creazione vs ricostituzione: `NewArticle` registra, `RehydrateArticle` no
- **Decisione.** `NewArticle` è la factory per un articolo *nuovo* e registra `ArticleCreated`.
  `RehydrateArticle` ricostruisce un articolo da stato persistito e registra **zero** eventi.
- **Regola che protegge.** Gli eventi sono fatti su cose *appena accadute*. Caricare una riga dal DB
  non è una nuova creazione e non deve ri-emettere `ArticleCreated`.
- **Scartata.** Un solo costruttore usato sia per articoli nuovi sia per quelli caricati — o
  ri-emetterebbe eventi di creazione a ogni load, o costringerebbe il repository a rimuovere gli
  eventi a posteriori.
- **Fit clean-arch.** Il repository di CP4 chiamerà `RehydrateArticle`; lo use case chiama
  `NewArticle`. Il confine è esplicito.

## 8. Quattro eventi (incl. `ArticlePriceChanged`); tempo catturato al momento della registrazione
- **Decisione.** Spediamo `ArticleCreated`, `InventoryAdjusted`, `StockReserved` **e**
  `ArticlePriceChanged`. Ogni evento cattura `OccurredAt = time.Now().UTC()` nel suo costruttore; il
  campo è non esportato (immutabile), esposto via `OccurredAt()`.
- **Regola che protegge.** Ogni operazione che cambia stato registra un fatto. `ChangePrice` deve
  registrare *qualcosa*; `ArticlePriceChanged` (con cents vecchio + nuovo) è quel fatto. La tabella
  "3 eventi" delle slide è illustrativa; il mapping tactical-DDD elenca `ArticlePriceChanged` e le
  regole richiedono un `ChangePrice` che registra un fatto.
- **Scartata.** (a) Un `ChangePrice` che muta in silenzio senza evento. (b) Iniettare un'interfaccia
  `Clock` — DI prematura per un BC didattico; i test verificano che `OccurredAt` sia recente, non
  esatto. (c) `StockReservationReleased` — nessuna operazione di rilascio nello scope di CP3.
- **Fit clean-arch.** I payload degli eventi sono solo primitivi, quindi `events/` non importa tipi
  di dominio e resta una foglia di dipendenze — pronto per la serializzazione (CP9) senza cicli.

## 9. Una health probe in `main.go`, tenuta fuori dal dominio
- **Decisione.** `main.go` avvia un server Echo che espone solo `GET /health → {"status":"ok"}` su
  `:8081`. **Non** importa nessun package di dominio.
- **Regola che protegge.** "Ferma al dominio": nessun handler HTTP per `/articles`, nessuna
  persistenza. I package di dominio non importano né web framework né driver DB.
- **Scartata.** Mettere logica di business o wiring del repository in `main.go` ora (è roba di CP4–CP6).
- **Fit clean-arch.** `main` è lo strato più esterno; può dipendere dagli strati interni ma qui non
  dipende da nessuno, così il dominio resta eseguibile-ma-isolato.

---

### Come verificare
```bash
cd phase-03-skeleton
go test ./... -v        # ogni invariante verde
go vet ./... && gofmt -l .
go run .                # poi: curl localhost:8081/health  → {"status":"ok"}
```
