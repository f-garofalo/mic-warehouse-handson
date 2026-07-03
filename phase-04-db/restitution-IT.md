# Restituzione — Fase 04: adapter del Warehouse BC e dual-write (Gruppo 3)

> English version: [`restitution.md`](./restitution.md)
> Script parlato per la restituzione di CP4 (~4 min). Confrontiamo la *forma* della cucitura, non la sintassi.
> Design in `repositories/legacy_article_repository.go` + `dual_write_article_repository.go`.

## Apertura
Siamo il Gruppo 3. In CP3 abbiamo costruito il dominio del Warehouse e la sua **porta**
`ArticleRepository` — un contratto senza implementazione. CP4 è dove quella porta smette di essere
astratta: le diamo **adapter reali** e li facciamo girare **accanto al database legacy**. È la
**Strangler Fig** che si accende.

## Lo scenario di migrazione — perché serve un ACL
Il Warehouse BC deve girare **fianco a fianco** col DB legacy durante la migrazione: ogni scrittura
atterra in **entrambi** gli store, così nulla si rompe mentre il traffico è ancora sul monolite. Ma i
due store modellano il prezzo in modo diverso, **di proposito**:

| | `legacy_db` | `warehouse_db` |
|---|---|---|
| prezzo | `DECIMAL(10,2)` — `29.99` | `price_cents BIGINT` — `2999` |
| valuta | assente (si assume EUR) | esplicita `CHAR(3)` — `EUR` |

Un **Anti-Corruption Layer (ACL)** contiene quel disordine così che non raggiunga mai il dominio pulito.

## Task 1 — l'adapter ACL legacy
`LegacyMySQLArticleRepository` implementa la porta contro `legacy_db`. Il cuore è la traduzione del
prezzo: `centsToDecimal` (`2999 → "29.99"`) e `decimalToCents` (`"29.99" → 2999`), fatte con
**aritmetica intera/stringa — mai `float64`**, perché un float corrompe silenziosamente il denaro. In
lettura **inventa `currency = EUR`** (legacy non ha la colonna); in scrittura **la elimina**. `Save` è
un upsert idempotente (`INSERT ... ON DUPLICATE KEY UPDATE`); `Find`/`List` **reidratano tramite le
factory di dominio** (`NewSKU`, `NewMoney`), così `entities.Article` vede sempre un `Money` valido
anche se la tabella legacy non sa rappresentarlo. Il dominio non conosce mai i dettagli legacy — è
tutto il senso dell'ACL.

## Task 2 — il decorator dual-write / single-read
`DualWriteArticleRepository` è un **decorator**: dall'esterno *è* un `ArticleRepository`; dentro tiene
gli adapter legacy + BC. Le **scritture** (`Save`/`Delete`) vanno su **entrambi** gli store, **legacy
per primo**: se legacy fallisce ci fermiamo e non tocchiamo BC; se BC fallisce ritorniamo l'errore
(legacy tiene l'articolo — nessun rollback in questo esercizio). Le **letture** vanno su **un solo**
store, scelto da un flag `ReadMode` (`ReadFromLegacy` di default → `ReadFromBC` dopo il cutover).
Nessun confronto, nessuna riconciliazione. Secondo ADR-013 questo decorator è **transitorio**: a
cutover completato viene eliminato.

## Cosa NON abbiamo fatto di proposito
Abbiamo toccato solo i due file degli adapter. **Non** abbiamo cambiato il dominio, la porta,
l'adapter BC o il fake in memoria — sono fissati. Persistere `Article.Inventories` è fuori scope in
questa fase (solo la riga dell'articolo). **Nessun `InventoryRepository`** — romperebbe il confine
dell'aggregato.

## Come l'abbiamo dimostrato
- **Test unitari (veloci, senza DB):** i test puri di conversione (`2999 ↔ "29.99"`, arrotondamento,
  round-trip) e i test del dual-write contro fake in memoria (scrivi-su-entrambi; legacy-fallisce →
  BC intatto; BC-fallisce → legacy tiene; lettura-instradata-dalla-modalità) — tutti verdi.
- **End-to-end contro i due MySQL reali**, via la CLI `seed` (che esegue gli stessi
  `dual.Save`/`dual.FindByID` che un handler chiamerà in CP6):
  - `seed write` → `✓ legacy write OK` / `✓ BC write OK`; `legacy.price=29.99, bc.price_cents=2999 EUR`
  - `seed read` → `price=2999 EUR` (mode=legacy)
  - `seed compare` → `OK: stores aligned`

Stesso concetto, due forme, una vista di dominio pulita — è l'ACL che funziona.

## Come si esegue
```bash
cd phase-04-db
docker compose run --build --rm test                                        # tutti i package verdi
docker compose --profile demo run --build --rm seed write demo-1 ABC-001 "Widget" 2999 EUR
docker compose --profile demo run --rm seed read    demo-1                  # price=2999 EUR
docker compose --profile demo run --rm seed compare demo-1                  # OK: stores aligned
docker compose down                                                         # spegni lo stack
```
Test unitari senza Docker: `go test ./...`.

## Chiusura
La porta di CP3 ora ha adapter reali e la cucitura è **viva**: ogni scrittura atterra in entrambi gli
store, le letture arrivano da uno, e l'ACL tiene il disordine legacy (DECIMAL/senza valuta) fuori dal
dominio. Prossimo passo (CP5): use case + un dispatcher che drena gli eventi di dominio, sopra questi
repository. Domande?

## Glossario
- **Adapter** — implementazione concreta di una porta contro uno store.
- **Anti-Corruption Layer (ACL)** — un adapter il cui compito è contenere il disordine legacy così che
  non raggiunga mai il dominio pulito (qui: DECIMAL↔centesimi, inventa/elimina la valuta).
- **Porta (port)** — l'interfaccia `ArticleRepository` (senza implementazione) da cui dipende il dominio.
- **Dual-write** — scrittura su entrambi gli store a ogni mutazione (legacy per primo).
- **Single-read** — letture servite da un solo store, scelto dalla modalità.
- **Cutover** — il passaggio da `ReadFromLegacy` a `ReadFromBC` quando il nuovo store è affidabile.
- **Upsert idempotente** — salva-due-volte-stesso-stato = una riga, nessun errore, nessun duplicato.
- **Strangler Fig** — estrai un BC in modo incrementale mentre il monolite continua a servire.
