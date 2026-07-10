# Phase 11 — MCP: a tool surface for AI agents


```
  __  __  ____ ____
 |  \/  |/ ___|  _ \
 | |\/| | |   | |_) |
 | |  | | |___|  __/
 |_|  |_|\____|_|

  Phase 11 — the Warehouse BC speaks tools.
```

Estimated time: ~45min in breakout rooms, then a shared restitution.

```text
CP1-CP5   Domain, adapters, use cases, first HTTP API
CP6bis    The Strangler facade: reads and creates flow through the BC
CP7       Login and the first policy: who are you, what may YOU do
CP11      A new kind of caller: AI agents, through MCP   <-- you are here
```

> **Plenary opening (before you start):** on slides we walk what MCP is, why a
> tool's schema and description are a contract whose reader is a model, and why
> the agent gets a machine identity instead of a back door. Then you build.

---

## Your mission

A new kind of caller wants your Warehouse BC: **AI agents**. They do not learn
your REST API from a wiki page; they speak **MCP** (Model Context Protocol):
they connect to a server, read the catalogue of **tools** it exposes, and
decide on their own, mid-conversation, whether and how to call them.

The server itself is **given** and deliberately thin: JSON-RPC in,
authenticated HTTP against the BC out. It holds no business logic and **no
permission logic**: it authenticates with the `client_credentials` grant as
`svc-warehouse-agent`, exactly like any service from Phase 07, and the policy
**you** wrote decides what that identity may do. Look at
[`policies/warehouse.rego`](./policies/warehouse.rego): one line in
`article_creators` is the entire "may the agent write?" decision. The BC
never learns that an AI is calling.

What is missing are the **tools**. One is given as the worked example
(`get_article`); you write two more (plus a flex). The new skill is not the
plumbing: it is writing a contract whose reader is a model. A vague
description produces wrong calls, and an error message that only says "failed"
leaves the agent stuck.

| Given / starter | Where |
|---|---|
| The Warehouse BC of Phase 07: auth, your policy, your Phase 04 dual-write | root of this phase (given) |
| IAM mock with the M2M client `svc-warehouse-agent` | `iam-mock/` (black box) |
| MCP server scaffold: stdio transport, M2M token, authenticated BC client | `mcp-server/main.go`, `mcp-server/client.go` (given) |
| The worked example tool | `mcp-server/tools_get.go` (given) |
| `list_articles` | `mcp-server/tools_list.go` — **STARTER, Task 1** |
| `create_article` | `mcp-server/tools_create.go` — **STARTER, Task 2** |
| `adjust_inventory` | `mcp-server/tools_adjust.go` — **STARTER, flex** |
| Test suite (Task 1, Task 2 and flex ship red) | `mcp-server/tools_*_test.go` |
| Ready-made agent configs | `.mcp.json`, `.vscode/mcp.json` |

---

## The consegna

