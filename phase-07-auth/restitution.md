# Restitution — Phase 07: authentication and the first policy (Group 3)

> Versione italiana: [`restitution-IT.md`](./restitution-IT.md)
> Spoken restitution script for CP7 (~4 min). We compare the shape, not the syntax.

## Opening
We're Group 3. In CP6bis we had the Strangler cut: reads and creates already flowed through the BC. CP7
wraps an **identity envelope** around the BC API. The cut is **broken on purpose**: the Warehouse BC now
sits behind an auth middleware and answers **401** to anyone without a valid JWT. The MIC UI calls
`/api/articles` with **no token**, so the migrated articles were down. The whole identity machinery is
**given**: a C# IAM mock (demo users **Alice** and **Bob**) signs RS512 tokens; the BC verifies the
signature via JWKS, runs the claim checklist, builds an `AuthContext`, and mirrors `X-TS-ID` /
`X-Workspace-ID` on every response. The **CP4 dual-write** still runs underneath. Two things were missing,
both ours.

## Part 1 — teach MIC to log in (JavaScript)
In `auth-overlay.js` (injected by the facade before MIC's `app.js`, so the monolith stays untouched) we
completed two spots. The **`window.fetch` wrapper**: on same-origin `/api/*` calls it adds
`Authorization: Bearer <token from sessionStorage>` and `X-Workspace-ID`, cloning `init` without mutating
it, and leaving `/iam/*` and external calls alone. And **`loginDemoUser`**: form-urlencoded `POST` to
`/iam/oauth/token` (`password` grant, user `client_id`, openid `scope`), storing the `access_token`. This
**repairs the 401**: after login the list loads and responses carry the identity. But Alice and Bob are
still indistinguishable — authentication says *who you are*, not *what you may do*.

## Part 2 — the first authorization policy (Rego)
The decision is not in Go: it lives in `policies/warehouse.rego`, which the BC **compiles in**
(`go:embed`) and queries on every guarded route. It started at `default allow := false` — it denied
everything. We added three rules: `article:read` for **every** authenticated caller; `article:create`
only for principals in `article_creators` (users identified by **email**, M2M services by **service_id**,
told apart by `principal.kind`); any other action **denied**. Note the family resemblance with the auth
checklist: the bouncer checked which *application* asked for the token; the policy checks what *this
identity* may do. Same shape — a list and a default — one level up.

## The three questions: 401, 403, 404
This is the thread of the phase. **401** asks *who are you?* (no identity). **403** says *I know you, and
no* (valid identity, missing permission). **404**, with a valid token, says *you're in, it does not
exist*. Three different questions — authentication, authorization, existence — and in this phase we
touched all three.

## How we proved it
- **Tests** (`docker compose run --build --rm test`): all green, including `TestCan` (7/7), the policy
  spec that shipped red.
- **End-to-end** (facade → adapter → BC → dual-write): no token `401`; Alice and Bob read `200`;
  **Alice creates `201`** (same id in **both** stores — `19.99` DECIMAL in legacy, `1999` cents in
  warehouse), **Bob creates `403`**, and the M2M service `svc-orders` creates `201`. Alice's responses
  carry `X-TS-ID` with her `sub` and `X-Workspace-ID: ws-acme`.
- `git diff` touches **only** the two files (`auth-overlay.js`, `warehouse.rego`).

## The five questions (Q1–Q5)
**Q1 — why start from the 401 instead of shipping the overlay already working?** Because the 401 makes
the seam *visible*: turning the boundary on is a real breaking change — it breaks every client that isn't
sending identity. You observe the failure first, then repair it while understanding *which* identity
travels and why. A pre-working overlay would have hidden exactly the lesson.

**Q2 — why propagate the user's token instead of letting the adapter mint an M2M token for everything?**
Because the user token preserves the real identity end-to-end: Alice stays Alice all the way to the BC,
into `X-TS-ID` and the audit trail. A single adapter-minted M2M token would flatten everyone onto the same
service — you'd lose *who* did *what*, and the policy couldn't tell Bob from Alice. It also keeps the
adapter from becoming an over-privileged *confused deputy*. User token = least privilege + accountability.

**Q3 — 401/403/404, which question does each answer and where did we meet them today?** 401 = *who are
you?* (before login, no token). 403 = *I know you, and no* (Bob trying to create). 404 = *you're in, it
does not exist* (valid token on a missing id). Authentication, authorization, existence.

**Q4 — why is deny-by-default the only safe default, and where had we already met it?** Because every
unforeseen case (a new action, a typo, a forgotten principal) **fails closed**, not open: a forgotten gap
denies rather than grants. The idea was already there twice: the auth allowlist (the bouncer only admits
listed applications) and `policy.go`, which **fails closed** on evaluation errors — and even panics if the
policy does not compile, rather than silently denying.

**Q5 — the rules are already Rego, Phase 08's policy-engine language: what is left for Phase 08 to
change?** Not the language, not the rules: their **home**. Today `warehouse.rego` is compiled into the BC
— changing a rule (adding Bob to the creators, a per-tenant policy) needs a **rebuild and redeploy** of the
BC. In Phase 08 the same Rego is served by an external engine (Policy Manager): versioned, auditable, and
changeable **without rebuilding** the BC. What changes is the policy's *lifecycle*, not its rules.

## How to run it
```bash
cd phase-07-auth
docker compose -f mic-integration/docker-compose.auth.yml up -d --build   # BC (Go) + IAM mock (C#) + MIC + facade
docker compose -f mic-integration/docker-compose.auth.yml --profile tools run --rm backfill-articles
docker compose run --build --rm test                                       # green suite (incl. TestCan)
#  UI: http://localhost:8081/#/articles  ->  Login Alice/Bob  ->  create (201 / 403)
docker compose -f mic-integration/docker-compose.auth.yml down
```

## Closing
The extracted BC now has an identity envelope: a valid JWT opens the boundary (authentication), and a Rego
policy decides what that identity may do (authorization), with CP4's dual-write persistence still
underneath. The same rules, in Phase 08, will leave the binary for an external policy engine. Questions?

## Glossary
- **Authentication vs authorization** — *who you are* (valid JWT, 401 if not) vs *what you may do* (policy, 403 if not).
- **JWKS** — the public-key set the BC uses to verify token signatures without ever knowing IAM's private key.
- **AuthContext** — the identity extracted from the token (kind user/M2M, email, service_id) the policy decides on.
- **Rego / OPA** — the declarative policy language; the BC evaluates it in-process and reads `allow`.
- **Deny-by-default** — anything not explicitly allowed is denied: forgotten cases fail closed.
- **User vs M2M token** — a user carries their own identity end-to-end; an M2M token identifies a service (`svc-orders`).
