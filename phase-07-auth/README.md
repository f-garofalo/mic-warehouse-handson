# Phase 07 — Authentication, and the first policy

```
    _    _   _ _____ _   _
   / \  | | | |_   _| | | |
  / _ \ | | | | | | | |_| |
 / ___ \| |_| | | | |  _  |
/_/   \_\\___/  |_| |_| |_|

  Phase 07 — Identity envelope around the BC API.
```

Estimated time: ~1h in breakout rooms, then a shared restitution.

```text
CP1-CP5   Domain, adapters, use cases, first HTTP API
CP6bis    You built the Strangler facade; reads and creates flow through the BC
CP7       Who are you? (login) — and what may YOU do? (the first policy)   <-- you are here
CP8+      Externalized policies (Policy Manager), events, observability
```

> **Plenary opening (before you start):** we walk the identity chain on slides — JWT anatomy,
> JWKS validation, user vs M2M tokens, correlation headers, and the line between
> authentication and authorization. Then you build.

---

## Your mission

The Strangler cut from Phase 06bis is **broken on purpose**: the Warehouse BC now sits behind
an auth middleware and answers `401` to anyone without a valid JWT. The MIC UI still calls
`/api/articles` with no token, so the migrated article operations are down.

The whole identity machinery is **given**: a local IAM mock (C#, a black box that behaves
like a TSID-style issuer, with two demo users: **Alice Demo** and **Bob Reader**) signs RS512
tokens; the BC verifies the signature through JWKS, runs the claim checklist, builds an
`AuthContext`, and mirrors `X-TS-ID` / `X-Workspace-ID` on every response. The BC also still
runs your Phase 04 dual-write underneath. Two things are missing, both yours:

1. **the login on the MIC side** (JavaScript, Part 1): the UI must obtain a token and send it —
   this repairs the cut and answers *who are you?*;
2. **the first authorization policy** (Rego, Part 2): a valid token opens the boundary, but it
   does not say what the caller may *do*. Alice manages the warehouse; Bob reads it. The
   policy is where that difference lives.

```
MIC UI ──/api/articles (no token)──> facade ──> adapter ──> BC ──> 401       (Part 1 repairs this)
MIC UI ──/iam/oauth/token──────────> IAM mock ──> access_token
MIC UI ──/api/articles + Bearer────> BC: signature + checklist (given) ──> 200
MIC UI ──POST /api/articles + Bearer──> BC: authenticated, but... may YOU create?   (Part 2 decides this)
```

| Given / starter | Where |
|---|---|
| IAM mock: `/oauth/token`, OpenID config, JWKS; users `alice` and `bob` (password `demo`) | `iam-mock/` (black box) |
| JWT validation: signature via JWKS + claim checklist | `middleware/auth.go` (given) |
| `AuthContext` + correlation headers on every response | `middleware/` (given) |
| Your Phase 04 dual-write + legacy ACL, under the API | `repositories/`, `main.go` (given) |
| Login overlay injected before MIC's `app.js` | `mic-integration/auth-overlay.js` — **STARTER, Part 1** |
| The authorization policy | `policies/warehouse.rego` — **STARTER, Part 2** |
| Policy evaluation: OPA embedded in the BC, `Can` builds the input document | `policies/policy.go`, `middleware/policy.go` (given) |
| Test suite (policy tests ship red) | `middleware/*_test.go` |
| Integrated stack: MIC + IAM + BC + adapter + facade + Adminer | `mic-integration/docker-compose.auth.yml` |

---

## The consegna

