package entities

import (
	"fmt"
	"regexp"
)

// skuPattern enforces the Warehouse SKU shape: uppercase letters, digits and
// hyphens, length 3..32. Compiled once at package load.
var skuPattern = regexp.MustCompile(`^[A-Z0-9-]{3,32}$`)

// SKU is a value object: the article code. It has no identity; two SKUs with
// the same code are the same SKU. Built only through NewSKU, so any SKU the
// domain holds is valid by construction.
type SKU struct {
	code string
}

// NewSKU is the factory. It fails loud on anything that does not match
// ^[A-Z0-9-]{3,32}$.
func NewSKU(code string) (SKU, error) {
	if !skuPattern.MatchString(code) {
		return SKU{}, fmt.Errorf("invalid SKU %q: must match ^[A-Z0-9-]{3,32}$", code)
	}
	return SKU{code: code}, nil
}

func (s SKU) Code() string { return s.code }

// Equals compares by value.
func (s SKU) Equals(other SKU) bool { return s.code == other.code }

func (s SKU) String() string { return s.code }
