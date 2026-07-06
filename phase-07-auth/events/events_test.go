package events

import (
	"testing"
	"time"
)

func TestArticleCreated_isDomainEvent(t *testing.T) {
	t0 := time.Date(2026, 5, 8, 12, 0, 0, 0, time.UTC)
	e := ArticleCreated{
		ArticleID:   "a1",
		SKU:         "ABC-001",
		ArticleName: "Widget",
		PriceCents:  2999,
		Currency:    "EUR",
		At:          t0,
	}

	var de DomainEvent = e
	if de.Name() != "ArticleCreated" {
		t.Errorf("expected name ArticleCreated, got %q", de.Name())
	}
	if !de.OccurredAt().Equal(t0) {
		t.Errorf("expected OccurredAt=%v, got %v", t0, de.OccurredAt())
	}
	if e.ArticleName != "Widget" {
		t.Errorf("expected ArticleName=Widget, got %q", e.ArticleName)
	}
}

func TestInventoryAdjusted_isDomainEvent(t *testing.T) {
	t0 := time.Date(2026, 5, 8, 12, 0, 0, 0, time.UTC)
	e := InventoryAdjusted{
		ArticleID:    "a1",
		LocationCode: "IT-MILANO1",
		Delta:        +5,
		NewQuantity:  15,
		Reason:       "manual restock",
		At:           t0,
	}

	var de DomainEvent = e
	if de.Name() != "InventoryAdjusted" {
		t.Errorf("expected name InventoryAdjusted, got %q", de.Name())
	}
	if e.Delta != 5 {
		t.Errorf("expected delta=5, got %d", e.Delta)
	}
}

func TestStockReserved_isDomainEvent(t *testing.T) {
	t0 := time.Date(2026, 5, 8, 12, 0, 0, 0, time.UTC)
	e := StockReserved{
		ArticleID:     "a1",
		LocationCode:  "IT-MILANO1",
		Quantity:      3,
		ReservationID: "r1",
		OrderID:       "o1",
		At:            t0,
	}

	var de DomainEvent = e
	if de.Name() != "StockReserved" {
		t.Errorf("expected name StockReserved, got %q", de.Name())
	}
}

func TestArticlePriceChanged_isDomainEvent(t *testing.T) {
	t0 := time.Date(2026, 5, 8, 12, 0, 0, 0, time.UTC)
	e := ArticlePriceChanged{
		ArticleID:     "a1",
		OldPriceCents: 1000,
		NewPriceCents: 1500,
		Currency:      "EUR",
		At:            t0,
	}
	var de DomainEvent = e
	if de.Name() != "ArticlePriceChanged" {
		t.Errorf("expected name ArticlePriceChanged, got %q", de.Name())
	}
}
