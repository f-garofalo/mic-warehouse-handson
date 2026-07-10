package events

import (
	"context"
	"strings"
	"testing"
	"time"
)

// ▸ Task 1 specification — the ArticleCreated tests ship red; the
// InventoryAdjusted test is green from the start: it exercises the worked
// example you pattern-match.

func TestHermesPublisher_inventoryAdjustedIsTheWorkedExample(t *testing.T) {
	pub := newTestPublisher(t)

	t0 := time.Date(2026, 5, 9, 12, 0, 0, 0, time.UTC)
	err := pub.Dispatch(context.Background(), []DomainEvent{
		InventoryAdjusted{ArticleID: "a1", LocationCode: "MAIN", Delta: -5, NewQuantity: 45, Reason: "damaged goods", At: t0},
	})
	if err != nil {
		t.Fatalf("dispatch worked example: %v", err)
	}

	envelopes := pub.PublishedCloudEvents()
	if len(envelopes) != 1 {
		t.Fatalf("expected 1 CloudEvents envelope, got %d", len(envelopes))
	}
	if envelopes[0].Type != "warehouse.inventory.adjusted.v1" {
		t.Errorf("expected type warehouse.inventory.adjusted.v1, got %q", envelopes[0].Type)
	}
	if envelopes[0].Subject != "articles/a1/inventory/MAIN" {
		t.Errorf("expected subject articles/a1/inventory/MAIN, got %q", envelopes[0].Subject)
	}
}

func TestHermesPublisher_articleCreatedBecomesAValidRecord(t *testing.T) {
	pub := newTestPublisher(t)

	t0 := time.Date(2026, 5, 9, 12, 0, 0, 0, time.UTC)
	err := pub.Dispatch(context.Background(), []DomainEvent{
		ArticleCreated{ArticleID: "a1", SKU: "ABC-001", ArticleName: "Widget", PriceCents: 1299, Currency: "EUR", At: t0},
	})
	if err != nil {
		t.Fatalf("dispatch ArticleCreated: %v", err)
	}

	envelopes := pub.PublishedCloudEvents()
	if len(envelopes) != 1 {
		t.Fatalf("expected 1 CloudEvents envelope, got %d", len(envelopes))
	}
	env := envelopes[0]

	// The envelope: technical metadata, the consumer's routing information.
	if env.SpecVersion != "1.0" {
		t.Errorf("expected specversion 1.0, got %q", env.SpecVersion)
	}
	if env.Type != "warehouse.article.v1" {
		t.Errorf("expected type warehouse.article.v1, got %q", env.Type)
	}
	if env.Source != "warehouse-bc" {
		t.Errorf("expected source warehouse-bc, got %q", env.Source)
	}
	if env.Subject != "articles/a1" {
		t.Errorf("expected subject articles/a1, got %q", env.Subject)
	}
	if !strings.HasPrefix(env.Time, "2026-05-09T12:00:00") {
		t.Errorf("expected the event time in the envelope, got %q", env.Time)
	}

	// The payload: the Data Product, exactly as the schema declares it.
	if env.Data["article_id"] != "a1" {
		t.Errorf("expected data.article_id a1, got %v", env.Data["article_id"])
	}
	if env.Data["sku"] != "ABC-001" {
		t.Errorf("expected data.sku ABC-001, got %v", env.Data["sku"])
	}
	if env.Data["name"] != "Widget" {
		t.Errorf(`expected data.name "Widget" (the contract says name, the event says ArticleName), got %v`, env.Data["name"])
	}
	if env.Data["price_cents"] != int64(1299) {
		t.Errorf("expected data.price_cents 1299, got %v", env.Data["price_cents"])
	}
	if env.Data["currency"] != "EUR" {
		t.Errorf("expected data.currency EUR, got %v", env.Data["currency"])
	}
	if _, ok := env.Data["updated_at"]; !ok {
		t.Error("expected data.updated_at (required by the schema)")
	}
}

func TestHermesPublisher_rejectsArticleCreatedWithEmptySKU(t *testing.T) {
	pub := newTestPublisher(t)

	// The schema declares sku as required and the registry enforces it: a
	// broken payload must never become a published record.
	bad := []DomainEvent{
		ArticleCreated{ArticleID: "a1", ArticleName: "X", PriceCents: 100, Currency: "EUR", At: time.Now().UTC()},
	}
	if err := pub.Dispatch(context.Background(), bad); err == nil {
		t.Fatal("expected a validation error for a missing SKU")
	}
	if got := pub.PublishedCloudEvents(); len(got) != 0 {
		t.Errorf("a rejected event must not be published, got %d records", len(got))
	}
}

func TestHermesPublisher_rejectsEventWithoutSchema(t *testing.T) {
	pub := newTestPublisher(t)

	// StockReserved has no Data Product schema in this exercise: the
	// publisher refuses it. Nothing leaves the BC without a contract.
	bad := []DomainEvent{
		StockReserved{ArticleID: "a1", LocationCode: "L1", Quantity: 5, ReservationID: "r1", OrderID: "o1", At: time.Now().UTC()},
	}
	if err := pub.Dispatch(context.Background(), bad); err == nil {
		t.Fatal("expected dispatch error: no schema for StockReserved")
	}
}

func TestValidateCloudEvent_rejectsMissingRequiredAttribute(t *testing.T) {
	err := validateCloudEvent(CloudEvent{
		SpecVersion:     "1.0",
		ID:              "id-1",
		Source:          "warehouse-bc",
		Type:            "",
		Time:            time.Now().UTC().Format(time.RFC3339Nano),
		DataContentType: "application/json",
		Data:            map[string]any{"article_id": "a1"},
	})
	if err == nil {
		t.Fatal("expected missing type error")
	}
}

func newTestPublisher(t *testing.T) *HermesPublisher {
	t.Helper()
	registry, err := LoadSchemaRegistry("../schemas")
	if err != nil {
		t.Fatalf("registry: %v", err)
	}
	return NewHermesPublisher(registry)
}
