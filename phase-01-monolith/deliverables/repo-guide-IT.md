# MIC — Guida al repo

> Cos'è e cosa fa ogni file/cartella di primo livello di `phase-01-monolith/`, e dove guardare per cambiare le cose.
> Un nuovo arrivato dovrebbe riuscire a trovare il file giusto da questa sola pagina.

## Primo livello di `phase-01-monolith/`

| Path | Cos'è |
|---|---|
| `README.md` / `README-IT.md` | Il brief del lab: la tua missione, i 5 deliverable, come avviare/esplorare. Parti da qui. |
| `docker-compose.yml` | Lo stack a 3 servizi: `mic-app`, `mysql`, `adminer`. Porte, volume `mic_db_data`, mount di init per schema/seed. |
| `Dockerfile` | Costruisce l'immagine unica `mic-app`: `php:8.2-fpm-alpine` + nginx + supervisor + `pdo_mysql`. Genera `supervisord.conf` inline. |
| `nginx.conf` | Config del web server: serve `/css` `/js` staticamente, manda tutto il resto a PHP-FPM via fastcgi. |
| `openapi.yaml` | Il contratto REST legacy (~500 righe) per le rotte `/api/*`. Riferimento per le forme richiesta/risposta. |
| `database/schema.sql` | **Tutto il modello dati**: le due tabelle generiche `business_data` + `business_relations` e i loro indici. Leggilo per primo per capire lo storage. |
| `database/seed.sql` | ~6.7k righe di `INSERT`: dati demo realistici per ogni record type (clienti, articoli, listini, ordini, fatture, movimenti, …). |
| `php-app/` | L'applicazione (vedi sotto). |
| `deliverables/` | **Questa analisi** (i 5 artefatti della Fase 01). Non fa parte dell'app in esecuzione. |

## `php-app/` — l'applicazione

```
php-app/
├── index.php              ← front controller: autoloader, serving statico, tabella rotte API, dispatch
├── src/
│   ├── Router.php         ← router a regex (/api/foo/:id/bar → handler)
│   ├── Database.php       ← PDO singleton, con retry mentre MySQL si scalda
│   ├── Repository.php     ← IL data layer generico sulle due tabelle (query/write/relations/raw)
│   ├── Models/
│   │   ├── BusinessData.php      ← envelope di riga generico (id, code, name, amount_1..4, text_1..5, ...)
│   │   └── BusinessRelation.php  ← arco generico (source_id, target_id, relation_type, amount, metadata)
│   └── Controllers/       ← 17 controller di entità + BaseController (vedi tabella)
└── public/               ← la SPA (servita staticamente)
    ├── index.html        ← shell: sidebar nav + topbar + <main id="view"> + uno <script> per schermata
    ├── css/style.css     ← tutto lo stile
    └── js/
        ├── app.js        ← il "framework" fatto in casa: router, wrapper fetch, listView, buildForm, modal, toast
        └── <entità>.js   ← uno per schermata, ciascuno chiama MIC.registerView(...)
```

### Il cuore PHP (leggi questi per capire qualsiasi richiesta)

- **`index.php`** — fa il parsing di metodo+URI; `/css|/js` → file statico; `/api/*` → `handleApi()`; tutto il resto → shell SPA. `handleApi()` costruisce **l'intera tabella delle rotte a ogni richiesta** (`index.php:84-135`) usando una closure `$crud` generica più sotto-rotte custom, poi dispatcha.
- **`Router.php`** — converte i segmenti `:name` in una regex con gruppi nominati e matcha metodo+path. Restituisce `[status, body]` o `null` (404).
- **`Database.php`** — un unico `PDO` (singleton), con loop di retry ×30 perché l'app sopravviva al cold start di MySQL (`Database.php:30-47`).
- **`Repository.php`** — **il cuore dello strato di storage.** Ogni controller passa di qui. Fornisce `findByType` (filtro + paginazione + search, con whitelist di colonne), `findById`, `create`, `update`, `delete`, e gli helper sulle relazioni (`relate`, `targetIdOf`, `relationsFrom/To`), più le escape hatch `rawAll`/`rawOne` che i controller usano per le join cross-entità.
- **`Models/BusinessData.php` + `BusinessRelation.php`** — DTO "stupidi" che mappano una riga del DB ↔ oggetto. Nessun comportamento; le colonne generiche qui sono volutamente non tipizzate.

### Controller (`src/Controllers/`)

Tutti estendono `BaseController` (CRUD JSON generico via `index/show/create/update/destroy`). Ogni controller concreto dichiara il suo `record_type` e i due metodi di traduzione: `toDto()` (colonne generiche → JSON amichevole) e `fromPayload()` (JSON amichevole → colonne generiche). **È qui che vive il significato di `amount_N`/`text_N`.**

