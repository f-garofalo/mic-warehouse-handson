# Fase 01 — Deliverable (Breakout 1: capire MIC)

> English version: [`README.md`](./README.md)

I cinque artefatti richiesti da [`../README.md`](../README.md). Ognuno è fondato sulla sorgente
(codice + `database/schema.sql` + `database/seed.sql`), con evidenze `file:riga` inline.

| # | Artefatto | Risponde a | EN |
|---|---|---|---|
| 1 | [`page-map-IT.md`](./page-map-IT.md) | Ogni schermata e a cosa serve (raggruppata per dominio). | [EN](./page-map.md) |
| 2 | [`architecture-IT.md`](./architecture-IT.md) | Componenti in esecuzione + come scorre una richiesta browser → DB → ritorno. | [EN](./architecture.md) |
| 3 | [`repo-guide-IT.md`](./repo-guide-IT.md) | Cos'è ogni file/cartella e dove guardare per cambiare le cose. | [EN](./repo-guide.md) |
| 4 | [`coupling-map-IT.md`](./coupling-map-IT.md) | Come sono davvero salvati i dati, chi legge/scrive chi, accoppiamenti da rompere. | [EN](./coupling-map.md) |
| 5 | [`worst-antipatterns-IT.md`](./worst-antipatterns-IT.md) | Top 3 antipattern: cosa, perché fa male, costo di tenerli. | [EN](./worst-antipatterns.md) |

Extra: [`sintesi.md`](./sintesi.md) — sintesi unica dei 5 punti + le conclusioni (per la restitution).
Diagramma: [`architecture.svg`](./architecture.svg) (condiviso tra EN e IT).

## Le due scoperte che convergono

1. **Lo storage.** Tutti i ~18 tipi di entità vivono in **due tabelle generiche** (`business_data` + `business_relations`), distinti solo da una colonna di tipo; il significato di ogni colonna generica vive nei controller PHP, non nello schema. Dieci schermate ordinate, zero confini sotto.
2. **L'accoppiamento.** **Gli ordini entrano direttamente nelle righe dell'articolo** per prezzo (`articolo.amount_1`) e IVA (`articolo.text_2` → `aliquota_iva`) in `OrderController::addRiga` — la dipendenza che un'estrazione del Warehouse deve rompere per prima.

> Run dal vivo non catturato: il motore Docker non era in esecuzione in questo ambiente.
> Per verificare contro uno stack live: `docker compose up --build` da `phase-01-monolith/`,
> poi i due smoke check in `../README.md` (§ Step 3).
