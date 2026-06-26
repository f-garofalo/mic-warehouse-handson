# MIC — Lista dei peggiori (Top 3 antipattern)

> Le tre scelte architetturali che fanno più male, perché ciascuna fa male, e cosa costa al team
> continuare a conviverci. In ordine di blast radius. Evidenze citate inline.

---

## #1 — La "God Table": uno schema generico per tutto il business

**Cos'è.** Tutti i ~18 tipi di entità condividono **una** tabella, `business_data`, distinta solo da una stringa `record_type`; tutte le relazioni condividono **una** tabella, `business_relations`, distinta da `relation_type` (`database/schema.sql`). Le colonne sono prive di senso da sole — `amount_1..4`, `text_1..5`, `date_1..3`, più un overflow `payload_json`. Il *significato* di ogni colonna esiste solo nel PHP, nel `toDto()`/`fromPayload()` di ogni controller (es. per un `articolo`, `amount_1`=prezzo e `text_2`=codice IVA — `ArticleController.php:20-55`). **Nessuna foreign key**, **nessun vincolo NOT NULL/CHECK per entità**, **nessuna colonna tipizzata**.

**Perché fa male.**
- Il database non può imporre **nulla**: non "un ordine ha un cliente", non "un prezzo è positivo", nemmeno che un `target_id` punti a una riga reale. Ogni invariante è opzionale e vive in PHP sparso.
- **Non puoi leggere i dati senza il codice.** Una riga `{record_type:'articolo', amount_1:12.5, text_2:'IVA22'}` è indecifrabile a meno di conoscere `ArticleController`. Onboarding, debug e reportistica richiedono tutti conoscenza tribale.
- **Nessun confine lungo cui estrarre.** Ogni dominio è fisicamente intrecciato nella stessa tabella, quindi non c'è cucitura da tagliare — ed è esattamente per questo che un'estrazione ha bisogno di un piano deliberato.
- Indici e query sono generici (`idx_type`, `idx_type_status`); non si possono ottimizzare per entità, e ogni query comincia con `WHERE record_type = …`.

**Cosa costa tenerla.** Tassa di onboarding permanente (tutti devono memorizzare la mappa significato-colonne); un flusso costante di bug "colonna/tipo sbagliato"; dati di cui non ti puoi fidare per finanza o magazzino; e una migrazione *più difficile del dovuto* perché lo strato di storage non codifica alcun confine. È la singola scelta di design che rende MIC più difficile da cambiare in sicurezza — **risolverla (dare a ogni bounded context il proprio store tipizzato) è tutto il senso del lab.**

---

## #2 — Nessun domain layer: la logica di business attraversa i domini leggendo colonne grezze

**Cos'è.** Non c'è uno strato di modello né contratti tra domini. Le regole di business vivono dentro i controller HTTP che leggono **le righe/colonne grezze di altri domini** direttamente:
- `OrderController::addRiga()` calcola prezzo/IVA leggendo `amount_1` (prezzo base) e `text_2` (codice IVA) dell'articolo, poi interrogando le righe `aliquota_iva` a mano (`OrderController.php:131-143`).
- `DashboardController::kpi()` hardcoda `record_type='fattura'|'ordine'|'articolo'|'cliente'|'audit_log'` e ci fa join in SQL grezzo (`DashboardController.php:18-100`).
- `CustomerController` istanzia `OrderController`/`InvoiceController` e ne chiama il `dto()` per renderizzare le sue tab (`CustomerController.php:64,78`).

**Perché fa male.**
- **Blast radius alto.** Un cambiamento di come l'Articolo memorizza prezzo o IVA rompe in silenzio Ordini; un cambiamento nello storage di qualunque entità rompe la Dashboard. Niente te lo dice a compile time.
- **Nessun confine anti-corruption.** Non puoi estrarre Catalogo/Warehouse senza rompere Ordini, perché Ordini dipende dalle *colonne fisiche* di Catalogo, non da un'interfaccia stabile.
- **Regole duplicate.** Il default IVA (`'IVA22'` / `22.0`) è hardcoded in più di un posto (`OrderController.php:137,142`); la logica di prezzo sta ovunque sia servita a qualcuno.

