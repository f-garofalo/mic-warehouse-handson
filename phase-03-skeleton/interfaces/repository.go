// Package interfaces holds the persistence ports of the Warehouse BC. It
// depends only on the domain (entities) — never the other way round.
package interfaces

import (
	"context"
	"errors"

	"warehouse.local/core/entities"
)

// ErrArticleNotFound is returned by lookups when no article matches.
var ErrArticleNotFound = errors.New("article not found")

// ArticleRepository is the single persistence port of the Warehouse BC. It
// loads and saves the Article aggregate as a whole; inventory levels are
// persisted as part of their Article. There is deliberately NO
// InventoryRepository — that would break the aggregate boundary.
//
// Save is an idempotent upsert keyed by Article ID: calling it twice with an
// aggregate in the same state stores the same bytes, returns no error and
// creates no duplicate. Save does NOT drain or publish the aggregate's pending
// events (Article.PullEvents) — draining and publishing is the caller's job
// (CP5). Implementations arrive in CP4; this is the port only.
type ArticleRepository interface {
	Save(ctx context.Context, article *entities.Article) error
	FindByID(ctx context.Context, id string) (*entities.Article, error)
	FindBySKU(ctx context.Context, sku entities.SKU) (*entities.Article, error)
	List(ctx context.Context) ([]*entities.Article, error)
	Delete(ctx context.Context, id string) error
}
