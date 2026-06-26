# Warehouse BC — Design Decisions (CP3)

> Italian version: [`design-decisions-IT.md`](./design-decisions-IT.md)

The Go domain layer encodes architectural choices, most of them implicit. Each one below states the
**decision**, the **rule/boundary it protects**, the **alternative we rejected**, and **how it fits
the clean-architecture target** from CP2. We defend this design, not the syntax.

Scope: domain layer only (`entities/`, `events/`, `interfaces/`) + a domain-free health probe.
Persistence, use cases, the `/articles` HTTP surface, auth, etc. arrive in later checkpoints.

## 1. `Article` is the only aggregate root
- **Decision.** `Article` is the single aggregate root; `InventoryLevel` lives *inside* it and is
  only created/mutated through `Article` methods (`AdjustInventory`, `ReserveStock`).
- **Rule it protects.** `reserved ≤ quantity`, `quantity ≥ 0`, "one inventory level per location" —
  invariants that span an article and its levels must be enforced in one consistency boundary.
- **Rejected.** Three separate aggregates (Article, StockReservation, StockMovement). Event Storming
  surfaced them, but splitting now would scatter the stock invariants across aggregates with no
  transactional guard between them.
- **Clean-arch fit.** The aggregate root *is* the unit of consistency the use-case and persistence
  layers will load and save whole.

## 2. `SKU` and `Money` are value objects with no identity
- **Decision.** `SKU` and `Money` are immutable structs with unexported fields, compared **by value**
  (`Equals`), built only through factories (`NewSKU`, `NewMoney`) that **fail loud**.
- **Rule it protects.** "A valid SKU/Money by construction" — validity is guaranteed at the boundary,
  so use-sites never re-validate. `SKU` matches `^[A-Z0-9-]{3,32}$`; `Money` is `cents ≥ 0` + ISO-4217
  currency.
- **Rejected.** (a) Validation in the caller — the restitution explicitly flags "validation in the
  caller, not the factory". (b) Identity/IDs on SKU or Money — they have no lifecycle; two equal
  values are the same thing.
- **Clean-arch fit.** Value objects are the innermost layer; everything else trusts them.

