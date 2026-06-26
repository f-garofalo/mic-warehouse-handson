package entities

import "testing"

func TestNewMoney(t *testing.T) {
	tests := []struct {
		name     string
		cents    int64
		currency string
		wantErr  bool
	}{
		{"valid", 1999, "EUR", false},
		{"zero is valid", 0, "EUR", false},
		{"negative rejected", -1, "EUR", true},
		{"lowercase currency", 100, "eur", true},
		{"too short currency", 100, "EU", true},
		{"too long currency", 100, "EURO", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewMoney(tt.cents, tt.currency)
			if (err != nil) != tt.wantErr {
				t.Fatalf("NewMoney(%d,%q) err=%v, wantErr=%v", tt.cents, tt.currency, err, tt.wantErr)
			}
		})
	}
}

func TestMoneyEqualsByValue(t *testing.T) {
	a, _ := NewMoney(500, "EUR")
	b, _ := NewMoney(500, "EUR")
	cur, _ := NewMoney(500, "USD")
	amt, _ := NewMoney(600, "EUR")
	if !a.Equals(b) {
		t.Error("same amount and currency must be Equal")
	}
	if a.Equals(cur) || a.Equals(amt) {
		t.Error("different currency or amount must not be Equal")
	}
}
