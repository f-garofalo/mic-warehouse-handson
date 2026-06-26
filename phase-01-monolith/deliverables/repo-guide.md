# MIC — Repo Guide

> What each top-level file/folder of `phase-01-monolith/` is, and where to look to change things.
> A newcomer should be able to find the right file from this page alone.

## Top level of `phase-01-monolith/`

| Path | What it is |
|---|---|
| `README.md` / `README-IT.md` | The lab brief: your mission, the 5 deliverables, how to run/explore. Start here. |
| `docker-compose.yml` | The 3-service stack: `mic-app`, `mysql`, `adminer`. Ports, the `mic_db_data` volume, and the schema/seed init mounts. |
| `Dockerfile` | Builds the single `mic-app` image: `php:8.2-fpm-alpine` + nginx + supervisor + `pdo_mysql`. Generates `supervisord.conf` inline. |
| `nginx.conf` | Web-server config: serve `/css` `/js` statically, send everything else to PHP-FPM via fastcgi. |
| `openapi.yaml` | The legacy REST contract (~500 lines) for the `/api/*` routes. Reference for request/response shapes. |
| `database/schema.sql` | **The whole data model**: the two generic tables `business_data` + `business_relations` and their indexes. Read this first to understand storage. |
| `database/seed.sql` | ~6.7k lines of `INSERT`s: realistic demo data for every record type (clienti, articoli, listini, ordini, fatture, movimenti, …). |
| `php-app/` | The application (see below). |
| `deliverables/` | **This analysis** (the 5 Phase-01 artifacts). Not part of the running app. |

## `php-app/` — the application

```
php-app/
├── index.php              ← front controller: autoloader, static serving, API route table, dispatch
├── src/
│   ├── Router.php         ← tiny regex router (/api/foo/:id/bar → handler)
│   ├── Database.php       ← PDO singleton, retries while MySQL warms up
│   ├── Repository.php     ← THE generic data layer over both tables (query/write/relations/raw)
│   ├── Models/
│   │   ├── BusinessData.php      ← generic row envelope (id, code, name, amount_1..4, text_1..5, ...)
│   │   └── BusinessRelation.php  ← generic edge (source_id, target_id, relation_type, amount, metadata)
│   └── Controllers/       ← 17 entity controllers + BaseController (see table)
└── public/               ← the SPA (served statically)
    ├── index.html        ← shell: sidebar nav + topbar + <main id="view"> + one <script> per screen
    ├── css/style.css     ← all styling
    └── js/
        ├── app.js        ← the home-grown "framework": router, fetch wrapper, listView, buildForm, modal, toast
        └── <entity>.js   ← one per screen, each calls MIC.registerView(...)
```

### The PHP core (read these to understand any request)

- **`index.php`** — parses method+URI; `/css|/js` → static file; `/api/*` → `handleApi()`; anything else → SPA shell. `handleApi()` builds the **entire route table on every request** (`index.php:84-135`) using a generic `$crud` closure plus custom sub-routes, then dispatches.
- **`Router.php`** — converts `:name` segments to a named-group regex and matches method+path. Returns `[status, body]` or `null` (404).
- **`Database.php`** — one shared `PDO` (singleton), with a 30× retry loop so the app survives MySQL cold start (`Database.php:30-47`).
- **`Repository.php`** — **the heart of the storage layer.** Every controller goes through it. Provides `findByType` (filter + paginate + search, with a column whitelist), `findById`, `create`, `update`, `delete`, and the relation helpers (`relate`, `targetIdOf`, `relationsFrom/To`), plus `rawAll`/`rawOne` escape hatches that controllers use for cross-entity joins.
- **`Models/BusinessData.php` + `BusinessRelation.php`** — dumb DTOs mapping a DB row ↔ object. No behaviour; the generic columns are untyped here on purpose.

### Controllers (`src/Controllers/`)

All extend `BaseController` (generic JSON CRUD via `index/show/create/update/destroy`). Each concrete controller declares its `record_type` and the two translation methods: `toDto()` (generic columns → friendly JSON) and `fromPayload()` (friendly JSON → generic columns). **This is where the meaning of `amount_N`/`text_N` lives.**

