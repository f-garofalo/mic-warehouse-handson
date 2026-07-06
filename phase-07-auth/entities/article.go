package entities

import (
	"errors"
	"strings"
	"time"
)

// Article is the aggregate root for the Warehouse domain.
// Per ADR-011: callers only ever talk to the root. Inventory levels and stock
// reservations are owned by this aggregate and never modified independently.
//
// Invariants enforced on construction and on every mutation:
//   - ID may be empty at construction: the id is minted by the system of
//     record at first Save (during the strangler migration that is the MIC
//     monolith's business_data table)
//   - Name is non-empty after trim
//   - Price.AmountCents > 0
//   - Price currency does not change after construction (use a separate
//     migration flow for that, not ChangePrice).
type Article struct {
	ID          string           `json:"id" db:"id"`
	SKU         SKU              `json:"sku" db:"sku"`
	Name        string           `json:"name" db:"name"`
	Description string           `json:"description" db:"description"`
	Price       Money            `json:"price" db:"price"`
	Inventories []InventoryLevel `json:"inventories,omitempty" db:"-"`
	CreatedAt   time.Time        `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time        `json:"updated_at" db:"updated_at"`

	// pendingEvents is an unexported slice of facts the aggregate has recorded
	// since it was loaded. The use case (or repository) drains these after
	// saving. See ADR-011: aggregates record events; they do not publish them.
	pendingEvents []PendingEvent
}

// PendingEvent is the minimal interface for an event recorded by an aggregate.
// The events package (ADR-011) provides concrete implementations.
type PendingEvent interface {
	Name() string
	OccurredAt() time.Time
}

// NewArticle constructs a fresh Article aggregate, enforcing invariants.
//
// An empty id is ACCEPTED: the id is minted by the system of record at first
// Save (the repository assigns it). Every other invariant is unchanged.
func NewArticle(id string, sku SKU, name, description string, price Money) (*Article, error) {
	if strings.TrimSpace(name) == "" {
		return nil, errors.New("article: name is required")
	}
	if price.AmountCents <= 0 {
		return nil, errors.New("article: price must be > 0")
	}
	now := time.Now().UTC()
	return &Article{
		ID:          id,
		SKU:         sku,
		Name:        name,
		Description: description,
		Price:       price,
		CreatedAt:   now,
		UpdatedAt:   now,
	}, nil
}

// ChangePrice mutates the aggregate's price, recording an event.
// Currency cannot change via this method; the caller must use a different flow
// for currency migrations.
func (a *Article) ChangePrice(newPrice Money) error {
	if newPrice.AmountCents <= 0 {
		return errors.New("article: new price must be > 0")
	}
	if newPrice.Currency != a.Price.Currency {
		return errors.New("article: currency change requires explicit migration; not allowed via ChangePrice")
	}
	if newPrice.Equals(a.Price) {
		return nil // no-op, no event
	}
	old := a.Price
	a.Price = newPrice
	a.UpdatedAt = time.Now().UTC()
	a.recordEvent(articlePriceChangedEvent{
		articleID:     a.ID,
		oldPriceCents: old.AmountCents,
		newPriceCents: newPrice.AmountCents,
		currency:      newPrice.Currency,
		occurredAt:    a.UpdatedAt,
	})
	return nil
}

// PendingEvents returns the events recorded by this aggregate since load.
// Callers (use case, repository) drain this after persistence.
func (a *Article) PendingEvents() []PendingEvent {
	return append([]PendingEvent(nil), a.pendingEvents...)
}

// ClearPendingEvents removes all recorded events. Called after publication.
func (a *Article) ClearPendingEvents() {
	a.pendingEvents = nil
}

func (a *Article) recordEvent(e PendingEvent) {
	a.pendingEvents = append(a.pendingEvents, e)
}

// articlePriceChangedEvent is a private aggregate-recorded event used to
// satisfy TestArticle_ChangePrice_recordsEvent. The richer canonical
// ArticlePriceChanged type lives in the events package and is wired by the
// use case layer in Plan 2B-iii.
type articlePriceChangedEvent struct {
	articleID     string
	oldPriceCents int64
	newPriceCents int64
	currency      string
	occurredAt    time.Time
}

func (e articlePriceChangedEvent) Name() string          { return "ArticlePriceChanged" }
func (e articlePriceChangedEvent) OccurredAt() time.Time { return e.occurredAt }

// AdjustInventory mutates an InventoryLevel for the given locationCode by delta.
// If no level exists for the location, one is created with the delta as its
// quantity (provided delta is non-negative).
//
// Invariants:
//   - locationCode is non-empty
//   - resulting quantity must be >= 0
//
// On success, an InventoryAdjusted event is recorded. A zero delta is a no-op.
func (a *Article) AdjustInventory(locationCode string, delta int32, reason string) error {
	if strings.TrimSpace(locationCode) == "" {
		return errors.New("article: locationCode is required")
	}
	if delta == 0 {
		return nil
	}

	now := time.Now().UTC()

	// Find existing level for the location.
	idx := -1
	for i := range a.Inventories {
		if a.Inventories[i].LocationCode == locationCode {
			idx = i
			break
		}
	}

	var newQty int32
	if idx >= 0 {
		newQty = a.Inventories[idx].Quantity + delta
		if newQty < 0 {
			return errors.New("article: inventory adjustment would make quantity negative")
		}
		a.Inventories[idx].Quantity = newQty
		a.Inventories[idx].UpdatedAt = now
	} else {
		if delta < 0 {
			return errors.New("article: cannot adjust to negative quantity on a fresh location")
		}
		newQty = delta
		a.Inventories = append(a.Inventories, InventoryLevel{
			ID:           a.ID + ":" + locationCode, // deterministic for the exercise
			ArticleID:    a.ID,
			LocationCode: locationCode,
			Quantity:     newQty,
			Reserved:     0,
			CreatedAt:    now,
			UpdatedAt:    now,
		})
	}

	a.UpdatedAt = now
	a.recordEvent(inventoryAdjustedEvent{
		articleID:    a.ID,
		locationCode: locationCode,
		delta:        delta,
		newQuantity:  newQty,
		reason:       reason,
		occurredAt:   now,
	})
	return nil
}

// inventoryAdjustedEvent is a private aggregate-recorded event satisfying the
// PendingEvent interface. The use case layer (Task 7) translates it into the
// canonical events.InventoryAdjusted before publication. Task 6 will collapse
// this and articlePriceChangedEvent into direct uses of the events package.
type inventoryAdjustedEvent struct {
	articleID    string
	locationCode string
	delta        int32
	newQuantity  int32
	reason       string
	occurredAt   time.Time
}

func (e inventoryAdjustedEvent) Name() string          { return "InventoryAdjusted" }
func (e inventoryAdjustedEvent) OccurredAt() time.Time { return e.occurredAt }
