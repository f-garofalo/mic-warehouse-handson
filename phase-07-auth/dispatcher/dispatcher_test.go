package dispatcher

import (
	"context"
	"testing"
	"time"

	"warehouse.local/core/events"
)

func TestInMemoryDispatcher_recordsEventsInOrder(t *testing.T) {
	d := NewInMemoryDispatcher()

	t0 := time.Date(2026, 5, 8, 12, 0, 0, 0, time.UTC)
	es := []events.DomainEvent{
		events.ArticleCreated{ArticleID: "a", SKU: "X-1", ArticleName: "N", PriceCents: 100, Currency: "EUR", At: t0},
		events.InventoryAdjusted{ArticleID: "a", LocationCode: "L1", Delta: 5, NewQuantity: 5, Reason: "stock", At: t0.Add(1)},
	}
	if err := d.Dispatch(context.Background(), es); err != nil {
		t.Fatalf("Dispatch: %v", err)
	}

	got := d.Events()
	if len(got) != 2 {
		t.Fatalf("expected 2 events, got %d", len(got))
	}
	if got[0].Name() != "ArticleCreated" {
		t.Errorf("expected first event ArticleCreated, got %q", got[0].Name())
	}
	if got[1].Name() != "InventoryAdjusted" {
		t.Errorf("expected second event InventoryAdjusted, got %q", got[1].Name())
	}
}

func TestInMemoryDispatcher_failOnDispatch(t *testing.T) {
	d := NewInMemoryDispatcher()
	d.FailOnDispatch = errSimulated{}
	err := d.Dispatch(context.Background(), nil)
	if err == nil {
		t.Fatal("expected simulated dispatch failure, got nil")
	}
}

type errSimulated struct{}

func (errSimulated) Error() string { return "simulated dispatcher failure" }
