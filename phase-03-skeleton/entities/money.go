package entities

import (
	"fmt"
	"regexp"
)

// currencyPattern is the ISO-4217 shape check: three uppercase letters.
var currencyPattern = regexp.MustCompile(`^[A-Z]{3}$`)

// Money is a value object: an amount in integer cents plus a currency. It is
// never a floating-point number. No identity; equality by value.
type Money struct {
	amountCents int64
	currency    string
}

// NewMoney is the factory. It rejects negative amounts and malformed currency
// codes. Money(0, "EUR") is valid; the stricter "price > 0" rule lives on the
// Article aggregate, not here (VOs define what is expressible; aggregates
// define what is acceptable).
func NewMoney(amountCents int64, currency string) (Money, error) {
	if amountCents < 0 {
		return Money{}, fmt.Errorf("invalid Money: amountCents must be >= 0, got %d", amountCents)
	}
	if !currencyPattern.MatchString(currency) {
		return Money{}, fmt.Errorf("invalid Money: currency %q must be 3 uppercase letters (ISO-4217)", currency)
	}
	return Money{amountCents: amountCents, currency: currency}, nil
}

func (m Money) AmountCents() int64 { return m.amountCents }
func (m Money) Currency() string   { return m.currency }

// Equals compares by value.
func (m Money) Equals(other Money) bool {
	return m.amountCents == other.amountCents && m.currency == other.currency
}

func (m Money) String() string { return fmt.Sprintf("%d %s", m.amountCents, m.currency) }
