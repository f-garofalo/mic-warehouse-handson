# MIC — Mappa degli accoppiamenti

> Come sono *davvero* salvati i dati di business, e quali parti del dominio leggono o scrivono
> i dati altrui. Gli accoppiamenti che una futura estrazione dovrà rompere sono indicati esplicitamente.
> Evidenze: `database/schema.sql`, `Repository.php`, e i controller citati inline.

## 1. Come sono realmente salvati i dati

La GUI mostra ~13 schermate ordinate e ben separate. Sotto, **ci sono esattamente due tabelle** e **zero confini reali** (`database/schema.sql`):

### `business_data` — una tabella per ~18 tipi di entità
Ogni entità di business — `cliente, fornitore, contatto, articolo, categoria, aliquota_iva, listino, voce_listino, sconto, magazzino, movimento, agente, ordine, riga_ordine, fattura, nota_credito, pagamento, utente, audit_log` — è una riga qui, distinta **solo** dalla colonna `record_type`. Le colonne sono volutamente generiche:

```
code, name, description,
parent_id, parent_type,          -- link polimorfico al padre
amount_1..amount_4,              -- prezzo / totale / % / qta / ...
date_1..date_3,                  -- creazione / scadenza / valido-da / ...
text_1..text_5,                  -- email / indirizzo / codice-iva / ...
status, payload_json             -- discarica per l'overflow
```

**Il significato di ogni colonna generica non è nello schema — vive nel PHP.** Il `toDto()` / `fromPayload()` di ogni controller è l'*unico* posto che sa, es., che per un `articolo` `amount_1` = prezzo base e `text_2` = codice IVA di default (`ArticleController.php:20-55`), mentre per una `fattura` `amount_1` = imponibile e `text_1` = sezionale (`InvoiceController.php`). Leggi il DB senza il codice e non riesci a interpretare una sola riga.

### `business_relations` — una tabella per tutte le relazioni
Ogni associazione è una riga: `source_id, target_id, relation_type, amount, metadata`. **Nessuna foreign key** da nessuna parte — l'integrità è per convenzione. La colonna `amount` fa doppio gioco come qta / prezzo unitario / sconto % / importo pagato a seconda del `relation_type`.

I relation type in uso (dai commenti di `schema.sql` + seed):

| relation_type | source → target | `amount` significa |
|---|---|---|
| `cliente_di_ordine` | ordine → cliente | — |
| `agente_di_ordine` | ordine → agente | — |
| `listino_di_ordine` | ordine → listino | — |
| `articolo_in_riga_ordine` | riga_ordine → articolo | qta |
| `iva_di_riga` | riga_ordine → aliquota_iva | % IVA |
| `voce_di_listino` | voce_listino → listino | — |
| `articolo_di_voce` | voce_listino → articolo | prezzo unitario |
| `fattura_di_ordine` | fattura → ordine | — |
| `cliente_di_fattura` | fattura → cliente | — |
| `pagamento_di_fattura` | pagamento → fattura | importo pagato |
| `nota_credito_di_fattura` | nota_credito → fattura | importo |
| `movimento_di_articolo` | movimento → articolo | qta |
| `movimento_in_magazzino` | movimento → magazzino | — |
| `sconto_su_ordine` | ordine → sconto | — |
| `categoria_di_articolo` | articolo → categoria | — |

### Due meccanismi di collegamento paralleli (incoerenti)
Alcuni link padre/figlio usano `business_relations`; altri usano **`parent_id` + `parent_type`** direttamente su `business_data` — es. `riga_ordine` → `ordine` e `voce_listino` → `listino` sono memorizzati via `parent_*` (`OrderController.php:88-91`, `ListinoController.php:38-43`), mentre `audit_log` → `utente` usa `parent_id` (`DashboardController.php:96-98`). Quindi "chi possiede questa riga" va risposto in modo diverso a seconda del tipo. Questa duplicazione è essa stessa un pericolo di accoppiamento.

## 2. Chi legge/scrive chi (accessi cross-dominio)

Proprietà per dominio funzionale (quali `record_type` un dominio "possiede") e dove esce dai propri confini:

