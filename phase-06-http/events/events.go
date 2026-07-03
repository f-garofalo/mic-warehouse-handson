package events

import "time"

// DomainEvent is the contract every event in the Warehouse BC implements.
// Per ADR-011: events are past-tense facts emitted by an aggregate; immutable;
// distributable to other BCs (ADR-014 governs serialization for Hermes).
type DomainEvent interface {
	Name() string
	OccurredAt() time.Time
}

// ArticleCreated records that a new Article was published.
// ArticleName carries the display name; the field is named ArticleName (not Name)
// to avoid collision with the DomainEvent.Name() method.
type ArticleCreated struct {
	ArticleID   string
	SKU         string
	ArticleName string
	PriceCents  int64
	Currency    string
	At          time.Time
}

func (e ArticleCreated) Name() string          { return "ArticleCreated" }
func (e ArticleCreated) OccurredAt() time.Time { return e.At }

// InventoryAdjusted records a positive or negative change in stock at a location.
// Delta carries the sign; NewQuantity is the level after the change.
type InventoryAdjusted struct {
	ArticleID    string
	LocationCode string
	Delta        int32
	NewQuantity  int32
	Reason       string
	At           time.Time
}

func (e InventoryAdjusted) Name() string          { return "InventoryAdjusted" }
func (e InventoryAdjusted) OccurredAt() time.Time { return e.At }

// StockReserved records that stock has been allocated to an order pending fulfilment.
type StockReserved struct {
	ArticleID     string
	LocationCode  string
	Quantity      int32
	ReservationID string
	OrderID       string
	At            time.Time
}

func (e StockReserved) Name() string          { return "StockReserved" }
func (e StockReserved) OccurredAt() time.Time { return e.At }

// ArticlePriceChanged records that an Article's price was updated.
// Currency does not change via this event; a separate currency-migration event
// will be introduced when needed.
type ArticlePriceChanged struct {
	ArticleID     string
	OldPriceCents int64
	NewPriceCents int64
	Currency      string
	At            time.Time
}

func (e ArticlePriceChanged) Name() string          { return "ArticlePriceChanged" }
func (e ArticlePriceChanged) OccurredAt() time.Time { return e.At }
