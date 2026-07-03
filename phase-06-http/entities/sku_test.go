package entities

import "testing"

func TestNewSKU_acceptsValid(t *testing.T) {
	cases := []string{"ABC-001", "WIDGET42", "A1B-2C3", "AAA"}
	for _, code := range cases {
		s, err := NewSKU(code)
		if err != nil {
			t.Errorf("expected %q to be valid, got %v", code, err)
		}
		if s.Code != code {
			t.Errorf("expected Code=%q, got %q", code, s.Code)
		}
	}
}

func TestNewSKU_rejectsInvalid(t *testing.T) {
	cases := []struct {
		name, code string
	}{
		{"empty", ""},
		{"too short", "AB"},
		{"too long (33 chars)", "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456"},
		{"lowercase", "abc-001"},
		{"spaces", "ABC 001"},
		{"underscore", "ABC_001"},
		{"unicode", "ABC-Ω01"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if _, err := NewSKU(c.code); err == nil {
				t.Errorf("expected %q to be invalid, got no error", c.code)
			}
		})
	}
}

func TestSKU_equalityByValue(t *testing.T) {
	a, _ := NewSKU("ABC-001")
	b, _ := NewSKU("ABC-001")
	c, _ := NewSKU("ABC-002")
	if !a.Equals(*b) {
		t.Error("expected ABC-001 == ABC-001")
	}
	if a.Equals(*c) {
		t.Error("expected ABC-001 != ABC-002")
	}
}
