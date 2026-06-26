package entities

import (
	"testing"

	"warehouse.local/core/events"
)

func mustSKU(t *testing.T, code string) SKU {
	t.Helper()
	s, err := NewSKU(code)
	if err != nil {
		t.Fatalf("NewSKU(%q): %v", code, err)
	}
	return s
}

func mustMoney(t *testing.T, cents int64, cur string) Money {
	t.Helper()
	m, err := NewMoney(cents, cur)
	if err != nil {
		t.Fatalf("NewMoney(%d,%q): %v", cents, cur, err)
	}
	return m
}

func mustArticle(t *testing.T) *Article {
	t.Helper()
	a, err := NewArticle("art-1", "Widget", "a widget", mustSKU(t, "WID-001"), mustMoney(t, 1000, "EUR"))
	if err != nil {
		t.Fatalf("NewArticle: %v", err)
	}
	return a
}

func TestNewArticleInvariants(t *testing.T) {
	sku := mustSKU(t, "WID-001")
	price := mustMoney(t, 1000, "EUR")

	if _, err := NewArticle("", "Widget", "", sku, price); err == nil {
		t.Error("empty id must fail")
	}
	if _, err := NewArticle("art-1", "", "", sku, price); err == nil {
		t.Error("empty name must fail")
	}
	// Money(0,"EUR") is valid, but the aggregate refuses a zero price.
	zero := mustMoney(t, 0, "EUR")
	if _, err := NewArticle("art-1", "Widget", "", sku, zero); err == nil {
		t.Error("zero price must fail on the aggregate even though Money(0,EUR) is valid")
	}
}

func TestNewArticleRecordsArticleCreated(t *testing.T) {
	a := mustArticle(t)
	evs := a.PullEvents()
	if len(evs) != 1 {
		t.Fatalf("expected 1 event, got %d", len(evs))
	}
	ac, ok := evs[0].(events.ArticleCreated)
	if !ok {
		t.Fatalf("expected ArticleCreated, got %T", evs[0])
	}
	if ac.PriceCents != 1000 || ac.Currency != "EUR" || ac.SKU != "WID-001" {
		t.Errorf("ArticleCreated payload wrong: %#v", ac)
	}
	if ac.OccurredAt().IsZero() {
		t.Error("OccurredAt must be set")
	}
}

func TestChangePrice(t *testing.T) {
	a := mustArticle(t)
	a.PullEvents() // drain ArticleCreated

	if err := a.ChangePrice(mustMoney(t, 1200, "USD")); err == nil {
		t.Error("currency change must be rejected")
	}
	if err := a.ChangePrice(mustMoney(t, 0, "EUR")); err == nil {
		t.Error("zero price must be rejected")
	}
	if err := a.ChangePrice(mustMoney(t, 1500, "EUR")); err != nil {
		t.Fatalf("valid ChangePrice failed: %v", err)
	}
	if a.Price().AmountCents() != 1500 {
		t.Errorf("price not updated, got %d", a.Price().AmountCents())
	}
	evs := a.PullEvents()
	if len(evs) != 1 {
		t.Fatalf("expected 1 event after the one valid change, got %d", len(evs))
	}
	pc, ok := evs[0].(events.ArticlePriceChanged)
	if !ok {
		t.Fatalf("expected ArticlePriceChanged, got %T", evs[0])
	}
	if pc.OldPriceCents != 1000 || pc.NewPriceCents != 1500 {
		t.Errorf("old/new wrong: %d -> %d", pc.OldPriceCents, pc.NewPriceCents)
	}
}

func TestAdjustInventory(t *testing.T) {
	a := mustArticle(t)
	a.PullEvents()

	if err := a.AdjustInventory("", 5, "init"); err == nil {
		t.Error("empty location must fail")
	}
	if err := a.AdjustInventory("IT-MIL", 10, "inbound"); err != nil {
		t.Fatalf("first adjust failed: %v", err)
	}
	if len(a.Levels()) != 1 {
		t.Fatalf("expected 1 level, got %d", len(a.Levels()))
	}
	// same location reuses the same level (one per location)
	if err := a.AdjustInventory("IT-MIL", 5, "inbound"); err != nil {
		t.Fatalf("second adjust failed: %v", err)
	}
	if len(a.Levels()) != 1 {
		t.Fatalf("still expected 1 level, got %d", len(a.Levels()))
	}
	if got := a.Levels()[0].Quantity(); got != 15 {
		t.Errorf("expected quantity 15, got %d", got)
	}
	// cannot go negative
	if err := a.AdjustInventory("IT-MIL", -100, "shrinkage"); err == nil {
		t.Error("adjust below zero must fail")
	}
	evs := a.PullEvents()
	if len(evs) != 2 {
		t.Fatalf("expected 2 InventoryAdjusted (failed ones record nothing), got %d", len(evs))
	}
	ia, ok := evs[1].(events.InventoryAdjusted)
	if !ok || ia.NewQuantity != 15 || ia.Delta != 5 {
		t.Errorf("second event wrong: %#v", evs[1])
	}
}