## 3. `Money` is integer cents, never a float
- **Decision.** `Money` is `int64 amountCents` + `string currency`, never a floating-point amount.
- **Rule it protects.** Monetary exactness — no binary-floating rounding drift on prices/totals.
- **Rejected.** `float64` price (the restitution's "price as a floating-point number" anti-signal).
- **Clean-arch fit.** Fixes one of the worst-of antipatterns we found in CP1 (the monolith's bare
  `DECIMAL` with no currency): money now carries its currency and an exact representation.

## 4. Exactly one repository port; **no** `InventoryRepository`
- **Decision.** `interfaces.ArticleRepository` is the only port, with aggregate-level operations
  (`Save`, `FindByID`, `FindBySKU`, `List`, `Delete`). `Save` is documented as an idempotent upsert
  that does **not** publish events.
- **Rule it protects.** The aggregate boundary: inventory is persisted as part of its `Article`.
- **Rejected.** An `InventoryRepository` (or `SKURepository`/`MoneyRepository`). The restitution
  flags "an InventoryRepository (breaks the boundary)" directly.
- **Clean-arch fit.** Ports point inward to the domain; the MySQL adapter (CP4) implements this
  interface without the domain knowing about it.

## 5. Domain events are **recorded, not published**
- **Decision.** The aggregate appends events to a private `pendingEvents` list; `PullEvents()` drains
  them (returns and clears). Nothing in the domain publishes or serializes.
- **Rule it protects.** A pure, side-effect-free domain: an operation only mutates the aggregate and
  records a fact. Publication order/transactionality is an outer concern.
- **Rejected.** Publishing the instant an event happens (the "events published the moment they
  happen" anti-signal) — it couples the domain to a broker and breaks "Save then publish".
- **Clean-arch fit.** CP5 use cases will `Save` then drain `PullEvents()` and dispatch — the seam is
  already here.

## 6. IDs are injected; the `InventoryLevel` id is a deterministic composite
- **Decision.** `NewArticle(id, …)` takes the id as input (no generation in the domain). The
  `InventoryLevel` id is derived deterministically as `articleID + ":" + locationCode`.
- **Rule it protects.** A pure, deterministic domain — no `crypto/rand`, no hidden clock-driven IDs;
  tests are reproducible. "One level per (article, location)" falls out of the composite key.
- **Rejected.** Generating UUIDs inside the factory — that pulls a side effect (randomness) into the
  domain and pins ID strategy prematurely; UUID generation belongs to the CP5 use-case layer.
- **Clean-arch fit.** A persistence-assigned UUID can replace the composite id in CP4 without
  changing the domain API.

## 7. Creation vs reconstitution: `NewArticle` records, `RehydrateArticle` does not
- **Decision.** `NewArticle` is the factory for a *new* article and records `ArticleCreated`.
  `RehydrateArticle` rebuilds an article from persisted state and records **no** events.
- **Rule it protects.** Events are facts about things that *happened now*. Loading a row from the DB
  is not a new creation and must not re-emit `ArticleCreated`.
- **Rejected.** A single constructor used both for new and loaded articles — it would either re-emit
  creation events on every load or force the repository to strip events after the fact.
- **Clean-arch fit.** The CP4 repository will call `RehydrateArticle`; the use case calls
  `NewArticle`. The boundary is explicit.

## 8. Four events (incl. `ArticlePriceChanged`); time captured at record-time
- **Decision.** We ship `ArticleCreated`, `InventoryAdjusted`, `StockReserved` **and**
  `ArticlePriceChanged`. Each event captures `OccurredAt = time.Now().UTC()` in its constructor; the
  field is unexported (immutable), exposed via `OccurredAt()`.
- **Rule it protects.** Every state-changing operation records a fact. `ChangePrice` must record
  *something*; `ArticlePriceChanged` (with old + new cents) is that fact. The deck's "3 events" table
  is illustrative; the tactical-DDD mapping lists `ArticlePriceChanged` and the rules require a
  fact-recording `ChangePrice`.
- **Rejected.** (a) A `ChangePrice` that mutates silently with no event. (b) Injecting a `Clock`
  interface — premature DI for a training BC; tests assert `OccurredAt` is recent, not exact.
  (c) `StockReservationReleased` — there is no release operation in CP3 scope.
- **Clean-arch fit.** Event payloads are primitives only, so `events/` imports no domain types and
  stays a dependency leaf — ready for serialization (CP9) with no import cycle.

## 9. A health probe in `main.go`, kept out of the domain
- **Decision.** `main.go` boots an Echo server exposing only `GET /health → {"status":"ok"}` on
  `:8081`. It imports **no** domain package.
- **Rule it protects.** "Stop at the domain": no HTTP handlers for `/articles`, no persistence. The
  domain packages import no web framework and no DB driver.
- **Rejected.** Putting any business logic or repository wiring in `main.go` now (that is CP4–CP6).
- **Clean-arch fit.** `main` is the outermost layer; it may depend on inner layers but here it
  depends on none, so the domain stays runnable-but-isolated.

---

## Domain events — when each one fires

Events are **recorded only on success**: an operation that returns an error records **nothing**, and
`RehydrateArticle` records nothing at all (loading from storage is not a business fact). `OccurredAt`
is set to `time.Now().UTC()` inside the event constructor at the moment of recording. The aggregate
appends to a private `pendingEvents` slice; `PullEvents()` returns the events **in the order they
were recorded** and clears the list (drained once). The domain never publishes — an outer layer
(CP5) calls `Save`, then drains and dispatches.

| Event | Recorded by | Fires when (all preconditions pass) | Payload (+ `OccurredAt`) |
|---|---|---|---|
| `ArticleCreated` | `NewArticle(id, name, …, sku, price)` | a new Article is constructed — `id` & `name` non-empty, `price.AmountCents > 0` | `ArticleID, SKU, ArticleName, PriceCents, Currency` |
| `ArticlePriceChanged` | `Article.ChangePrice(newPrice)` | the price changes — `newPrice` has the **same currency** and `AmountCents > 0` | `ArticleID, OldPriceCents, NewPriceCents, Currency` |
| `InventoryAdjusted` | `Article.AdjustInventory(location, delta, reason)` | stock at a location changes — `location` non-empty; the level is created at 0 on first use; the resulting `quantity` stays `≥ 0` and `≥ reserved` | `ArticleID, LocationCode, Delta, NewQuantity, Reason` |
| `StockReserved` | `Article.ReserveStock(location, qty, reservationID, orderID)` | a reservation succeeds — the level **already exists**, `qty > 0`, and `reserved + qty ≤ quantity` | `ArticleID, LocationCode, Quantity, ReservationID, OrderID` |

**When NO event fires** (the call returns an error and state is left unchanged — each case is covered
by a test):
- `ChangePrice` → different currency, or `newPrice ≤ 0`.
- `AdjustInventory` → empty location, or a `delta` that would push `quantity` below `0` or below the
  units already `reserved`.
- `ReserveStock` → the location has no inventory level, `qty ≤ 0`, or `reserved + qty > quantity`.

**Not emitted in CP3:** `StockReservationReleased`. There is no release/cancel/confirm operation
until the reservation becomes its own entity (CP5+), so no code path records it yet.

Every event implements `DomainEvent` (`EventName()` + `OccurredAt()`). Payload fields are exported
primitives; `occurredAt` is unexported and set only by the constructor (immutable). So `events/`
imports no domain type, stays a dependency leaf, and is ready for on-the-wire serialization (CP9)
without an import cycle.

---

### How to verify
```bash
cd phase-03-skeleton                           # prerequisite: Go 1.22+ (no Docker, no DB)

go test ./...                                  # all tests (concise)
go test ./... -v                               # verbose: every invariant, one by one
go test ./entities/ -run TestReserveStock -v   # one rule in isolation (reserved <= quantity)
go test ./... -cover                           # coverage

go vet ./... && gofmt -l .                     # no warnings, no unformatted files

go run .                                       # boot the service, then in another shell:
curl -s localhost:8081/health                  # -> {"status":"ok"}
```
