package repositories

import "testing"

// These tests are the acceptance criteria for the price conversion in Phase 04
// Task 1. The conversions are pure functions (no database), so they are the
// fastest way to know your legacy adapter handles money correctly. They are RED
// until you implement centsToDecimal / decimalToCents in
// legacy_article_repository.go. Make them green — WITHOUT float64.

func TestCentsToDecimal(t *testing.T) {
	cases := []struct {
		cents int64
		want  string
	}{
		{0, "0.00"},
		{5, "0.05"},
		{99, "0.99"},
		{100, "1.00"},
		{2999, "29.99"},
		{123456, "1234.56"},
	}
	for _, c := range cases {
		if got := centsToDecimal(c.cents); got != c.want {
			t.Errorf("centsToDecimal(%d) = %q, want %q", c.cents, got, c.want)
		}
	}
}

func TestDecimalToCents(t *testing.T) {
	cases := []struct {
		in   string
		want int64
	}{
		{"0.00", 0},
		{"0.05", 5},
		{"0.99", 99},
		{"1", 100},
		{"1.5", 150},
		{"29.99", 2999},
		{"1234.56", 123456},
	}
	for _, c := range cases {
		got, err := decimalToCents(c.in)
		if err != nil {
			t.Fatalf("decimalToCents(%q) returned error: %v", c.in, err)
		}
		if got != c.want {
			t.Errorf("decimalToCents(%q) = %d, want %d", c.in, got, c.want)
		}
	}
}

func TestConversionRoundTrip(t *testing.T) {
	for _, cents := range []int64{0, 5, 99, 100, 2999, 123456} {
		s := centsToDecimal(cents)
		got, err := decimalToCents(s)
		if err != nil {
			t.Fatalf("round-trip %d -> %q returned error: %v", cents, s, err)
		}
		if got != cents {
			t.Errorf("round-trip mismatch: %d -> %q -> %d", cents, s, got)
		}
	}
}