> This file is self-sufficient: everything you need for the hour is here.
> Start the stack first ([Run](#run): it compiles Go and C#, it takes
> minutes) — the tests need no stack, so you can write code while it builds.

### ▸ Task 1 — `list_articles` (`mcp-server/tools_list.go`)

The BC endpoint `GET /articles` returns everything and takes no filters: fine
for a program, hostile for a conversation. Your tool shapes the surface.

1. **Read the worked example** (`tools_get.go`): every tool has the same
   three parts — input/output structs (the schema the agent reads), a
   registration with the description, a handler that calls the BC.
2. **Complete the three TODOs**: the two `jsonschema` field descriptions, the
   tool description, and the handler (fetch all, filter case-insensitive on
   SKU and name, cap at limit, default 20). The specification is
   `tools_list_test.go`.

   Write your own AI prompt from this checklist — if it does not say what must
   NOT change, narrow it:

   | Element | Must appear in the prompt |
   |---|---|
   | Modifiable file | `mcp-server/tools_list.go` only |
   | The contract | filter and default-limit behavior, from the tests |
   | The reader | descriptions are read by a model deciding how to call |
   | Forbidden | tests, `client.go`, `main.go`, `tools_get.go`, the BC |
   | Required AI output | show the diff |

3. **Verify.** `docker compose run --build --rm test` — the Task 1 tests go
   green. Then rebuild the server and reconnect the agent:

   ```bash
   docker compose --profile mcp build mcp-server   # then /mcp → reconnect
   ```

   Ask it, in your own words, what is in the warehouse and which articles
   mention a word you pick. Watch **which parameters it chooses**: that is
   your schema working, or failing.

✅ **Green when:** the Task 1 suite is green and the agent answers catalogue
questions with real warehouse data, choosing `query` and `limit` on its own.

### ▸ Task 2 — `create_article` (`mcp-server/tools_create.go`)

A tool that **writes**. The description now carries real weight: the agent
must understand from your text that this has a permanent side effect, when it
is appropriate, and that `price_cents` is integer cents, never a decimal.

1. **Complete the three TODOs** (same shape as Task 1). Note in the handler
   spec: empty SKU or name is rejected **without calling the BC**, the request
   carries **no id** (the system of record mints it, Phase 04), and a `403` is
   translated into an error that names the policy — a denial is something the
   agent should explain, not retry. The specification is
   `tools_create_test.go`.
2. **Verify with tests**, then with the agent: reconnect and ask it to create
   a test article with a price in euros. It must convert to cents on its own —
   your schema told it to.
3. **See the write land.** Adminer (`http://localhost:8082`, server
   `integration-mysql`, root/root): the same minted id appears in
   `mic.business_data` AND `warehouse_db.articles`. The agent's create went
   through your policy and your dual-write like any Phase 07 caller.

✅ **Green when:** the whole non-flex suite is green, the agent creates an
article conversationally, and the row is in both stores with the same id.

### ▸ Flex task — `adjust_inventory` (skip if behind schedule)

Same pattern a third time, on
`POST /articles/{id}/inventory/adjust`. Nothing later depends on it: if the
hour is running out, go to the restitution questions instead. Specification:
`tools_adjust_test.go`.

### ▸ Done when

- [ ] `docker compose run --build --rm test` green (flex tests excluded if skipped).
- [ ] `git diff` touches only the three starter files.
- [ ] The agent lists and searches articles with real data (Task 1).
- [ ] The agent creates an article; Adminer shows the same id in both stores (Task 2).
- [ ] You can answer Q1-Q5 below.

---

## Run

> Prerequisites: Rancher Desktop with the Docker engine, and an MCP-capable
> AI agent. No Go and no .NET needed locally.

**Step 0 — free the ports** (this phase binds 8082, 8083, 9001):

```bash
cd ../phase-07-auth && docker compose -f mic-integration/docker-compose.auth.yml down
cd ../phase-11-mcp
docker ps --filter "publish=8083" --filter "publish=9001"   # expect no rows
```

**Step 1 — start the stack and warm the server build:**

```bash
docker compose up -d --build
docker compose --profile mcp build mcp-server
```

**Step 2 — load the warehouse** (one-time copy of the legacy articles, same
ids, as in Phase 07):

```bash
docker compose --profile tools run --rm backfill-articles
```

**Step 3 — connect your agent.** The server speaks stdio: the agent starts it
by itself with the command in the config, one process per session. The stack
must already be up.

- **Claude Code (CLI):** open the agent **from this folder** — it picks up
  [`.mcp.json`](./.mcp.json); approve the `warehouse-bc` server when asked.
- **VS Code (Copilot):** open **this folder**; [`.vscode/mcp.json`](./.vscode/mcp.json)
  registers the same server.

The agent runs the image built in Step 1: after every change to your tool
code, rebuild it (`docker compose --profile mcp build mcp-server`) and
reconnect the server (`/mcp` → reconnect). Check the connection by listing
the MCP tools in your agent (`/mcp` in the CLI): four tools, one working
(`get_article` — try "show me article 100"), three waiting for you.

> **Corporate proxy (Netskope):** on `x509: certificate signed by unknown
> authority` build errors, set `NETSKOPE_CA_BUNDLE` to your
> `netskope-cert-bundle.pem` path and append
> `-f docker-compose.netskope.yml` to build/up commands.

Shut down at the end with `docker compose down`.

---

## Scope & how to work

You may modify exactly three files: `tools_list.go`, `tools_create.go` and
`tools_adjust.go` in `mcp-server/`. Everything else — `client.go`, `main.go`,
`tools_get.go`, the tests, the BC, `iam-mock/` — is given and off limits. If
the AI proposes touching them, **stop**.

### Two hints while you work

> 💡 **The agent is your integration test.** After every green suite, rebuild
> and reconnect, then ask the agent to use your tool and watch which
> parameters it picked. If it guesses wrong, the bug is in your description,
> not in the model: sharpen the text, rebuild, reconnect. You are debugging
> prose.

> 💡 **Turn the code into a language you know.** Everything here is Go — if a
> snippet is opaque, ask the AI for an equivalent in C#, Java, TypeScript, or
> pseudocode first, understand it there, then come back.

Hold for restitution: **Q1** the MCP server contains zero permission checks —
where is "may the agent create?" decided, which single line grants it, and why
is that better than a check inside the MCP server? **Q2** who reads your
`jsonschema` descriptions, and when — and what breaks when they are vague:
compile time, call time, or conversation time? **Q3** the BC was not touched
in this phase; what does it see when the agent calls, and which phase built
that machinery? **Q4** why do the tests insist that error messages name the
failing id and suggest `list_articles` — who is the reader of an error? **Q5**
the agent acted with its own service identity; what would still be missing
before letting it act **on behalf of a user** in production?

---

## Solutions

Reference implementations:
[`solutions/tools_list.expected.go.txt`](./solutions/tools_list.expected.go.txt),
[`solutions/tools_create.expected.go.txt`](./solutions/tools_create.expected.go.txt),
[`solutions/tools_adjust.expected.go.txt`](./solutions/tools_adjust.expected.go.txt),
walkthrough and Q1-Q5 answers in [`solutions/README.md`](./solutions/README.md).
**Open them only after your own green** — use as a checklist, not a copy source.

---

*MIC v0.1 - by Mic (Michele Mondora) - 2026*
