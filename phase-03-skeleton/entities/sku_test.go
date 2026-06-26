package entities

import (
	"strings"
	"testing"
)

func TestNewSKU(t *testing.T) {
	tests := []struct {
		name    string
		code    string
		wantErr bool
	}{
		{"valid simple", "ABC", false},
		{"valid digits and hyphen", "SKU-001", false},
		{"min length 3", strings.Repeat("A", 3), false},
		{"max length 32", strings.Repeat("A", 32), false},
		{"too short 2", strings.Repeat("A", 2), true},
		{"too long 33", strings.Repeat("A", 33), true},
		{"lowercase", "abc", true},
		{"illegal underscore", "AB_C", true},
		{"space", "AB C", true},
		{"empty", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewSKU(tt.code)
			if (err != nil) != tt.wantErr {
				t.Fatalf("NewSKU(%q) err=%v, wantErr=%v", tt.code, err, tt.wantErr)
			}
		})
	}
}

func TestSKUEqualsByValue(t *testing.T) {
	a, _ := NewSKU("SKU-1")
	b, _ := NewSKU("SKU-1")
	c, _ := NewSKU("SKU-2")
	if !a.Equals(b) {
		t.Error("two SKUs with the same code must be Equal")
	}
	if a.Equals(c) {
		t.Error("two SKUs with different codes must not be Equal")
	}
}
