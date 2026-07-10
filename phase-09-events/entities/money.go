package entities

import (
	"errors"
	"regexp"
)

// Money is a value object representing an amount in a given currency.
// Per ADR-011 (DDD building blocks): no identity; equality by value;
// constructed via factory; immutable from the consumer's perspective.
type Money struct {
	AmountCents int64
	Currency    string
}

var iso4217Pattern = regexp.MustCompile(`^[A-Z]{3}$`)

// NewMoney constructs a Money value object, validating invariants.
//   - AmountCents must be >= 0
//   - Currency must match ISO 4217 short form (three uppercase letters)
func NewMoney(amountCents int64, currency string) (*Money, error) {
	if amountCents < 0 {
		return nil, errors.New("money: amount cents must be >= 0")
	}
	if !iso4217Pattern.MatchString(currency) {
		return nil, errors.New("money: currency must be ISO 4217 (three uppercase letters)")
	}
	return &Money{AmountCents: amountCents, Currency: currency}, nil
}

// Equals reports whether two Money values are equal.
func (m Money) Equals(other Money) bool {
	return m.AmountCents == other.AmountCents && m.Currency == other.Currency
}

// AsDecimal returns the amount as a float64 in major units (e.g. EUR).
// Provided for display; do not use for arithmetic.
func (m Money) AsDecimal() float64 {
	return float64(m.AmountCents) / 100.0
}
