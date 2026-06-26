# MIC — Sintesi dell'analisi (Gruppo 3)

> Cosa è MIC, com'è fatto, e dov'è il problema. Tutto ricavato dal codice (codice + `schema.sql` + `seed.sql`), con evidenze `file:riga`. Indice completo dei deliverable: [`README-IT.md`](./README-IT.md).

## 1. Page map — le schermate
SPA con **13 schermate** nella sidebar, raggruppabili per dominio: **Catalogo** (Articoli, Categorie, Aliquote IVA), **Pricing** (Listini), **Vendite** (Ordini, Sconti, Agenti), **Clienti**, **Fatturazione** (Fatture, Pagamenti), **Magazzino** (Magazzini/Movimenti/Giacenze), **Sistema** (Utenti) + una **Dashboard** trasversale. Tre entità hanno API ma **nessuna schermata** (Note di credito, Audit log, Contatti). **Non c'è login**: utente "admin" hardcoded, MIC non ha autenticazione.

## 2. Architettura — cosa gira e come scorre una richiesta
**Tre container**: `mic-app` (nginx + PHP-FPM nella **stessa** immagine, via supervisor), `mysql` (unico datastore), `adminer` (browser DB). Una richiesta fa: **browser SPA → nginx → `index.php` (front controller) → Router → Controller → Repository → PDO → MySQL** e ritorno in JSON. **Niente framework, niente ORM, niente Composer**: routing, accesso dati e SPA sono scritti a mano. Envelope uniforme `{ data, meta }`. *(Diagramma in [`architecture.svg`](./architecture.svg).)*

## 3. Repo guide — dov'è cosa
Il cuore PHP: `index.php` (rotte), `Router.php`, `Database.php` (PDO singleton), **`Repository.php` (l'unico data layer generico, ci passa tutto)**, `Models/` (`BusinessData`/`BusinessRelation`, DTO senza logica), `Controllers/` (**17 + Base**, qui vive il significato delle colonne via `toDto`/`fromPayload`). La SPA: `public/js/app.js` (il "framework" fatto in casa: router, fetch, `listView`, `buildForm`, modal) + un `.js` per schermata. I dati: `database/schema.sql` + `seed.sql`. **Per capire qualunque richiesta basta leggere `index.php` → il controller → `Repository`.**

## 4. Coupling map — come sono salvati i dati e chi legge chi
**Lo storage reale:** tutti i ~18 tipi di entità in **una sola tabella** `business_data`, distinti da `record_type`; tutte le relazioni in `business_relations`. Colonne generiche (`amount_1..4`, `text_1..5`) il cui **significato vive solo nel PHP**, non nello schema. **Nessuna foreign key**, e **due meccanismi paralleli** per i legami (`business_relations` *e* `parent_id`/`parent_type`).

**Le letture cross-dominio (chi legge chi):**
- **Ordini → Articolo** (prezzo + IVA) — *l'accoppiamento chiave*;
- **Pricing → Articolo** (le voci di listino puntano all'articolo, prezzo duplicato);
- **Warehouse** legge solo l'**identità** dell'articolo (e non legge nessun altro: è un *sink*);
- **Fatturazione → Ordini/Clienti**; **Dashboard → tutto** (SQL grezzo su tanti `record_type`).

## 5. Worst-of — i 3 antipattern peggiori
1. **La "God Table".** Tutto il business in 2 tabelle generiche: il DB **non può imporre nessun vincolo** (né FK, né tipi, né invarianti), e i dati sono **illeggibili senza il codice**.
2. **Nessun domain layer.** Le regole di business e le letture di altri domini stanno nei controller HTTP che **leggono le colonne grezze altrui** (Ordini→Articolo, Dashboard→tutto): **blast radius enorme**, niente contratti, niente anti-corruption.
3. **Numeri di cui non fidarsi.** Dati finanziari e di stock **derivati/duplicati**, in più punti **inventati**: "articoli sotto scorta" = `conteggio × 0,18`, "Genera fattura" spacca il totale **82/18** a mano, la **giacenza può andare negativa**, e il `Money` **non ha valuta**. Per un gestionale di fatturazione è **correttezza, non estetica**.

---

## Conclusioni

**Da dove partiremmo: estrarre per primo il Warehouse.** È l'unico contesto **sink** — ha lettori (Ordini, Pricing, Fatturazione) ma **non legge nessun altro dominio**: niente a valle che si rompe. Lo si tira su come servizio Go e si spostano i lettori uno alla volta, col monolite che continua a servire. È anche il contesto **più pesante** (stock, movimenti), quindi quello che più merita un suo ciclo di vita.

**La prima dipendenza da rompere: `Ordini → Articolo` (prezzo + IVA).** In [`OrderController::addRiga`](../php-app/src/Controllers/OrderController.php:131) l'ordine legge `articolo.amount_1` (prezzo) e `articolo.text_2` (codice IVA) e poi la tabella `aliquota_iva`. Va sostituita con un **contratto** (una query/valore pubblicato) di proprietà del servizio estratto, invece della lettura diretta delle colonne.

**La scelta che rende MIC più difficile da cambiare in sicurezza: la God Table** — perché toglie ogni confine e ogni invariante garantito dal DB. **Cosa faremmo:** estrarre i Bounded Context **uno alla volta** (Strangler Fig), dando a ciascuno il **proprio store tipizzato** e un **contratto alla cucitura**, partendo dal taglio più sicuro (Warehouse).

**I domini funzionali presenti** (e dove vivono — schermate / controller / `record_type` nelle 2 tabelle): **Catalog, Pricing, Warehouse, Orders, Invoicing, Customers**, più la **Dashboard** trasversale e un sottile strato **Sistema**. Questi sono esattamente i confini su cui agiranno le fasi successive.
