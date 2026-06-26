// Package events holds the Warehouse domain events. They are past-tense,
// immutable facts. An aggregate RECORDS them on its pending list; an outer
// layer (CP5) drains and publishes them. Nothing here publishes or serializes.
//
// Payloads are primitives only (strings, ints, time) — the package imports no
// domain types, so it stays a dependency leaf and is ready for serialization
// later (CP9) without an import cycle.
package events

import "time"

// DomainEvent is the common interface for every recorded fact.
type DomainEvent interface {
	EventName() string
	OccurredAt() time.Time
}

// ArticleCreated is recorded when a new Article is published.
type ArticleCreated struct {
	ArticleID   string
	SKU         string
	ArticleName string
	PriceCents  int64
	Currency    string
	occurredAt  time.Time
}

func NewArticleCreated(articleID, sku, articleName string, priceCents int64, currency string) ArticleCreated {
	return ArticleCreated{
		ArticleID: articleID, SKU: sku, ArticleName: articleName,
		PriceCents: priceCents, Currency: currency, occurredAt: time.Now().UTC(),
	}
}

func (e ArticleCreated) EventName() string     { return "ArticleCreated" }
func (e ArticleCreated) OccurredAt() time.Time { return e.occurredAt }

// ArticlePriceChanged is recorded when an Article's price changes (currency
// unchanged).
type ArticlePriceChanged struct {
	ArticleID     string
	OldPriceCents int64
	NewPriceCents int64
	Currency      string
	occurredAt    time.Time
}

func NewArticlePriceChanged(articleID string, oldPriceCents, newPriceCents int64, currency string) ArticlePriceChanged {
	return ArticlePriceChanged{
		ArticleID: articleID, OldPriceCents: oldPriceCents, NewPriceCents: newPriceCents,
		Currency: currency, occurredAt: time.Now().UTC(),
	}
}

func (e ArticlePriceChanged) EventName() string     { return "ArticlePriceChanged" }
func (e ArticlePriceChanged) OccurredAt() time.Time { return e.occurredAt }

// InventoryAdjusted is recorded when an InventoryLevel quantity changes.
type InventoryAdjusted struct {
	ArticleID    string
	LocationCode string
	Delta        int64
	NewQuantity  int64
	Reason       string
	occurredAt   time.Time
}

func NewInventoryAdjusted(articleID, locationCode string, delta, newQuantity int64, reason string) InventoryAdjusted {
	return InventoryAdjusted{
		ArticleID: articleID, LocationCode: locationCode, Delta: delta,
		NewQuantity: newQuantity, Reason: reason, occurredAt: time.Now().UTC(),
	}
}

func (e InventoryAdjusted) EventName() string     { return "InventoryAdjusted" }
func (e InventoryAdjusted) OccurredAt() time.Time { return e.occurredAt }

// StockReserved is recorded when stock is reserved against an InventoryLevel.
type StockReserved struct {
	ArticleID     string
	LocationCode  string
	Quantity      int64
	ReservationID string
	OrderID       string
	occurredAt    time.Time
}

func NewStockReserved(articleID, locationCode string, quantity int64, reservationID, orderID string) StockReserved {
	return StockReserved{
		ArticleID: articleID, LocationCode: locationCode, Quantity: quantity,
		ReservationID: reservationID, OrderID: orderID, occurredAt: time.Now().UTC(),
	}
}

func (e StockReserved) EventName() string     { return "StockReserved" }
func (e StockReserved) OccurredAt() time.Time { return e.occurredAt }
