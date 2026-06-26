# MIC — Coupling Map

> How the business data is *actually* stored, and which parts of the domain read or write
> each other's data. The couplings a future extraction must break are called out explicitly.
> Evidence: `database/schema.sql`, `Repository.php`, and the controllers cited inline.

## 1. How the data is really stored

The GUI shows ~13 tidy, well-separated screens. Underneath, **there are exactly two tables** and **zero real boundaries** (`database/schema.sql`):

### `business_data` — one table for ~18 entity types
Every business entity — `cliente, fornitore, contatto, articolo, categoria, aliquota_iva, listino, voce_listino, sconto, magazzino, movimento, agente, ordine, riga_ordine, fattura, nota_credito, pagamento, utente, audit_log` — is a row here, told apart **only** by the `record_type` column. The columns are deliberately generic:

```
code, name, description,
parent_id, parent_type,          -- polymorphic parent link
amount_1..amount_4,              -- price / total / % / qty / ...
date_1..date_3,                  -- created / due / valid-from / ...
text_1..text_5,                  -- email / address / vat-code / ...
status, payload_json             -- overflow dumping ground
```

**The meaning of each generic column is not in the schema — it lives in PHP.** Each controller's `toDto()` / `fromPayload()` is the *only* place that knows, e.g., that for an `articolo` `amount_1` = base price and `text_2` = default VAT code (`ArticleController.php:20-55`), while for a `fattura` `amount_1` = imponibile and `text_1` = sezionale (`InvoiceController.php`). Read the DB without the code and you cannot interpret a single row.

### `business_relations` — one table for all relationships
Every association is a row: `source_id, target_id, relation_type, amount, metadata`. There are **no foreign keys** anywhere — integrity is by convention. The `amount` column doubles as qty / unit price / discount % / paid amount depending on `relation_type`.

The relation types in use (from `schema.sql` comments + seed):

| relation_type | source → target | `amount` means |
|---|---|---|
| `cliente_di_ordine` | ordine → cliente | — |
| `agente_di_ordine` | ordine → agente | — |
| `listino_di_ordine` | ordine → listino | — |
| `articolo_in_riga_ordine` | riga_ordine → articolo | qty |
| `iva_di_riga` | riga_ordine → aliquota_iva | VAT % |
| `voce_di_listino` | voce_listino → listino | — |
| `articolo_di_voce` | voce_listino → articolo | unit price |
| `fattura_di_ordine` | fattura → ordine | — |
| `cliente_di_fattura` | fattura → cliente | — |
| `pagamento_di_fattura` | pagamento → fattura | paid amount |
| `nota_credito_di_fattura` | nota_credito → fattura | amount |
| `movimento_di_articolo` | movimento → articolo | qty |
| `movimento_in_magazzino` | movimento → magazzino | — |
| `sconto_su_ordine` | ordine → sconto | — |
| `categoria_di_articolo` | articolo → categoria | — |

### Two parallel linking mechanisms (inconsistent)
Some parent/child links use `business_relations`; others use **`parent_id` + `parent_type`** directly on `business_data` — e.g. `riga_ordine` → `ordine` and `voce_listino` → `listino` are stored via `parent_*` (`OrderController.php:88-91`, `ListinoController.php:38-43`), while `audit_log` → `utente` uses `parent_id` (`DashboardController.php:96-98`). So "what owns this row" must be answered differently per type. This duplication is itself a coupling hazard.

## 2. Who reads/writes whom (cross-domain data access)

Ownership by functional domain (which `record_type`s a domain "owns") and where it reaches outside itself:

| Domain | Owns (record_type) | Reads from other domains |
|---|---|---|
| Catalog | `articolo`, `categoria`, `aliquota_iva` | — |
| Pricing | `listino`, `voce_listino` | **Catalog** (`articolo`) |
| Orders | `ordine`, `riga_ordine`, `sconto`, `agente` | **Catalog** (`articolo` price + VAT, `aliquota_iva`) |
| Invoicing | `fattura`, `nota_credito`, `pagamento` | **Orders** (`ordine`), **Customers** (`cliente`) |
| Warehouse | `magazzino`, `movimento` | **Catalog** (`articolo` identity) |
| Customers | `cliente`, `contatto` | reads back Orders + Invoices for its detail tabs |
| Dashboard | — | **everything** (fattura, ordine, articolo, cliente, audit_log) |

