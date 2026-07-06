# Mappa route MIC locale

Questo file elenca le route MIC che contano per il lab Strangler di Phase 06bis.
Il lab cambia solo la read della lista articoli; tutte le altre route MIC restano
sul monolite locale.

| Route | Owner in questo lab | Perche' conta |
|---|---|---|
| `GET /api/articles` | La Strangler facade puo' instradare al monolite o all'adapter | Primo cutover read-only |
| `POST /api/articles` | Monolite MIC | Le write restano legacy in questa phase |
| `GET /api/articles/:id` | Monolite MIC | Le item read sono fuori da questo primo taglio |
| `PUT /api/articles/:id` | Monolite MIC | La migrazione update richiede lavoro di write safety |
| `DELETE /api/articles/:id` | Monolite MIC | La migrazione delete richiede lavoro di write safety |
| Altre route `/api/*` | Monolite MIC | Le altre schermate MIC sono fuori dal blast radius |