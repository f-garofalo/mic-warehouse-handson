package entities

import (
	"errors"
	"regexp"
)

// SKU is a value object representing a Stock Keeping Unit code.
// Per ADR-011: no identity; equality by value; validated on construction.
type SKU struct {
	Code string
}

var skuPattern = regexp.MustCompile(`^[A-Z0-9-]{3,32}$`)

// NewSKU constructs a SKU, rejecting codes that do not match
// `^[A-Z0-9-]{3,32}$` (uppercase letters, digits, hyphens; length 3..32).
func NewSKU(code string) (*SKU, error) {
	if !skuPattern.MatchString(code) {
		return nil, errors.New("sku: must match ^[A-Z0-9-]{3,32}$")
	}
	return &SKU{Code: code}, nil
}

// Equals reports whether two SKUs are equal.
func (s SKU) Equals(other SKU) bool {
	return s.Code == other.Code
}

// String implements fmt.Stringer for log-friendly output.
func (s SKU) String() string { return s.Code }
