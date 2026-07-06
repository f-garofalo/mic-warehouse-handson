# MIC Local Route Map

This file lists the MIC routes that matter for the Phase 06bis Strangler lab.
The lab changes only the list read route; all other MIC routes stay on the
local monolith.

| Route | Owner in this lab | Why it matters |
|---|---|---|
| `GET /api/articles` | Strangler facade can route to monolith or adapter | First read path cutover |
| `POST /api/articles` | MIC monolith | Writes stay legacy in this phase |
| `GET /api/articles/:id` | MIC monolith | Item reads are outside this first cut |
| `PUT /api/articles/:id` | MIC monolith | Update migration needs write-safety work |
| `DELETE /api/articles/:id` | MIC monolith | Delete migration needs write-safety work |
| Other `/api/*` routes | MIC monolith | Other MIC screens are outside the blast radius |
