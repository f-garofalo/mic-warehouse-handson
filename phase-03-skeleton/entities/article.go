package entities

import (
	"fmt"

	"warehouse.local/core/events"
)

// Article is the aggregate root of the Warehouse bounded context. Callers only
// ever talk to Article: SKU and Money are its value objects, and InventoryLevel
// entities are reached only through it. Identity is by ID.
//
// Invariants enforced by the aggregate: price > 0, currency stable after
// creation, name non-empty, and (via InventoryLevel) quantity >= 0,
// reserved >= 0, reserved <= quantity, one level per location.
type Article struct {
	id            string
	name          string
	description   string
	sku           SKU
	price         Money
	levels        []*InventoryLevel
	pendingEvents []events.DomainEvent
}

// NewArticle is the factory for a brand-new Article. It fails loud on an empty
// id or name and on a non-positive price (the aggregate is stricter than Money,
// which allows zero). It records an ArticleCreated event.
func NewArticle(id, name, description string, sku SKU, price Money) (*Article, error) {
	if id == "" {
		return nil, fmt.Errorf("article id must not be empty")
	}
	if name == "" {
		return nil, fmt.Errorf("article name must not be empty")
	}
	if price.AmountCents() <= 0 {
		return nil, fmt.Errorf("article price must be > 0, got %d", price.AmountCents())
	}
	a := &Article{id: id, name: name, description: description, sku: sku, price: price}
	a.record(events.NewArticleCreated(a.id, a.sku.Code(), a.name, a.price.AmountCents(), a.price.Currency()))
	return a, nil
}

// RehydrateArticle reconstitutes an Article from persisted state (used by the
// repository in CP4). It records NO events: loading an aggregate is not a
// business fact and must not re-emit ArticleCreated.
func RehydrateArticle(id, name, description string, sku SKU, price Money, levels []*InventoryLevel) *Article {
	return &Article{id: id, name: name, description: description, sku: sku, price: price, levels: levels}
}

// ChangePrice replaces the price, keeping the currency. A currency change is a
// separate migration flow, not this method. Records ArticlePriceChanged.
func (a *Article) ChangePrice(newPrice Money) error {
	if newPrice.Currency() != a.price.Currency() {
		return fmt.Errorf("cannot change currency from %s to %s here (separate migration flow)",
			a.price.Currency(), newPrice.Currency())
	}
	if newPrice.AmountCents() <= 0 {
		return fmt.Errorf("article price must be > 0, got %d", newPrice.AmountCents())
	}
	old := a.price.AmountCents()
	a.price = newPrice
	a.record(events.NewArticlePriceChanged(a.id, old, newPrice.AmountCents(), newPrice.Currency()))
	return nil
}

// AdjustInventory changes stock at a location by delta, creating the level on
// first use (one level per location). Records InventoryAdjusted on success; on
// a rejected adjustment nothing is recorded.
func (a *Article) AdjustInventory(locationCode string, delta int64, reason string) error {
	if locationCode == "" {
		return fmt.Errorf("locationCode must not be empty")
	}
	level := a.levelAt(locationCode)
	if level == nil {
		l, err := newInventoryLevel(a.inventoryLevelID(locationCode), locationCode, 0, 0)
		if err != nil {
			return err
		}
		level = l
		a.levels = append(a.levels, level)
	}
	newQty, err := level.adjust(delta)
	if err != nil {
		return err
	}
	a.record(events.NewInventoryAdjusted(a.id, locationCode, delta, newQty, reason))
	return nil
}

// ReserveStock reserves qty units at an existing location. Reserving where no
// stock exists fails loud. Records StockReserved on success.
func (a *Article) ReserveStock(locationCode string, qty int64, reservationID, orderID string) error {
	if locationCode == "" {
		return fmt.Errorf("locationCode must not be empty")
	}
	level := a.levelAt(locationCode)
	if level == nil {
		return fmt.Errorf("cannot reserve at location %q: no inventory there", locationCode)
	}
	if err := level.reserve(qty); err != nil {
		return err
	}
	a.record(events.NewStockReserved(a.id, locationCode, qty, reservationID, orderID))
	return nil
}

// PullEvents returns the recorded events and clears the pending list (drained
// once). An outer layer (CP5) publishes them after Save succeeds.
func (a *Article) PullEvents() []events.DomainEvent {
	out := a.pendingEvents
	a.pendingEvents = nil // hand the backing array to the caller, drop our reference
	return out
}

func (a *Article) ID() string          { return a.id }
func (a *Article) Name() string        { return a.name }
func (a *Article) Description() string { return a.description }
func (a *Article) SKU() SKU            { return a.sku }
func (a *Article) Price() Money        { return a.price }

// Levels returns a copy of the slice header so callers cannot append to or
// reorder the aggregate's own inventory. The InventoryLevel mutators are
// unexported, so the pointed-to entities cannot be mutated from outside.
func (a *Article) Levels() []*InventoryLevel {
	out := make([]*InventoryLevel, len(a.levels))
	copy(out, a.levels)
	return out
}

func (a *Article) record(e events.DomainEvent) { a.pendingEvents = append(a.pendingEvents, e) }

func (a *Article) levelAt(locationCode string) *InventoryLevel {
	for _, l := range a.levels {
		if l.locationCode == locationCode {
			return l
		}
	}
	return nil
}

// inventoryLevelID derives a deterministic id for the (article, location) pair.
// A persistence-assigned UUID can replace this in CP4 without changing the API.
func (a *Article) inventoryLevelID(locationCode string) string {
	return a.id + ":" + locationCode
}