| Controller | `record_type` | Route base | Notable beyond plain CRUD |
|---|---|---|---|
| `BaseController` | — | — | Generic CRUD + `readJsonBody`; the shared 80%. |
| `DashboardController` | — (reads many) | `/api/dashboard/kpi` | Raw cross-type aggregation SQL; **does not** use `BaseController`. |
| `ArticleController` | `articolo` | `/api/articles` | **Warehouse-BC target.** Article = SKU + base price (`amount_1`) + default VAT code (`text_2`) + qta min. |
| `CategoryController` | `categoria` | `/api/categories` | Plain CRUD. |
| `IvaController` | `aliquota_iva` | `/api/iva-rates` | VAT rate %; looked up by code from order lines. |
| `ListinoController` | `listino` | `/api/listini` | `voci(:id)` / `addVoce(:id)` → price-list entries linked to articles. |
| `CustomerController` | `cliente` | `/api/customers` | `orders(:id)` / `invoices(:id)` — reuse Order/Invoice controllers to render. |
| `SupplierController` | `fornitore` | `/api/suppliers` | Plain CRUD; no downstream flow. |
| `OrderController` | `ordine` | `/api/orders` | `righe`/`addRiga` compute price+IVA+totals; **reaches into article + IVA rows** (the headline coupling). |
| `ScontoController` | `sconto` | `/api/sconti` | Plain CRUD. |
| `AgenteController` | `agente` | `/api/agenti` | Plain CRUD; linkable to orders. |
| `MagazzinoController` | `magazzino` | `/api/magazzini` | `giacenze(:id)` derives stock as `SUM(movimenti)`. **Warehouse-BC target.** |
| `MovimentoController` | `movimento` | `/api/movimenti` | `create` wires `movimento_di_articolo` + `movimento_in_magazzino`. |
| `InvoiceController` | `fattura` | `/api/invoices` | `inviaSdi(:id)` mock; relates to cliente + ordine. |
| `NotaCreditoController` | `nota_credito` | `/api/note-credito` | API only, no screen. |
| `PagamentoController` | `pagamento` | `/api/pagamenti` | Relates to a fattura. |
| `UserController` | `utente` | `/api/users` | Placeholder password hash; no auth. |
| `AuditController` | `audit_log` | `/api/audit-log` | Read-only; surfaced in the dashboard. |

### The frontend (`public/js/`)

- **`app.js`** — read this once and the rest is obvious. It exposes the global `MIC` with: the hash router (`navigate`/`boot`), the `api/get/post/put/del` fetch wrapper, and the three building blocks every screen reuses — `listView` (paginated table from a list endpoint), `buildForm` (form from a field spec), and `modal`/`toast`/`confirmDialog`.
- **`dashboard.js`** — KPIs + Chart.js charts. The only non-CRUD screen.
- **`articles.js`, `customers.js`, `suppliers.js`, `listini.js`, `orders.js`, `invoices.js`, `magazzino.js`, `sconti.js`, `agenti.js`, `settings.js`** — one screen each (`settings.js` bundles Categorie + IVA + Utenti). `orders.js`, `invoices.js`, `listini.js`, `magazzino.js`, `customers.js` have richer detail modals (sub-resources, actions).

## "Where do I look to…" cheat-sheet

| I want to… | Go to |
|---|---|
| Understand how data is stored | `database/schema.sql` → `Repository.php` |
| Know what a column means for an entity | that entity's controller `toDto()` / `fromPayload()` |
| Add/trace an API route | `php-app/index.php` (`handleApi`) → the controller method |
| Add a new entity | new `XController extends BaseController` (define `type/toDto/fromPayload`) + register CRUD in `index.php` + new `public/js/x.js` + add a `<nav>` link + `<script>` in `index.html` |
| Change business logic (pricing/VAT/stock/totals) | `OrderController`, `MagazzinoController`, `ListinoController`, `InvoiceController` |
| Change the look of a screen | `public/js/<entity>.js` (+ `css/style.css`) |
| Run/inspect the stack | `docker-compose.yml`; Adminer at `:8082` |