| Dominio | Possiede (record_type) | Legge da altri domini |
|---|---|---|
| Catalogo | `articolo`, `categoria`, `aliquota_iva` | — |
| Pricing | `listino`, `voce_listino` | **Catalogo** (`articolo`) |
| Ordini | `ordine`, `riga_ordine`, `sconto`, `agente` | **Catalogo** (prezzo + IVA dell'`articolo`, `aliquota_iva`) |
| Fatturazione | `fattura`, `nota_credito`, `pagamento` | **Ordini** (`ordine`), **Clienti** (`cliente`) |
| Magazzino | `magazzino`, `movimento` | **Catalogo** (identità dell'`articolo`) |
| Clienti | `cliente`, `contatto` | rilegge Ordini + Fatture per le sue tab di dettaglio |
| Dashboard | — | **tutto** (fattura, ordine, articolo, cliente, audit_log) |

### L'accoppiamento principale: **Ordini → Articolo (prezzo + IVA)**
È la dipendenza che l'estrazione del Warehouse deve rompere. In `OrderController::addRiga()` (`OrderController.php:121-167`), aggiungere una riga d'ordine **entra direttamente nelle colonne dell'articolo**:

```php
$articolo = $this->repo->findById($art_id);          // carica la riga di Catalogo
if ($prezzo <= 0) $prezzo = (float)($articolo->amount1 ?? 0);   // ← legge il PREZZO BASE dell'articolo
$iva_code = $articolo->text2 ?? 'IVA22';              // ← legge il CODICE IVA DI DEFAULT dell'articolo
$ivaRow = ... SELECT amount_1 FROM business_data WHERE record_type='aliquota_iva' AND code=?
$iva_perc = $ivaRow ? $ivaRow['amount_1'] : 22.0;     // ← legge la riga dell'aliquota IVA direttamente
```

Ordini non chiama nessun servizio Catalogo/Pricing — legge i grezzi `amount_1`/`text_2` della riga articolo e interroga le righe `aliquota_iva` a mano. **Nel momento in cui prezzo o IVA dell'Articolo si spostano in un servizio Warehouse/Catalogo, questa riga si rompe a meno che un contratto non sostituisca la lettura della colonna.** (Nota: gli ordini ignorano del tutto il prezzo del `listino` e usano il prezzo base dell'articolo — quindi il listino è una *seconda fonte di prezzo, parallela*, che il percorso d'ordine non consulta nemmeno.)

### Altre letture cross-dominio notevoli
- **Pricing → Catalogo**: `ListinoController::addVoce()` collega `voce → articolo` (`articolo_di_voce`) e memorizza il prezzo in **due** posti — `voce.amount_1` *e* l'`amount` della relazione (`ListinoController.php:72-81`).
- **Lo stock del Magazzino è derivato, non memorizzato**: `MagazzinoController::giacenze()` calcola lo stock con una join a 4 vie che somma le quantità dei `movimento` risalendo all'articolo (`MagazzinoController.php:46-62`). **Non c'è contatore di stock né invariante** — la giacenza può andare negativa (`magazzino.js:76`).
- **La fatturazione inventa i numeri**: il flusso "Genera fattura" imposta `imponibile = totale*0.82`, `iva = totale*0.18` lato client (`orders.js:147-148`), ignorando l'IVA reale per riga.
- **Clienti ← Ordini/Fatture in lettura**: `CustomerController::orders()/invoices()` istanziano `OrderController`/`InvoiceController` e ne chiamano il `dto()` (`CustomerController.php:64,78`) — accoppiamento controller-su-controller.
- **Dashboard → tutto**: `DashboardController::kpi()` esegue SQL grezzo con `record_type='fattura'|'ordine'|'articolo'|'cliente'|'audit_log'` hardcoded e fa persino join su `cliente_di_fattura` (`DashboardController.php:18-100`). Qualsiasi cambiamento di come uno di questi tipi è memorizzato rompe la dashboard in silenzio.

### Direzione delle dipendenze (perché il Warehouse è il primo taglio sicuro)
```
Ordini ───legge prezzo/IVA──►  ┐
Pricing ──collega all'articolo─► ├─►  Catalogo/Articolo  ◄──identità── Warehouse(magazzino/movimento)
Fatturazione ─legge via ordine─► ┘
Dashboard ─legge tutti i tipi──► (tutto)
```
**Warehouse** (`magazzino` + `movimento`, attorno all'`articolo` che traccia) ha lettori in ingresso ma **non legge le regole di business di nessun altro dominio** — gli serve solo l'identità dell'articolo. È la fetta più autosufficiente, quindi l'estrazione naturale per prima.

## 3. Accoppiamenti che una futura estrazione deve rompere

1. **Lettura colonna prezzo/IVA Ordini → Articolo** (`OrderController::addRiga`). Sostituire la lettura diretta di `articolo.amount1`/`articolo.text2` + il lookup di `aliquota_iva` con un contratto (una query/valore pubblicato) di proprietà del servizio estratto. *È la dipendenza n.1.*
2. **La tabella condivisa `business_data`.** Ogni dominio vive fisicamente nella stessa tabella; "estrarre il Warehouse" significa dare ai dati Articolo/stock un proprio store e fermare ogni altro controller dal `SELECT … WHERE record_type='articolo'`. La tabella condivisa è l'accoppiamento *strutturale* dietro a tutti gli altri.
3. **L'SQL grezzo cross-type della Dashboard.** Deve leggere via API / read-model, non facendo join sui record type, o si rompe nell'istante in cui un tipo viene estratto.
4. **Doppio link di proprietà** (`parent_id/parent_type` vs `business_relations`). Scegliere un solo modello di ownership per aggregato prima di tracciare i confini.

## Risposte alle exit question (da questa mappa)

- *Quale dipendenza dobbiamo affrontare per scavare fuori un pezzo per primo?* **Ordini → Articolo (prezzo + IVA)** letto in `OrderController::addRiga` — attraversa il confine Catalogo/Warehouse leggendo colonne grezze, quindi l'estrazione deve metterci un contratto.
- *Dove vive ogni dominio funzionale in schermate / codice / dati?* Vedi la tabella di proprietà sopra: ogni dominio = un insieme di schermate (`page-map-IT.md`), un controller (o pochi), e un valore di `record_type` dentro le due tabelle condivise.
