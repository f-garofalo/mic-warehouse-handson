# MIC — Mappa delle schermate

> Ogni schermata esposta dall'app e a cosa serve, raggruppata per dominio di business.
> Evidenze: `php-app/public/index.html` (sidebar + tag script), `php-app/public/js/*.js`, tabella delle rotte in `php-app/index.php`.

La UI è una SPA con routing via hash. La sidebar (`index.html:18-33`) espone **13 schermate** in due gruppi ("principale" + "Impostazioni"). Alcune entità hanno un'API funzionante ma **nessuna schermata** (indicate in fondo). **Non esiste una schermata di login**: la topbar mostra un utente fisso "Michele Mondora / admin" (`index.html:48-54`); MIC non ha autenticazione.

## Gruppo 1 — Operativo

| Schermata | Rotta | API | A cosa serve |
|---|---|---|---|
| **Dashboard** | `#/dashboard` | `GET /api/dashboard/kpi` | Pagina di atterraggio. 4 card KPI (fatturato del mese, ordini in corso, **articoli sotto scorta**, pagamenti scaduti) + 4 grafici (fatturato 6 mesi, top-5 clienti, ordini per stato, ultime 5 attività audit). Sola lettura; aggrega molti tipi di entità in un'unica query. ⚠️ "articoli sotto scorta" è **inventato** (`floor(count*0.18)`, `DashboardController.php:42`). |

## Gruppo 2 — Anagrafiche (master data)

| Schermata | Rotta | API | A cosa serve |
|---|---|---|---|
| **Clienti** | `#/customers` | `/api/customers` (+ `/:id/orders`, `/:id/invoices`) | Lista clienti + crea/modifica. Il modale di dettaglio ha tab **Anagrafica / Ordini / Fatture / Pagamenti**; la tab "Pagamenti" è calcolata lato client a partire dalla lista fatture (`customers.js:118-129`). B2B/B2C, P.IVA/CF, PEC, codice SDI. |
| **Fornitori** | `#/suppliers` | `/api/suppliers` | Lista fornitori + form. ⚠️ I fornitori esistono come anagrafica ma **nessun flusso di acquisto/ricezione** li usa — di fatto un dominio orfano in questa baseline. |

## Gruppo 3 — Catalogo & Pricing

| Schermata | Rotta | API | A cosa serve |
|---|---|---|---|
| **Articoli** | `#/articles` | `/api/articles` | Il catalogo prodotti: SKU, nome, categoria, IVA di default, **prezzo di listino base**, quantità minima, stato. Modale crea/modifica. **È il target di estrazione del BC Warehouse** (`ArticleController.php:6-15`). La riga Articolo porta insieme identità di catalogo **e** prezzo base **e** codice IVA di default. |
| **Categorie** | `#/categories` | `/api/categories` | Categorie articoli (impostazioni). CRUD semplice. |
| **Aliquote IVA** | `#/iva` | `/api/iva-rates` | Aliquote IVA (codice + %). CRUD semplice. Le righe d'ordine le cercano per codice. |
| **Listini** | `#/listini` | `/api/listini` (+ `/:id/voci`) | Listini prezzi. Il modale di dettaglio elenca le **voci** (articolo → prezzo) e permette di aggiungerne. Una fonte di prezzo per cliente/listino — ma nota che gli ordini **non** la consultano (vedi mappa accoppiamenti). |

## Gruppo 4 — Vendite

| Schermata | Rotta | API | A cosa serve |
|---|---|---|---|
| **Ordini** | `#/orders` | `/api/orders` (+ `/:id/righe`) | Ordini di vendita. Lista + modale di dettaglio con le **righe**. Aggiungendo una riga si sceglie un articolo e prezzo/IVA/totali vengono calcolati lato server. Ha un pulsante **"Genera fattura"** che crea una fattura con split imponibile/IVA **inventato** 82/18 (`orders.js:147-148`). |
| **Sconti** | `#/sconti` | `/api/sconti` | Definizioni di sconto (%, validità). CRUD. Collegabili agli ordini via `sconto_su_ordine` (presente nel seed), ma nessuna UI li collega. |
| **Agenti** | `#/agenti` | `/api/agenti` | Agenti di vendita (zona, provvigione). CRUD. Selezionabili in fase di creazione ordine. |

## Gruppo 5 — Fatturazione

| Schermata | Rotta | API | A cosa serve |
|---|---|---|---|
| **Fatture** | `#/invoices` | `/api/invoices` (+ `/:id/invia-sdi`) | Fatture. Lista + dettaglio. Azioni: **"Invia a SDI (mock)"** (porta `sdi_status` a `INVIATA`) e **"Registra pagamento"** (crea un `pagamento` collegato alla fattura e la segna `pagato`, `invoices.js:83-95`). |

## Gruppo 6 — Magazzino

| Schermata | Rotta | API | A cosa serve |
|---|---|---|---|
| **Magazzino** | `#/magazzino` | `/api/magazzini` (+ `/:id/giacenze`), `/api/movimenti` | Magazzini + movimenti recenti + "Giacenze" (stock per magazzino). ⚠️ **La giacenza non è memorizzata** — è una `SUM` live delle quantità dei movimenti (`MagazzinoController.php:46-62`) e **può andare negativa** (`magazzino.js:76` colora di rosso i valori negativi). "Nuovo movimento" registra un'entrata/uscita/ecc. |

## Gruppo 7 — Sistema (impostazioni)

| Schermata | Rotta | API | A cosa serve |
|---|---|---|---|
| **Utenti** | `#/users` | `/api/users` | Account utente (email, nome, ruolo). ⚠️ Nessuna vera autenticazione: gli utenti creati ricevono un hash password placeholder (`UserController.php:29`) e i ruoli non sono applicati da nessuna parte. |

## Entità con API ma **senza schermata**

Hanno un controller + rotte + dati nel seed, ma nessuna voce nella sidebar / vista JS:

- **Note di credito** — `NotaCreditoController`, `/api/note-credito` (nessun `js/notecredito.js`, assente dalla nav).
- **Audit log** — `AuditController`, `/api/audit-log` (sola lettura; emerge solo nel widget "Ultime 5 attività" della Dashboard).
- **Contatti** — esiste un record type `contatto` nello schema/seed (figlio di un cliente via `parent_id`) ma senza controller né schermata.

## Domini funzionali che le schermate rivelano (prep per le exit question)

Catalogo (Articoli/Categorie/IVA) · Pricing (Listini) · Vendite/Ordini (Ordini/Sconti/Agenti) · Clienti · Fatturazione (Fatture/Pagamenti/Note credito) · Magazzino (Magazzini/Movimenti/Giacenze) · più la Dashboard trasversale e un sottile strato Sistema (Utenti/Audit). I Fornitori compaiono come anagrafica senza un flusso. Questi domini mappano in modo pulito sui candidati Bounded Context analizzati nella Fase 02.
