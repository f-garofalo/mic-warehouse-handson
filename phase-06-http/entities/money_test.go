package entities

import "testing"

func TestNewMoney_validInputs(t *testing.T) {
	m, err := NewMoney(2999, "EUR")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if m.AmountCents != 2999 {
		t.Errorf("expected 2999 cents, got %d", m.AmountCents)
	}
	if m.Currency != "EUR" {
		t.Errorf("expected EUR, got %s", m.Currency)
	}
}

func TestNewMoney_rejectsNegativeAmount(t *testing.T) {
	_, err := NewMoney(-1, "EUR")
	if err == nil {
		t.Fatal("expected error for negative amount, got nil")
	}
}

func TestNewMoney_rejectsEmptyCurrency(t *testing.T) {
	_, err := NewMoney(100, "")
	if err == nil {
		t.Fatal("expected error for empty currency, got nil")
	}
}

func TestNewMoney_rejectsInvalidCurrency(t *testing.T) {
	_, err := NewMoney(100, "EU")
	if err == nil {
		t.Fatal("expected error for non-3-letter currency, got nil")
	}
}

func TestMoney_equalityByValue(t *testing.T) {
	a, _ := NewMoney(100, "EUR")
	b, _ := NewMoney(100, "EUR")
	c, _ := NewMoney(100, "USD")

	if !a.Equals(*b) {
		t.Error("expected (100 EUR) to equal (100 EUR)")
	}
	if a.Equals(*c) {
		t.Error("expected (100 EUR) NOT to equal (100 USD)")
	}
}

func TestMoney_decimalView(t *testing.T) {
	m, _ := NewMoney(2999, "EUR")
	if got := m.AsDecimal(); got != 29.99 {
		t.Errorf("expected 29.99, got %v", got)
	}
}
