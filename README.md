# MIC → Warehouse BC: a tool surface for AI agents (Lesson 10 · Phase 11)

A hands-on lab continuing the extraction of the **Warehouse** Bounded Context out of **MIC**, a legacy
PHP invoicing monolith, into a clean **Go** microservice, with an **AI coding agent** as your engine.
Lesson 7 built the domain (Phases 1–3), Lesson 8 added persistence and a first HTTP API (Phases 4–5),
Lesson 9 put the BC in front of real MIC traffic (Phases 6bis–7); **Lesson 10** retires the dual-write,
makes facts travel as events, and then gives the BC a tool surface for AI agents.

This branch is the lab for **Lesson 10 — Phase 11** (an **MCP server** in front of the BC: tools an
AI agent can discover and call, behind the Phase 07 identity and the policy you wrote), and includes
Phase 9 and everything before it. You do the work; the slides do not hand you the answer.

## How this repo is organised

Lesson 10 is split by phase, each phase as **two branches** (starting point + reference solution):

| Branch | What it is |
|---|---|
| `lezione-10-fase-9` | Phase 9 starting point (includes Phases 1–7). No solutions. |
| `lezione-10-fase-9-soluzione` | Phase 9 with the worked solutions. |
| `lezione-10-fase-11` | Phase 11 starting point (includes Phase 9). No solutions. |
| `lezione-10-fase-11-soluzione` | Phase 11 with the worked solutions. |

Build your own work first. Reach for the solution branch only afterwards.

## The checkpoints

| Checkpoint | Folder | What you do |
|---|---|---|
| CP1 — Understand | [`phase-01-monolith/`](./phase-01-monolith/README.md) | *(Lesson 7)* Bring MIC up and map it. |
| CP2 — Decide | [`phase-02-analysis/`](./phase-02-analysis/README.md) | *(Lesson 7)* DDD analysis → the **Warehouse** BC. |
| CP3 — Build | [`phase-03-skeleton/`](./phase-03-skeleton/README.md) | *(Lesson 7)* The Go domain layer. |
| CP4 — Persist | [`phase-04-db/`](./phase-04-db/README.md) | *(Lesson 8)* Real adapters (legacy **ACL**) + a **dual-write** decorator. |
| CP5 — Serve | [`phase-05-usecases/`](./phase-05-usecases/README.md) | *(Lesson 8)* Use cases and a thin HTTP API. |
| CP6bis — Route | [`phase-06bis-strangler/`](./phase-06bis-strangler/README.md) | *(Lesson 9)* Write the facade's **routing decision**, then operate the cutover. |
| CP7 — Protect | [`phase-07-auth/`](./phase-07-auth/README.md) | *(Lesson 9)* The **identity envelope** (login, JWT/JWKS) and the **first policy**. |
| CP9 — Publish | [`phase-09-events/`](./phase-09-events/README.md) | *(This lesson, activity 1)* The dual-write retires; facts travel as Hermes Data Products. |
| **CP11 — Expose** | [**`phase-11-mcp/`**](./phase-11-mcp/README.md) | **This lesson:** write the BC's **MCP tools**: schema, description, handler; then talk to your warehouse through an AI agent. |

**Where to start:** open [`phase-11-mcp/README.md`](./phase-11-mcp/README.md).
Phase 9 and everything before it are included as context.

## Using an AI coding agent

AI coding agents are part of the method here, not a shortcut around it.

- **Start the agent in the right folder.** Open it on the phase folder you are working in
  (`phase-11-mcp/`), not the whole repo, so it sees the code that matters.
- **You own the conclusions.** The agent reads, drafts, and writes syntax; you decide the design, the
  invariants, and what goes into your deliverables.
- **Push back.** When it asserts a rule, ask *"where in the code did you see that?"* before you trust it.

## Reference

Architecture Decision Records for the patterns this lab practises live in [`docs/adr/`](./docs/adr/):
Strangler Fig (ADR-001), MIC PHP baseline (ADR-006), Go clean architecture (ADR-007), Event Storming
(ADR-010), DDD building blocks (ADR-011), Clean Architecture layers (ADR-002), Dual-Write (ADR-013),
Data Products on Hermes (ADR-014).
