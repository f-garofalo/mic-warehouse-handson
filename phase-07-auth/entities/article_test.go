package entities

import "testing"

func TestNewArticle_validInputs(t *testing.T) {
	sku, _ := NewSKU("ABC-001")
	price, _ := NewMoney(2999, "EUR")

	a, err := NewArticle("article-1", *sku, "Widget", "A useful widget", *price)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if a.ID != "article-1" {
		t.Errorf("expected ID article-1, got %q", a.ID)
	}
	if !a.SKU.Equals(*sku) {
		t.Errorf("expected SKU ABC-001, got %v", a.SKU)
	}
	if !a.Price.Equals(*price) {
		t.Errorf("expected Price 2999 EUR, got %v", a.Price)
	}
	if a.Name != "Widget" {
		t.Errorf("expected Name Widget, got %q", a.Name)
	}
}

func TestNewArticle_allowsEmptyID(t *testing.T) {
	// An empty id is legal at construction: the id is minted by the system of
	// record at first Save. The aggregate carries the empty id until then.
	sku, _ := NewSKU("ABC-001")
	price, _ := NewMoney(100, "EUR")
	a, err := NewArticle("", *sku, "X", "", *price)
	if err != nil {
		t.Fatalf("expected empty ID to be accepted at construction, got %v", err)
	}
	if a.ID != "" {
		t.Errorf("expected ID to stay empty until Save mints it, got %q", a.ID)
	}
}

func TestNewArticle_rejectsEmptyName(t *testing.T) {
	sku, _ := NewSKU("ABC-001")
	price, _ := NewMoney(100, "EUR")
	if _, err := NewArticle("id", *sku, "", "", *price); err == nil {
		t.Fatal("expected error for empty name")
	}
	if _, err := NewArticle("id", *sku, "   ", "", *price); err == nil {
		t.Fatal("expected error for whitespace-only name")
	}
}

func TestNewArticle_rejectsZeroPrice(t *testing.T) {
	sku, _ := NewSKU("ABC-001")
	zero, _ := NewMoney(0, "EUR")
	if _, err := NewArticle("id", *sku, "X", "", *zero); err == nil {
		t.Fatal("expected error for price=0 (invariant: price > 0)")
	}
}

func TestArticle_ChangePrice_recordsEvent(t *testing.T) {
	sku, _ := NewSKU("ABC-001")
	old, _ := NewMoney(100, "EUR")
	a, _ := NewArticle("id", *sku, "X", "", *old)

	newPrice, _ := NewMoney(200, "EUR")
	if err := a.ChangePrice(*newPrice); err != nil {
		t.Fatalf("expected ChangePrice to succeed, got %v", err)
	}
	if !a.Price.Equals(*newPrice) {
		t.Errorf("expected price=200 after ChangePrice, got %v", a.Price)
	}
	if len(a.PendingEvents()) != 1 {
		t.Fatalf("expected 1 pending event, got %d", len(a.PendingEvents()))
	}
}

func TestArticle_ChangePrice_rejectsZero(t *testing.T) {
	sku, _ := NewSKU("ABC-001")
	old, _ := NewMoney(100, "EUR")
	a, _ := NewArticle("id", *sku, "X", "", *old)

	zero, _ := NewMoney(0, "EUR")
	if err := a.ChangePrice(*zero); err == nil {
		t.Fatal("expected error when changing price to 0")
	}
}

func TestArticle_ChangePrice_rejectsCurrencyMismatch(t *testing.T) {
	sku, _ := NewSKU("ABC-001")
	old, _ := NewMoney(100, "EUR")
	a, _ := NewArticle("id", *sku, "X", "", *old)

	usd, _ := NewMoney(100, "USD")
	if err := a.ChangePrice(*usd); err == nil {
		t.Fatal("expected error when changing currency via ChangePrice")
	}
}

func TestArticle_AdjustInventory_creditsExistingLocation(t *testing.T) {
	sku, _ := NewSKU("ABC-001")
	price, _ := NewMoney(1000, "EUR")
	a, _ := NewArticle("id", *sku, "Widget", "", *price)

	if err := a.AdjustInventory("IT-MILANO1", 10, "initial restock"); err != nil {
		t.Fatalf("AdjustInventory: %v", err)
	}
	// Same location, additional delta.
	if err := a.AdjustInventory("IT-MILANO1", 5, "restock"); err != nil {
		t.Fatalf("AdjustInventory: %v", err)
	}

	if len(a.Inventories) != 1 {
		t.Fatalf("expected 1 inventory level, got %d", len(a.Inventories))
	}
	if a.Inventories[0].Quantity != 15 {
		t.Errorf("expected quantity 15 after two credits, got %d", a.Inventories[0].Quantity)
	}
	if len(a.PendingEvents()) != 2 {
		t.Errorf("expected 2 pending events, got %d", len(a.PendingEvents()))
	}
}

func TestArticle_AdjustInventory_separateLocations(t *testing.T) {
	sku, _ := NewSKU("ABC-001")
	price, _ := NewMoney(1000, "EUR")
	a, _ := NewArticle("id", *sku, "Widget", "", *price)

	_ = a.AdjustInventory("IT-MILANO1", 10, "")
	_ = a.AdjustInventory("IT-ROMA1", 7, "")

	if len(a.Inventories) != 2 {
		t.Fatalf("expected 2 inventory levels, got %d", len(a.Inventories))
	}
}

func TestArticle_AdjustInventory_rejectsNegativeQuantity(t *testing.T) {
	sku, _ := NewSKU("ABC-001")
	price, _ := NewMoney(1000, "EUR")
	a, _ := NewArticle("id", *sku, "Widget", "", *price)
	_ = a.AdjustInventory("IT-MILANO1", 5, "")

	if err := a.AdjustInventory("IT-MILANO1", -10, "shrinkage"); err == nil {
		t.Fatal("expected error: quantity would go negative")
	}
}

func TestArticle_AdjustInventory_rejectsEmptyLocation(t *testing.T) {
	sku, _ := NewSKU("ABC-001")
	price, _ := NewMoney(1000, "EUR")
	a, _ := NewArticle("id", *sku, "Widget", "", *price)

	if err := a.AdjustInventory("", 5, ""); err == nil {
		t.Fatal("expected error for empty location code")
	}
}

func TestArticle_AdjustInventory_zeroDeltaIsNoOp(t *testing.T) {
	sku, _ := NewSKU("ABC-001")
	price, _ := NewMoney(1000, "EUR")
	a, _ := NewArticle("id", *sku, "Widget", "", *price)
	_ = a.AdjustInventory("IT-MILANO1", 5, "")
	a.ClearPendingEvents()

	if err := a.AdjustInventory("IT-MILANO1", 0, ""); err != nil {
		t.Fatalf("zero-delta should be no-op, got %v", err)
	}
	if got := len(a.PendingEvents()); got != 0 {
		t.Errorf("zero-delta should not record an event, got %d", got)
	}
}