func TestAdjustInventoryBelowReservedRejected(t *testing.T) {
	a := mustArticle(t)
	if err := a.AdjustInventory("IT-MIL", 10, "inbound"); err != nil {
		t.Fatal(err)
	}
	if err := a.ReserveStock("IT-MIL", 8, "res-1", "ord-1"); err != nil {
		t.Fatal(err)
	}
	// quantity 10, reserved 8 -> adjust -5 would leave quantity 5 < reserved 8
	if err := a.AdjustInventory("IT-MIL", -5, "shrinkage"); err == nil {
		t.Error("adjust dropping quantity below reserved must fail")
	}
	if got := a.Levels()[0].Quantity(); got != 10 {
		t.Errorf("quantity must be unchanged at 10, got %d", got)
	}
}

func TestReserveStock(t *testing.T) {
	a := mustArticle(t)
	a.PullEvents()

	// reserve at unknown location fails loud
	if err := a.ReserveStock("IT-MIL", 1, "res-0", "ord-0"); err == nil {
		t.Error("reserve at unknown location must fail")
	}
	if err := a.AdjustInventory("IT-MIL", 10, "inbound"); err != nil {
		t.Fatal(err)
	}
	a.PullEvents()

	if err := a.ReserveStock("IT-MIL", 0, "res-x", "ord-x"); err == nil {
		t.Error("reserve qty 0 must fail")
	}
	if err := a.ReserveStock("IT-MIL", 11, "res-x", "ord-x"); err == nil {
		t.Error("reserve beyond quantity must fail")
	}
	if err := a.ReserveStock("IT-MIL", 6, "res-1", "ord-1"); err != nil {
		t.Fatalf("valid reserve failed: %v", err)
	}
	if a.Levels()[0].Reserved() != 6 || a.Levels()[0].Available() != 4 {
		t.Errorf("reserved/available wrong: %d/%d", a.Levels()[0].Reserved(), a.Levels()[0].Available())
	}
	// reserving the remaining 4 is fine; one more must fail (reserved <= quantity)
	if err := a.ReserveStock("IT-MIL", 4, "res-2", "ord-2"); err != nil {
		t.Fatalf("reserve remaining failed: %v", err)
	}
	if err := a.ReserveStock("IT-MIL", 1, "res-3", "ord-3"); err == nil {
		t.Error("reserve beyond quantity must fail (reserved <= quantity)")
	}
	evs := a.PullEvents()
	if len(evs) != 2 {
		t.Fatalf("expected 2 StockReserved, got %d", len(evs))
	}
	if _, ok := evs[0].(events.StockReserved); !ok {
		t.Fatalf("expected StockReserved, got %T", evs[0])
	}
}

func TestEventsRecordedInOrderAndDrainedOnce(t *testing.T) {
	a := mustArticle(t) // ArticleCreated
	if err := a.AdjustInventory("IT-MIL", 10, "in"); err != nil {
		t.Fatal(err)
	}
	if err := a.ReserveStock("IT-MIL", 3, "r", "o"); err != nil {
		t.Fatal(err)
	}
	evs := a.PullEvents()
	if len(evs) != 3 {
		t.Fatalf("expected 3 events, got %d", len(evs))
	}
	want := []string{"ArticleCreated", "InventoryAdjusted", "StockReserved"}
	for i, w := range want {
		if evs[i].EventName() != w {
			t.Errorf("event %d = %s, want %s", i, evs[i].EventName(), w)
		}
	}
	if again := a.PullEvents(); len(again) != 0 {
		t.Errorf("events must drain once; second PullEvents got %d", len(again))
	}
}

func TestRehydrateRecordsNoEvents(t *testing.T) {
	lvl, err := newInventoryLevel("art-1:IT-MIL", "IT-MIL", 10, 2)
	if err != nil {
		t.Fatal(err)
	}
	a := RehydrateArticle("art-1", "Widget", "", mustSKU(t, "WID-001"), mustMoney(t, 1000, "EUR"), []*InventoryLevel{lvl})
	if evs := a.PullEvents(); len(evs) != 0 {
		t.Errorf("rehydrate must record no events, got %d", len(evs))
	}
	if got := a.Levels()[0].Available(); got != 8 {
		t.Errorf("expected available 8, got %d", got)
	}
}
