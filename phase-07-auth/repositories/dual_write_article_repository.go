package repositories

import (
	"context"

	"warehouse.local/core/entities"
	"warehouse.local/core/interfaces"
)

// ReadMode controls where DualWriteArticleRepository routes reads.
//
// Writes always go to BOTH stores (legacy first, BC second), regardless of mode.
// Reads go to exactly ONE store, chosen by the mode (single read). See ADR-013.
type ReadMode int

const (
	// ReadFromLegacy routes all reads to the legacy store. Default during early migration.
	ReadFromLegacy ReadMode = iota
	// ReadFromBC routes all reads to the BC store. Use after cutover.
	ReadFromBC
)

// DualWriteArticleRepository wraps two interfaces.ArticleRepository implementations
// (legacy + BC). It writes to both on every mutation and routes reads to a single
// store chosen by mode.
//
// This is the artifact built in Phase 04, now in production: in this phase the
// "legacy" side is the real ACL over the MIC monolith's business_data table
// (LegacyMICArticleRepository) and the "BC" side is the warehouse_db adapter
// (MySQLArticleRepository). The read dial is flipped via the READ_MODE env var.
//
// Per ADR-013: this decorator is transitional. Once cutover completes (mode is
// ReadFromBC and writes to legacy can stop), it is deleted.
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

var _ interfaces.ArticleRepository = (*DualWriteArticleRepository)(nil)

// Save writes to legacy first. If legacy fails, the BC is not touched and the
// error is returned. If legacy succeeds and BC fails, the error is returned;
// legacy keeps the article (there is no rollback policy in this exercise).
//
// Ordering matters for creates: the legacy store is the system of record and
// MINTS the id (legacy.Save assigns aggregate.ID on insert), so the subsequent
// bc.Save persists the warehouse row under the SAME id. Both stores stay
// keyed identically, which is what makes the read dial flippable.
func (r *DualWriteArticleRepository) Save(ctx context.Context, a *entities.Article) error {
	if err := r.legacy.Save(ctx, a); err != nil {
		return err
	}
	if err := r.bc.Save(ctx, a); err != nil {
		return err
	}
	return nil
}

// Delete removes from legacy first, then BC (same failure rule as Save).
func (r *DualWriteArticleRepository) Delete(ctx context.Context, id string) error {
	if err := r.legacy.Delete(ctx, id); err != nil {
		return err
	}
	if err := r.bc.Delete(ctx, id); err != nil {
		return err
	}
	return nil
}

func (r *DualWriteArticleRepository) FindByID(ctx context.Context, id string) (*entities.Article, error) {
	return r.reader().FindByID(ctx, id)
}

func (r *DualWriteArticleRepository) FindBySKU(ctx context.Context, skuCode string) (*entities.Article, error) {
	return r.reader().FindBySKU(ctx, skuCode)
}

func (r *DualWriteArticleRepository) List(ctx context.Context) ([]*entities.Article, error) {
	return r.reader().List(ctx)
}

// reader returns the single store reads are routed to, per the current mode.
func (r *DualWriteArticleRepository) reader() interfaces.ArticleRepository {
	if r.mode == ReadFromBC {
		return r.bc
	}
	return r.legacy
}
