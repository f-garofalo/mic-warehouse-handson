package entities

import "fmt"

// InventoryLevel is an entity inside the Article aggregate: the stock of one
// Article at one warehouse location. It has identity (ID) but is created,
// loaded and mutated only through Article — it has no repository of its own.
//
// Invariants, true at every moment: quantity >= 0, reserved >= 0,
// reserved <= quantity.
type InventoryLevel struct {
	id           string
	locationCode string
	quantity     int64
	reserved     int64
}

// newInventoryLevel is unexported: only Article builds inventory levels.
func newInventoryLevel(id, locationCode string, quantity, reserved int64) (*InventoryLevel, error) {
	if id == "" {
		return nil, fmt.Errorf("inventory level id must not be empty")
	}
	if locationCode == "" {
		return nil, fmt.Errorf("inventory level locationCode must not be empty")
	}
	if quantity < 0 {
		return nil, fmt.Errorf("inventory level quantity must be >= 0, got %d", quantity)
	}
	if reserved < 0 {
		return nil, fmt.Errorf("inventory level reserved must be >= 0, got %d", reserved)
	}
	if reserved > quantity {
		return nil, fmt.Errorf("inventory level reserved (%d) must not exceed quantity (%d)", reserved, quantity)
	}
	return &InventoryLevel{id: id, locationCode: locationCode, quantity: quantity, reserved: reserved}, nil
}

// adjust changes quantity by delta. Unexported: callers go through
// Article.AdjustInventory. The result must stay >= 0 and must not fall below
// the units already reserved (or reserved <= quantity would break).
func (l *InventoryLevel) adjust(delta int64) (int64, error) {
	newQty := l.quantity + delta
	if newQty < 0 {
		return l.quantity, fmt.Errorf("adjust would make quantity negative (%d %+d = %d)", l.quantity, delta, newQty)
	}
	if newQty < l.reserved {
		return l.quantity, fmt.Errorf("adjust would drop quantity (%d) below reserved (%d)", newQty, l.reserved)
	}
	l.quantity = newQty
	return l.quantity, nil
}

// reserve adds qty to reserved, keeping reserved <= quantity. Unexported:
// callers go through Article.ReserveStock.
func (l *InventoryLevel) reserve(qty int64) error {
	if qty <= 0 {
		return fmt.Errorf("reserve quantity must be > 0, got %d", qty)
	}
	if l.reserved+qty > l.quantity {
		return fmt.Errorf("cannot reserve %d: only %d available (quantity %d, reserved %d)",
			qty, l.quantity-l.reserved, l.quantity, l.reserved)
	}
	l.reserved += qty
	return nil
}

func (l *InventoryLevel) ID() string           { return l.id }
func (l *InventoryLevel) LocationCode() string { return l.locationCode }
func (l *InventoryLevel) Quantity() int64      { return l.quantity }
func (l *InventoryLevel) Reserved() int64      { return l.reserved }
func (l *InventoryLevel) Available() int64     { return l.quantity - l.reserved }