> This file is self-sufficient: everything you need for the hour is here. Start the stack
> build first ([Run](#run): it compiles Go and C#, it takes minutes) — the Part 2 policy
> tests need no stack, so you can flip the two parts if the build is slow.

### ▸ Part 1 — Teach MIC to log in (`mic-integration/auth-overlay.js`)

1. **Observe the failure.** With the stack up, run
   `curl.exe -i "http://localhost:8081/api/articles?limit=3"` (plain `curl` on Bash).
   Expected: `401` with `warehouse_bc_auth_required`. Open `http://localhost:8081/#/articles`:
   the Articoli screen shows an authentication error. Routing works; identity is missing.
2. **Complete the two TODOs** in the overlay (the facade injects it before MIC's `app.js`,
   so the monolith itself stays untouched):
   - the `window.fetch` wrapper: for same-origin `/api/*` calls, add
     `Authorization: Bearer <token from sessionStorage>` and `X-Workspace-ID`; leave `/iam/*`
     and external calls alone; never mutate the caller's `init` object;
   - `loginDemoUser`: POST form-urlencoded to `/iam/oauth/token` with
     `grant_type=password`, `username`, `password=demo`, `client_id` (the constant in the
     file), `scope=openid profile offline_access`; store `access_token` in sessionStorage
     under `TOKEN_KEY`; reload the page. (The **Login Alice** and **Login Bob** buttons both
     call this function.)

   Write your own AI prompt from this checklist — if it does not say what must NOT change, narrow it:

   | Element | Must appear in the prompt |
   |---|---|
   | Modifiable file | `mic-integration/auth-overlay.js`, the two TODOs only |
   | Token endpoint + grant | `/iam/oauth/token`, form-urlencoded `password` grant |
   | Storage | `sessionStorage`, keys already defined in the file |
   | Forbidden | adapter, monolith files, Go code, tests |
   | Required AI output | show the diff |

3. **Verify.** `git diff` must touch only the overlay. Restart the facade, then use the UI:

   ```bash
   docker compose -f mic-integration/docker-compose.auth.yml restart strangler-facade
   ```

   Reload `http://localhost:8081/#/articles`, click **Login Alice**: the list loads. In
   DevTools Network the requests carry `Authorization`, and the responses carry `X-TS-ID`
   (Alice's identity, derived from the token's `sub`) and `X-Workspace-ID`. Log out, log in
   as **Bob**: the list loads for him too — authentication does not distinguish them. Yet.

✅ **Green when:** before login the Articoli screen fails with `401`; after **Login Alice**
(or Bob) it loads, with the identity visible in the response headers.

### ▸ Part 2 — Write the first policy (`policies/warehouse.rego`)

Alice is the warehouse manager; Bob reads reports. Both are *authenticated* — but should Bob
be able to **create** articles? Try it: logged in as Alice, use **Nuovo articolo** in the UI
(or any create request). Expected right now: `403 forbidden` — **even for Alice**. The policy
exists, it is wired on the create route (the BC compiles `policies/warehouse.rego` in and asks
it for every decision), and it denies everything, because nobody wrote the rules yet:

```rego
default allow := false # deny everything until you build it
```

The policy is not Go: it is **Rego**, the policy language of OPA (Open Policy Agent), the same
language TeamSystem policies are written in. The BC sends it an input document (`action` +
`principal`, the shape is in the file's header comment) and reads back `allow`. Note the family
resemblance with the auth checklist: the bouncer checked which *application* asked for the
token; the policy checks what *this identity* may do. Same shape — a list and a default — one
level up, and this time the default is a literal line. The rules (the TODO comment in the file
spells them out):

1. no identity, no decision: when the input carries no `principal`, no rule may match, so
   the default answers. Deny by default covers this case for free: keep it that way;
2. `article:read`: every authenticated caller may read;
3. `article:create`: only principals in the `article_creators` set — users are identified by
   **email**, M2M services by **service id**. Alice and `svc-orders` are on it; Bob is not;
4. anything else: denied. **Deny by default** is the only safe default for authorization.

The specification is `middleware/policy_test.go` (ships red; the Go tests call your Rego
through `Can`). Your own prompt, from this checklist:

| Element | Must appear in the prompt |
|---|---|
| Modifiable file | `policies/warehouse.rego`, adding rules under the given `default` |
| The rules | the four rules above, plus the input document shape from the file header |
| Forbidden | tests, all Go code, the rest of `middleware/` and `policies/`, `iam-mock/` |
| Required AI output | show the diff |

Then run the suite and rebuild the BC with your policy in it:

```bash
docker compose run --build --rm test
docker compose -f mic-integration/docker-compose.auth.yml up -d --build warehouse-bc
```

**Verify from the UI**, and read the status codes like an operator:

- **Login Alice** → **Nuovo articolo** → the create succeeds (`201`): it went facade →
  adapter → BC → your Phase 04 dual-write, and the new article is in **both** stores
  (Adminer: same id in `mic.business_data` and `warehouse_db.articles`);
- **Login Bob** → **Nuovo articolo** → `403 forbidden`: valid token, wrong permissions.
  Bob can still read the list: `200`;
- no login → `401`: no identity at all.

`401` asks *who are you?* — `403` says *I know you, and no* — `404`, with a valid token,
says *you are in, it does not exist*. Three different questions, and now you have touched
all three.

✅ **Green when:** the whole suite is green, `git diff` touches only `policies/warehouse.rego`,
Alice creates, Bob reads but cannot create.

### ▸ Done when

- [ ] Before login: `/api/articles` fails with `401 warehouse_bc_auth_required`.
- [ ] After login: the list loads for Alice **and** Bob, responses carry `X-TS-ID` / `X-Workspace-ID`.
- [ ] `docker compose run --build --rm test` fully green; the diff touches only `policies/warehouse.rego`.
- [ ] Alice creates an article (`201`, same id in both stores); Bob gets `403` on create and `200` on read.
- [ ] You can answer Q1-Q5 below.

---

## Run

> Prerequisites: Rancher Desktop with the Docker engine. No Go and no .NET needed locally.

**Step 0 — free the ports** (this phase binds 8081, 8082, 9001, 3306):

```bash
cd ../phase-06bis-strangler && docker compose -f mic-integration/docker-compose.strangler.yml down
cd ../phase-07-auth
docker ps --filter "publish=8081" --filter "publish=9001" --filter "publish=3306"   # expect no rows
```

**Step 1 — start the authenticated stack** (first build compiles the Go BC *and* the C# IAM
mock: start it, then read the given-code map above while it builds):

```bash
docker compose -f mic-integration/docker-compose.auth.yml up -d --build
docker compose -f mic-integration/docker-compose.auth.yml ps
```

Expected: `integration-mysql` healthy; `iam-mock`, `mic-app`, `warehouse-bc`,
`mic-compat-adapter`, `strangler-facade`, `adminer` up.

**Step 2 — backfill the warehouse** (one-time copy, same ids as legacy):

```bash
docker compose -f mic-integration/docker-compose.auth.yml --profile tools run --rm backfill-articles
```

> **Corporate proxy (Netskope):** on `x509: certificate signed by unknown authority` build
> errors, set `NETSKOPE_CA_BUNDLE` to your `netskope-cert-bundle.pem` path and append
> `-f mic-integration/docker-compose.netskope.yml` to build/up commands (for the test runner:
> `-f docker-compose.netskope.yml`).

Shut down at the end with `docker compose -f mic-integration/docker-compose.auth.yml down`.

---

## Scope & how to work

You may modify exactly two things: the two TODOs in `mic-integration/auth-overlay.js`
(Part 1) and the rules in `policies/warehouse.rego` (Part 2). Everything else — tests, all
the Go code (`middleware/`, `policies/policy.go`, handlers), `iam-mock/`, adapter, monolith —
is given and off limits. If the AI proposes touching them, **stop**.

### Two hints while you work

> 💡 **Watch the identity travel.** DevTools Network is your instrument: the `Authorization`
> header goes out, `X-TS-ID` comes back. When something is denied, check *which* question
> failed: no header at all (`401`, Part 1 territory) or the wrong principal (`403`, Part 2
> territory)?

> 💡 **Turn the code into a language you know.** The overlay is plain JavaScript and the
> policy is Rego, a declarative language — if either is opaque, ask the AI to restate it as
> plain if/else pseudocode (or C#, Java, TypeScript) first, understand it there, then come back.

Hold for restitution: **Q1** why is the initial `401` useful in the teaching path, instead
of shipping the overlay already working? **Q2** why does the overlay propagate the *user's*
token instead of letting the adapter mint an M2M token for everything? **Q3** `401`, `403`,
`404`: which question does each one answer, and where did you meet each today? **Q4** why is
deny-by-default the only safe default for a policy — and where did you already meet that idea
in this phase? **Q5** your rules are already Rego, the language of Phase 08's policy engine:
what is left for Phase 08 to change?

---

## Optional stretch — poke the identity machinery

Finished early? Pick one:

- **Look inside the trust machinery.** The IAM mock is exposed on `http://localhost:9001`:
  inspect `/.well-known/openid-configuration` and `/.well-known/jwks.json`, then decode your
  token's payload (paste it into an AI chat or any JWT decoder: it is signed, not secret) and
  find the claims the checklist reads. Checkpoint: what lets the BC validate tokens without
  ever knowing IAM's private key?
- **Be a machine.** Get a `client_credentials` token for `svc-orders` (secret `demo`, scope
  `platform.m2m`) from `http://localhost:9001/oauth/token` and create an article with it:
  it works, because the *service* is on the creators list — and `X-TS-ID` says `svc-orders`,
  not a person. Who should be on that list in a real warehouse?
- **Promote Bob.** Add Bob's email to `article_creators` in the policy file, rebuild the BC,
  and watch his `403` become `201`. Then revert, and ask: who is allowed to edit this file in
  production, and why should changing it not require a rebuild? (That question *is* Phase 08.)

---

## Solutions

Reference implementations: [`solutions/auth_overlay.expected.js`](./solutions/auth_overlay.expected.js)
(Part 1), [`solutions/warehouse.expected.rego`](./solutions/warehouse.expected.rego)
(Part 2), walkthrough and Q1-Q5 answers in [`solutions/README.md`](./solutions/README.md).
**Open them only after your own green** — use as a checklist, not a copy source.

---

## Next phase

→ **Phase 08** keeps your rules and changes their *home*: today `warehouse.rego` is compiled
into the BC; there it is served by a policy engine — versioned, auditable, and changeable
without rebuilding the BC.

---

*MIC v0.1 - by Mic (Michele Mondora) - 2026*
