package usecases

import (
	"context"
	"sync"

	"warehouse.local/core/entities"
	"warehouse.local/core/interfaces"
)

// ErrArticleNotFound aliases the canonical port sentinel so a not-found from any
// adapter (this in-memory fake, or the dual-write/MySQL repo wired in main) is
// matched by the same errors.Is check in the use cases and handlers. CP6 unifies
// what CP4/CP5 kept as separate package-level sentinels.
var ErrArticleNotFound = interfaces.ErrArticleNotFound

// InMemoryArticleRepository is a goroutine-safe map-backed implementation of
// interfaces.ArticleRepository, intended for use case tests only.
type InMemoryArticleRepository struct {
	mu         sync.Mutex
	byID       map[string]*entities.Article
	bySKU      map[string]*entities.Article
	FailOnSave error
	FailOnFind error
}

func NewInMemoryArticleRepository() *InMemoryArticleRepository {
	return &InMemoryArticleRepository{
		byID:  make(map[string]*entities.Article),
		bySKU: make(map[string]*entities.Article),
	}
}

var _ interfaces.ArticleRepository = (*InMemoryArticleRepository)(nil)

func (r *InMemoryArticleRepository) Save(ctx context.Context, a *entities.Article) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.FailOnSave != nil {
		return r.FailOnSave
	}
	r.byID[a.ID] = a
	r.bySKU[a.SKU.Code] = a
	return nil
}

func (r *InMemoryArticleRepository) FindByID(ctx context.Context, id string) (*entities.Article, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.FailOnFind != nil {
		return nil, r.FailOnFind
	}
	a, ok := r.byID[id]
	if !ok {
		return nil, ErrArticleNotFound
	}
	return a, nil
}

func (r *InMemoryArticleRepository) FindBySKU(ctx context.Context, skuCode string) (*entities.Article, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.FailOnFind != nil {
		return nil, r.FailOnFind
	}
	a, ok := r.bySKU[skuCode]
	if !ok {
		return nil, ErrArticleNotFound
	}
	return a, nil
}

func (r *InMemoryArticleRepository) List(ctx context.Context) ([]*entities.Article, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]*entities.Article, 0, len(r.byID))
	for _, a := range r.byID {
		out = append(out, a)
	}
	return out, nil
}

func (r *InMemoryArticleRepository) Delete(ctx context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	a, ok := r.byID[id]
	if !ok {
		return ErrArticleNotFound
	}
	delete(r.byID, id)
	delete(r.bySKU, a.SKU.Code)
	return nil
}

func (r *InMemoryArticleRepository) Count() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.byID)
}
