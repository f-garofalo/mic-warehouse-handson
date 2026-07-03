package repositories

import (
	"context"

	"warehouse.local/core/entities"
	"warehouse.local/core/interfaces"
)

// ReadMode controls where DualWriteArticleRepository routes reads.
//
// Writes always go to BOTH stores (legacy first, BC second), regardless of mode.
// Reads go to exactly ONE store, chosen by the mode (single read).
type ReadMode int

const (
	// ReadFromLegacy routes all reads to the legacy store. Default during early migration.
	ReadFromLegacy ReadMode = iota
	// ReadFromBC routes all reads to the BC store. Use after cutover.
	ReadFromBC
)

// DualWriteArticleRepository wraps two interfaces.ArticleRepository implementations
// (legacy + BC). It writes to both on every mutation and routes reads to a single
// store chosen by mode. Per ADR-013 the decorator is transitional: once cutover
// completes it is deleted.
type DualWriteArticleRepository struct {
	legacy interfaces.ArticleRepository
	bc     interfaces.ArticleRepository
	mode   ReadMode
}

func NewDualWriteArticleRepository(
	legacy interfaces.ArticleRepository,
	bc interfaces.ArticleRepository,
	mode ReadMode,
) *DualWriteArticleRepository {
	return &DualWriteArticleRepository{legacy: legacy, bc: bc, mode: mode}
}

// compile-time interface check
var _ interfaces.ArticleRepository = (*DualWriteArticleRepository)(nil)

// Save writes to legacy FIRST; if legacy fails we return the error and do NOT
// touch BC. Then it writes to BC; if BC fails we return the error — legacy keeps
// the article, there is no rollback policy in this exercise.
func (r *DualWriteArticleRepository) Save(ctx context.Context, a *entities.Article) error {
	if err := r.legacy.Save(ctx, a); err != nil {
		return err
	}
	return r.bc.Save(ctx, a)
}

// Delete follows the same order and failure rule as Save: legacy first, then BC.
func (r *DualWriteArticleRepository) Delete(ctx context.Context, id string) error {
	if err := r.legacy.Delete(ctx, id); err != nil {
		return err
	}
	return r.bc.Delete(ctx, id)
}

func (r *DualWriteArticleRepository) FindByID(ctx context.Context, id string) (*entities.Article, error) {
	return r.readStore().FindByID(ctx, id)
}

func (r *DualWriteArticleRepository) FindBySKU(ctx context.Context, skuCode string) (*entities.Article, error) {
	return r.readStore().FindBySKU(ctx, skuCode)
}

func (r *DualWriteArticleRepository) List(ctx context.Context) ([]*entities.Article, error) {
	return r.readStore().List(ctx)
}

// readStore returns the single store reads are routed to, per mode.
func (r *DualWriteArticleRepository) readStore() interfaces.ArticleRepository {
	if r.mode == ReadFromBC {
		return r.bc
	}
	return r.legacy
}
