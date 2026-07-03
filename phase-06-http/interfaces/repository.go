package interfaces

import (
	"context"
	"errors"

	"warehouse.local/core/entities"
)

// ErrArticleNotFound is the canonical "aggregate not found" sentinel of the
// repository port. Every adapter returns it, and use cases + handlers match it
// via errors.Is. CP6 unifies the separate sentinels that CP4 (repositories) and
// CP5 (usecases) each declared into this single one.
var ErrArticleNotFound = errors.New("article not found")

// ArticleRepository persists and retrieves the Article aggregate.
// Per ADR-011: only aggregate-level operations exposed; no per-entity or
// per-value-object methods.
//
// The repository is responsible for loading the aggregate's contents
// (InventoryLevels, etc.) atomically and for persisting them in one transaction.
type ArticleRepository interface {
	// Save persists a new aggregate or updates an existing one.
	// Implementation must be idempotent for the same aggregate state.
	Save(ctx context.Context, article *entities.Article) error

	// FindByID returns the aggregate, or (nil, ErrNotFound) if absent.
	FindByID(ctx context.Context, id string) (*entities.Article, error)

	// FindBySKU returns the aggregate by its SKU code, or (nil, ErrNotFound) if absent.
	FindBySKU(ctx context.Context, skuCode string) (*entities.Article, error)

	// List returns all aggregates. Use with care; intended for admin/back-office paths only.
	List(ctx context.Context) ([]*entities.Article, error)

	// Delete removes the aggregate and its contents.
	Delete(ctx context.Context, id string) error
}