| Controller | `record_type` | Base rotta | Notevole oltre al CRUD |
|---|---|---|---|
| `BaseController` | — | — | CRUD generico + `readJsonBody`; l'80% condiviso. |
| `DashboardController` | — (legge molti) | `/api/dashboard/kpi` | SQL di aggregazione cross-type; **non** usa `BaseController`. |
| `ArticleController` | `articolo` | `/api/articles` | **Target del BC Warehouse.** Articolo = SKU + prezzo base (`amount_1`) + codice IVA default (`text_2`) + qta min. |
| `CategoryController` | `categoria` | `/api/categories` | CRUD semplice. |
| `IvaController` | `aliquota_iva` | `/api/iva-rates` | % IVA; cercata per codice dalle righe d'ordine. |
| `ListinoController` | `listino` | `/api/listini` | `voci(:id)` / `addVoce(:id)` → voci di listino collegate agli articoli. |
| `CustomerController` | `cliente` | `/api/customers` | `orders(:id)` / `invoices(:id)` — riusano Order/Invoice controller per il render. |
| `SupplierController` | `fornitore` | `/api/suppliers` | CRUD semplice; nessun flusso a valle. |
| `OrderController` | `ordine` | `/api/orders` | `righe`/`addRiga` calcolano prezzo+IVA+totali; **entrano nelle righe di articolo + IVA** (l'accoppiamento principale). |
| `ScontoController` | `sconto` | `/api/sconti` | CRUD semplice. |
| `AgenteController` | `agente` | `/api/agenti` | CRUD semplice; collegabile agli ordini. |
| `MagazzinoController` | `magazzino` | `/api/magazzini` | `giacenze(:id)` deriva lo stock come `SUM(movimenti)`. **Target del BC Warehouse.** |
| `MovimentoController` | `movimento` | `/api/movimenti` | `create` cabla `movimento_di_articolo` + `movimento_in_magazzino`. |
| `InvoiceController` | `fattura` | `/api/invoices` | `inviaSdi(:id)` mock; collega a cliente + ordine. |
| `NotaCreditoController` | `nota_credito` | `/api/note-credito` | Solo API, nessuna schermata. |
| `PagamentoController` | `pagamento` | `/api/pagamenti` | Collega a una fattura. |
| `UserController` | `utente` | `/api/users` | Hash password placeholder; nessuna auth. |
| `AuditController` | `audit_log` | `/api/audit-log` | Sola lettura; emerge nella dashboard. |

### Il frontend (`public/js/`)

- **`app.js`** — leggilo una volta e il resto è ovvio. Espone il globale `MIC` con: l'hash router (`navigate`/`boot`), il wrapper fetch `api/get/post/put/del`, e i tre mattoni che ogni schermata riusa — `listView` (tabella paginata da un endpoint di lista), `buildForm` (form da una field-spec) e `modal`/`toast`/`confirmDialog`.
- **`dashboard.js`** — KPI + grafici Chart.js. L'unica schermata non-CRUD.
- **`articles.js`, `customers.js`, `suppliers.js`, `listini.js`, `orders.js`, `invoices.js`, `magazzino.js`, `sconti.js`, `agenti.js`, `settings.js`** — una schermata ciascuno (`settings.js` raggruppa Categorie + IVA + Utenti). `orders.js`, `invoices.js`, `listini.js`, `magazzino.js`, `customers.js` hanno modali di dettaglio più ricchi (sotto-risorse, azioni).

## Cheat-sheet "dove guardo per…"

| Voglio… | Vai a |
|---|---|
| Capire come sono salvati i dati | `database/schema.sql` → `Repository.php` |
| Sapere cosa significa una colonna per un'entità | `toDto()` / `fromPayload()` di quel controller |
| Aggiungere/tracciare una rotta API | `php-app/index.php` (`handleApi`) → il metodo del controller |
| Aggiungere una nuova entità | nuovo `XController extends BaseController` (definisci `type/toDto/fromPayload`) + registra il CRUD in `index.php` + nuovo `public/js/x.js` + aggiungi un link `<nav>` + `<script>` in `index.html` |
| Cambiare la logica di business (prezzi/IVA/stock/totali) | `OrderController`, `MagazzinoController`, `ListinoController`, `InvoiceController` |
| Cambiare l'aspetto di una schermata | `public/js/<entità>.js` (+ `css/style.css`) |
| Avviare/ispezionare lo stack | `docker-compose.yml`; Adminer su `:8082` |
