package entities

import (
	"errors"
	"strings"
	"time"
)

// InventoryLevel is an entity within the Article aggregate.
// One InventoryLevel exists per (Article, WarehouseLocation) pair.
// Per ADR-011: entities have identity (ID), but they are loaded and saved
// only via their aggregate root (Article).
type InventoryLevel struct {
	ID           string    `json:"id" db:"id"`
	ArticleID    string    `json:"article_id" db:"article_id"`
	LocationCode string    `json:"location_code" db:"location_code"`
	Quantity     int32     `json:"quantity" db:"quantity"`
	Reserved     int32     `json:"reserved" db:"reserved"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
}

// NewInventoryLevel constructs an InventoryLevel for a fresh location.
// Invariants:
//   - ID, ArticleID, LocationCode all non-empty
//   - Quantity and Reserved non-negative
//   - Reserved <= Quantity
func NewInventoryLevel(id, articleID, locationCode string, quantity, reserved int32) (*InventoryLevel, error) {
	if strings.TrimSpace(id) == "" {
		return nil, errors.New("inventory: id is required")
	}
	if strings.TrimSpace(articleID) == "" {
		return nil, errors.New("inventory: articleID is required")
	}
	if strings.TrimSpace(locationCode) == "" {
		return nil, errors.New("inventory: locationCode is required")
	}
	if quantity < 0 {
		return nil, errors.New("inventory: quantity must be >= 0")
	}
	if reserved < 0 {
		return nil, errors.New("inventory: reserved must be >= 0")
	}
	if reserved > quantity {
		return nil, errors.New("inventory: reserved must be <= quantity")
	}
	now := time.Now().UTC()
	return &InventoryLevel{
		ID:           id,
		ArticleID:    articleID,
		LocationCode: locationCode,
		Quantity:     quantity,
		Reserved:     reserved,
		CreatedAt:    now,
		UpdatedAt:    now,
	}, nil
}
