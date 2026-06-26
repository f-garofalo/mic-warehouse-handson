# MIC — Page Map

> Every screen the app exposes and what it is for, grouped by business domain.
> Evidence: `php-app/public/index.html` (sidebar + script tags), `php-app/public/js/*.js`, route table in `php-app/index.php`.

The UI is a hash-routed SPA. The sidebar (`index.html:18-33`) exposes **13 screens** in two groups ("main" + "Impostazioni"). A few entities have a working API but **no screen** (noted at the end). There is **no login screen** — the topbar shows a hardcoded user "Michele Mondora / admin" (`index.html:48-54`); MIC has no auth.

## Group 1 — Operativo

| Screen | Route | API | What it's for |
|---|---|---|---|
| **Dashboard** | `#/dashboard` | `GET /api/dashboard/kpi` | Landing page. 4 KPI cards (fatturato del mese, ordini in corso, **articoli sotto scorta**, pagamenti scaduti) + 4 charts (fatturato 6 mesi, top-5 clienti, ordini per stato, ultime 5 attività audit). Read-only; aggregates across many entity types in one query. ⚠️ "articoli sotto scorta" is **fabricated** (`floor(count*0.18)`, `DashboardController.php:42`). |

## Group 2 — Anagrafiche (master data)

| Screen | Route | API | What it's for |
|---|---|---|---|
| **Clienti** | `#/customers` | `/api/customers` (+ `/:id/orders`, `/:id/invoices`) | Customer list + create/edit. Detail modal has tabs **Anagrafica / Ordini / Fatture / Pagamenti**; the "Pagamenti" tab is computed client-side from the invoice list (`customers.js:118-129`). B2B/B2C, P.IVA/CF, PEC, codice SDI. |
| **Fornitori** | `#/suppliers` | `/api/suppliers` | Supplier list + form. ⚠️ Suppliers exist as master data but **no purchase/receiving flow** uses them — effectively an orphan domain in this baseline. |

## Group 3 — Catalogo & Pricing

| Screen | Route | API | What it's for |
|---|---|---|---|
| **Articoli** | `#/articles` | `/api/articles` | The product catalog: SKU, nome, categoria, IVA di default, **prezzo di listino base**, qta minima, stato. Create/edit modal. **This is the Warehouse-BC extraction target** (`ArticleController.php:6-15`). The Article row carries catalog identity **and** base price **and** default VAT code together. |
| **Categorie** | `#/categories` | `/api/categories` | Article categories (settings). Plain CRUD. |
| **Aliquote IVA** | `#/iva` | `/api/iva-rates` | VAT rates (code + %). Plain CRUD. Order lines look these up by code. |
| **Listini** | `#/listini` | `/api/listini` (+ `/:id/voci`) | Price lists. Detail modal lists **voci** (article → price) and lets you add a voce. A per-customer/per-list price source — but note orders don't actually consult it (see coupling map). |

## Group 4 — Vendite (sales)

| Screen | Route | API | What it's for |
|---|---|---|---|
| **Ordini** | `#/orders` | `/api/orders` (+ `/:id/righe`) | Sales orders. List + detail modal with **righe** (order lines). Adding a line picks an article and auto-computes price/IVA/totals server-side. Has a **"Genera fattura"** button that creates an invoice with **fabricated** 82/18 imponibile/IVA split (`orders.js:147-148`). |
| **Sconti** | `#/sconti` | `/api/sconti` | Discount definitions (%, validity). CRUD. Linkable to orders via `sconto_su_ordine` (seeded), but no UI wires it. |
| **Agenti** | `#/agenti` | `/api/agenti` | Sales agents (zona, provvigione). CRUD. Selectable when creating an order. |

## Group 5 — Fatturazione (billing)

| Screen | Route | API | What it's for |
|---|---|---|---|
| **Fatture** | `#/invoices` | `/api/invoices` (+ `/:id/invia-sdi`) | Invoices. List + detail. Actions: **"Invia a SDI (mock)"** (flips `sdi_status` to `INVIATA`) and **"Registra pagamento"** (creates a `pagamento` linked to the invoice and marks it `pagato`, `invoices.js:83-95`). |

## Group 6 — Magazzino (warehouse)

| Screen | Route | API | What it's for |
|---|---|---|---|
| **Magazzino** | `#/magazzino` | `/api/magazzini` (+ `/:id/giacenze`), `/api/movimenti` | Warehouses + recent stock movements + "Giacenze" (per-warehouse stock). ⚠️ **Stock is not stored** — `giacenza` is a live `SUM` of movement quantities (`MagazzinoController.php:46-62`) and **can go negative** (`magazzino.js:76` paints negatives red). "Nuovo movimento" registers an entrata/uscita/etc. |

## Group 7 — Sistema (settings)

| Screen | Route | API | What it's for |
|---|---|---|---|
| **Utenti** | `#/users` | `/api/users` | User accounts (email, nome, ruolo). ⚠️ No real auth: created users get a placeholder password hash (`UserController.php:29`) and roles aren't enforced anywhere. |

## Entities with an API but **no screen**

These have a controller + routes + seed data, but no sidebar entry / JS view:

- **Note di credito** — `NotaCreditoController`, `/api/note-credito` (no `js/notecredito.js`, not in nav).
- **Audit log** — `AuditController`, `/api/audit-log` (read-only; surfaced only inside the Dashboard "Ultime 5 attività" widget).
- **Contatti** — a `contatto` record type exists in the schema/seed (child of a cliente via `parent_id`) but has no controller or screen.

## Functional domains the screens reveal (exit-question prep)

Catalog (Articoli/Categorie/IVA) · Pricing (Listini) · Sales/Orders (Ordini/Sconti/Agenti) · Customers (Clienti) · Invoicing (Fatture/Pagamenti/Note credito) · Warehouse (Magazzino/Movimenti/Giacenze) · plus cross-cutting Dashboard and a thin System area (Utenti/Audit). Suppliers appear as master data without a flow. These map cleanly onto the bounded-context candidates analysed in Phase 02.
