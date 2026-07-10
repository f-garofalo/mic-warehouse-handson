package events

import (
	"testing"
	"time"
)

func TestDataProductSchema_validatesRequiredFields(t *testing.T) {
	registry, err := LoadSchemaRegistry("../schemas")
	if err != nil {
		t.Fatalf("LoadSchemaRegistry: %v", err)
	}

	t0 := time.Date(2026, 5, 9, 12, 0, 0, 0, time.UTC)
	good := ArticleCreated{ArticleID: "a1", SKU: "ABC-001", ArticleName: "X", PriceCents: 100, Currency: "EUR", At: t0}
	if err := registry.Validate("warehouse.article.v1", toMap(good)); err != nil {
		t.Errorf("expected valid event, got %v", err)
	}

	missing := map[string]any{"article_id": "a1"} // missing required fields
	if err := registry.Validate("warehouse.article.v1", missing); err == nil {
		t.Error("expected validation error for missing required fields")
	}
}

func TestDataProductSchema_unknownIDIsError(t *testing.T) {
	registry, _ := LoadSchemaRegistry("../schemas")
	if err := registry.Validate("warehouse.unknown.v1", map[string]any{}); err == nil {
		t.Error("expected error for unknown schema id")
	}
}

func TestDataProductSchema_lifecycleStable(t *testing.T) {
	registry, _ := LoadSchemaRegistry("../schemas")
	stage, err := registry.Lifecycle("warehouse.article.v1")
	if err != nil {
		t.Fatalf("Lifecycle: %v", err)
	}
	if stage != "stable" {
		t.Errorf("expected stable, got %s", stage)
	}
}

// toMap converts any DomainEvent (here ArticleCreated) into a generic map for validation.
// Real impl might use json marshal+unmarshal; we keep it explicit for clarity.
func toMap(e ArticleCreated) map[string]any {
	return map[string]any{
		"article_id":  e.ArticleID,
		"sku":         e.SKU,
		"name":        e.ArticleName,
		"price_cents": e.PriceCents,
		"currency":    e.Currency,
		"updated_at":  e.At.Format(time.RFC3339Nano),
	}
}