### The headline coupling: **Orders → Article (price + VAT)**
This is the dependency the Warehouse extraction has to break. In `OrderController::addRiga()` (`OrderController.php:121-167`), adding an order line reaches **directly into the article's columns**:

```php
$articolo = $this->repo->findById($art_id);          // load the Catalog row
if ($prezzo <= 0) $prezzo = (float)($articolo->amount1 ?? 0);   // ← read Article BASE PRICE
$iva_code = $articolo->text2 ?? 'IVA22';              // ← read Article's DEFAULT VAT CODE
$ivaRow = ... SELECT amount_1 FROM business_data WHERE record_type='aliquota_iva' AND code=?
$iva_perc = $ivaRow ? $ivaRow['amount_1'] : 22.0;     // ← read the IVA-rate row directly
```

Orders does not call any Catalog/Pricing service — it reads the raw `amount_1`/`text_2` of the article row and queries the `aliquota_iva` rows by hand. **The moment Article's price or VAT moves to a Warehouse/Catalog service, this line breaks unless a contract replaces the column read.** (Note: orders ignore the `listino` price entirely and use the article base price — so the price list is a *second, parallel* price source that the order path doesn't even consult.)

### Other notable cross-domain reads
- **Pricing → Catalog**: `ListinoController::addVoce()` links `voce → articolo` (`articolo_di_voce`) and stores the price in **two** places — `voce.amount_1` *and* the relation `amount` (`ListinoController.php:72-81`).
- **Warehouse stock is derived, not stored**: `MagazzinoController::giacenze()` computes stock with a 4-way join summing `movimento` quantities back to the article (`MagazzinoController.php:46-62`). There is **no stock counter and no invariant** — giacenza can go negative (`magazzino.js:76`).
- **Invoicing fabricates numbers**: the "Genera fattura" flow sets `imponibile = totale*0.82`, `iva = totale*0.18` client-side (`orders.js:147-148`), ignoring the real per-line VAT.
- **Customers ← Orders/Invoices at read time**: `CustomerController::orders()/invoices()` instantiate `OrderController`/`InvoiceController` and call their `dto()` (`CustomerController.php:64,78`) — controller-to-controller coupling.
- **Dashboard → all**: `DashboardController::kpi()` runs raw SQL hardcoding `record_type='fattura'|'ordine'|'articolo'|'cliente'|'audit_log'` and even joins `cliente_di_fattura` (`DashboardController.php:18-100`). Any change to how one of those types is stored breaks the dashboard silently.

### Dependency direction (why Warehouse is the safe first cut)
```
Orders ───reads price/VAT──►  ┐
Pricing ──links to article──► ├─►  Catalog/Article  ◄──identity── Warehouse(magazzino/movimento)
Invoicing ─reads via order──► ┘
Dashboard ─reads all types──► (everything)
```
**Warehouse** (`magazzino` + `movimento`, around the `articolo` it tracks) has inbound readers but reads **no other domain's** business rules — it only needs the article's identity. It is the most self-sufficient slice, hence the natural first extraction.

## 3. Couplings a future extraction must break

1. **Orders → Article price/VAT column read** (`OrderController::addRiga`). Replace the direct read of `articolo.amount1`/`articolo.text2` + the `aliquota_iva` lookup with a contract (a query/published value) owned by the extracted service. *This is the #1 dependency.*
2. **The shared `business_data` table.** Every domain physically lives in the same table; "extracting Warehouse" means giving the Article/stock data its own store and stopping every other controller from `SELECT … WHERE record_type='articolo'`. The shared table is the *structural* coupling behind all the others.
3. **Dashboard's cross-type raw SQL.** It must read through APIs / read-models, not by joining record types, or it breaks the instant any type is extracted.
4. **Dual ownership links** (`parent_id/parent_type` vs `business_relations`). Pick one model of ownership per aggregate before carving boundaries.

## Exit-question answers (from this map)

- *Which dependency must we deal with to carve a piece out first?* **Orders → Article (price + VAT)** read in `OrderController::addRiga` — it crosses the Catalog/Warehouse boundary by reading raw columns, so the extraction must put a contract there.
- *Where does each functional domain live in screens / code / data?* See the ownership table above: each domain = a set of screens (`page-map.md`), a controller (or few), and a `record_type` value inside the shared two tables.