**Cosa costa tenerlo.** Ogni cambiamento cross-dominio diventa un grep-and-pray attento su tutto il repo. L'estrazione Strangler è costretta a costruire anti-corruption layer proprio qui (Ordini↔Catalogo) prima che qualcosa possa muoversi — il lavoro che questo lab sequenzia nelle fasi successive.

---

## #3 — Numeri di cui non fidarsi: dati finanziari e di stock derivati/duplicati, con valori inventati e senza invarianti

**Cos'è.** Soldi e stock **non hanno una fonte di verità unica** e, in alcuni punti, sono semplicemente inventati:
- **Lo stock è derivato, mai memorizzato**, come `SUM(movimento.amount)` via una join a 4 vie (`MagazzinoController.php:46-62`) — e **può andare negativo**: la UI ha letteralmente uno stile rosso per `giacenza < 0` (`magazzino.js:76`). Nessuna prenotazione, nessun lock, niente impedisce di vendere sotto zero.
- **I totali d'ordine sono sia memorizzati sia ricalcolati**: ogni riga tiene `imponibile/iva/iva_perc` in `payload_json` *e* in `amount_*`, e `recalcOrdine()` ri-deriva la testata d'ordine (`OrderController.php:146-198`) — tre copie che possono divergere.
- **Il prezzo è duplicato** tra `articolo.amount_1`, `voce_listino.amount_1` e l'`amount` della relazione `articolo_di_voce` (`ListinoController.php:72-81`).
- **Numeri inventati**: "Genera fattura" spezza un totale come `imponibile=totale*0.82`, `iva=totale*0.18` (`orders.js:147-148`); la KPI "articoli sotto scorta" della Dashboard è `floor(count*0.18)` — **pura finzione** (`DashboardController.php:41-42`).
- **`Money` non ha valuta** — ogni importo è un `DECIMAL` nudo (`schema.sql`), quindi il modello non può rappresentare né proteggere la valuta.

**Perché fa male.** Lo stesso fatto di business (un totale, un livello di stock, un prezzo) dà risposte diverse a seconda di dove lo leggi. Per un prodotto di *fatturazione e magazzino*, è un problema di correttezza, non estetico: IVA sbagliata, stock venduto sotto zero, KPI di cui nessuno si fida. La concorrenza rende lo stock derivato insicuro.

**Cosa costa tenerlo.** Riconciliazione continua e indagini "perché questi numeri non tornano?"; un rischio reale di emettere fatture errate o magazzino negativo; e un blocco all'estrazione, perché un servizio pulito deve prima definire un proprietario unico e un'invariante per ogni valore (es. modellare le prenotazioni come un contatore esplicito, secondo il target della Fase 02).

---

## Menzioni d'onore (non nella top 3, ma da segnalare)

- **Nessun platform layer / niente auth.** Un utente "admin" hardcoded nella topbar (`index.html:48-54`), hash password placeholder (`UserController.php:29`), ruoli applicati da nessuna parte. Va bene per un monolite didattico; pericoloso se mai scambiato per produzione.
- **Tabella delle rotte ricostruita a ogni richiesta** (`index.php:84-135`) e **Chart.js caricato da CDN** (`index.html:9`) — minore, ma un accoppiamento runtime a un host esterno.
- **Domini orfani nei dati ma non nel flusso** — `fornitore`, `sconto`, `contatto` esistono con dati nel seed ma nessun processo reale li usa.

## Aggancio alle exit question
- *Singola scelta di design che rende MIC più difficile da cambiare in sicurezza?* **#1, la God Table** — rimuove ogni confine e ogni invariante imposta dal DB.
- *Cosa faresti?* Estrarre i bounded context uno alla volta (Strangler Fig), dando a ciascuno il proprio store tipizzato e un contratto alla cucitura — partendo dalla fetta più autosufficiente (**Warehouse**), e rompendo per prima la lettura **Ordini→Articolo prezzo/IVA**.
