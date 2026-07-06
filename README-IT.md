# MIC → Warehouse BC: login e la prima policy (Lezione 9 · Fase 7)

> English version: [`README.md`](./README.md)

Un lab pratico che continua l'estrazione del Bounded Context **Warehouse** da **MIC**, un monolite PHP
legacy di fatturazione, verso un microservizio **Go** pulito, con un **agente di coding AI** come motore.
La Lezione 7 ha costruito il dominio (Fasi 1–3), la Lezione 8 ha aggiunto persistenza e una prima API
HTTP (Fasi 4–5); la **Lezione 9** mette il BC davanti al traffico reale di MIC: costruisci la
**Strangler facade** e operi il primo cutover.

Questo branch è il lab per la **Lezione 9 — Fase 7** (la busta di identità: login, e la prima policy
di autorizzazione) e include la Fase 6bis e tutto ciò che la precede. Il lavoro lo fai tu; le slide
non ti danno la risposta.

## Com'è organizzata la repo

La Lezione 9 è divisa per fase, ogni fase come **due branch** (punto di partenza + soluzione di riferimento):

| Branch | Cos'è |
|---|---|
| `lezione-9-fase-6bis` | Punto di partenza della Fase 6bis (include le Fasi 1–5). Senza soluzioni. |
| `lezione-9-fase-6bis-soluzione` | Fase 6bis con le soluzioni svolte. |
| `lezione-9-fase-7` | Punto di partenza della Fase 7 (include la Fase 6bis). Senza soluzioni. |
| `lezione-9-fase-7-soluzione` | Fase 7 con le soluzioni svolte. |

Fai prima il tuo lavoro. Vai al branch soluzione solo dopo.

## I checkpoint

| Checkpoint | Cartella | Cosa fai |
|---|---|---|
| CP1 — Capire | [`phase-01-monolith/`](./phase-01-monolith/README-IT.md) | *(Lezione 7)* Avvia MIC e mappalo. |
| CP2 — Decidere | [`phase-02-analysis/`](./phase-02-analysis/README-IT.md) | *(Lezione 7)* Analisi DDD → il BC **Warehouse**. |
| CP3 — Costruire | [`phase-03-skeleton/`](./phase-03-skeleton/README-IT.md) | *(Lezione 7)* Il domain layer in Go. |
| CP4 — Persistere | [`phase-04-db/`](./phase-04-db/README-IT.md) | *(Lezione 8)* Adapter reali (**ACL** legacy) + il decorator **dual-write**. |
| CP5 — Servire | [`phase-05-usecases/`](./phase-05-usecases/README-IT.md) | *(Lezione 8)* Use case e una API HTTP essenziale. |
| CP6bis — Instradare | [`phase-06bis-strangler/`](./phase-06bis-strangler/README.md) | Scrivi la **decisione di routing** della facade, poi operi il cutover. |
| **CP7 — Proteggere** | [**`phase-07-auth/`**](./phase-07-auth/README.md) | **Questa lezione:** la **busta di identità** (insegna a MIC il login, JWT/JWKS) e la **prima policy** (Alice crea, Bob legge). |

**Da dove partire:** apri [`phase-07-auth/README.md`](./phase-07-auth/README.md).
La Fase 6bis e tutto ciò che la precede sono inclusi come contesto.

## Usare un agente di coding AI

Gli agenti di coding AI fanno parte del metodo, non sono una scorciatoia per aggirarlo.

- **Avvia l'agente nella cartella giusta.** Aprilo sulla cartella della fase su cui lavori
  (`phase-07-auth/`), non sull'intera repo, così vede il codice che conta.
- **Le conclusioni sono tue.** L'agente legge, abbozza e scrive sintassi; tu decidi il design, gli
  invarianti e cosa finisce nei tuoi deliverable.
- **Metti in discussione.** Quando afferma una regola, chiedi *"dove nel codice l'hai visto?"* prima di fidarti.

## Riferimenti

Le Architecture Decision Record dei pattern praticati in questo lab sono in [`docs/adr/`](./docs/adr/):
Strangler Fig (ADR-001), baseline PHP di MIC (ADR-006), clean architecture Go (ADR-007), Event Storming
(ADR-010), building block DDD (ADR-011), layer di Clean Architecture (ADR-002), Dual-Write (ADR-013),
Data Product su Hermes (ADR-014).
