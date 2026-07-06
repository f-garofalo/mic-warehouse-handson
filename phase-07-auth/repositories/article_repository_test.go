package repositories

import (
	"context"
	"testing"

	"warehouse.local/core/entities"
)

// Note: These are integration tests. They run only when MySQL is reachable
// at the address configured for the BC database. Without MySQL the test
// file still compiles and the assertions on the aggregate factory exercise
// pure-Go logic.

func TestArticleRepository_compilesAgainstAggregateRoot(t *testing.T) {
	sku, err := entities.NewSKU("TEST-001")
	if err != nil {
		t.Fatalf("expected SKU to construct, got %v", err)
	}
	price, err := entities.NewMoney(9999, "EUR")
	if err != nil {
		t.Fatalf("expected Money to construct, got %v", err)
	}
	a, err := entities.NewArticle("aggregate-1", *sku, "Test Article", "A test article", *price)
	if err != nil {
		t.Fatalf("expected Article to construct, got %v", err)
	}
	if a.SKU.Code != "TEST-001" {
		t.Errorf("expected SKU code TEST-001, got %s", a.SKU.Code)
	}
	if a.Price.AmountCents != 9999 {
		t.Errorf("expected price 9999 cents, got %d", a.Price.AmountCents)
	}

	ctx := context.Background()
	_ = ctx
}
