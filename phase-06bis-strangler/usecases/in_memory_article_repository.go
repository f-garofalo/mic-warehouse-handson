package usecases

import (
	"context"
	"errors"
	"strconv"
	"sync"

	"warehouse.local/core/entities"
	"warehouse.local/core/interfaces"
)

// ErrArticleNotFound is returned by the in-memory repo when no row matches.
// Mirrors phase-04's repositories.ErrArticleNotFound; we duplicate here to
// avoid importing phase-04's persistence package into use case tests.
var ErrArticleNotFound = errors.New("usecases: article not found")

// InMemoryArticleRepository is a goroutine-safe map-backed implementation of
// interfaces.ArticleRepository, intended for use case tests only.
type InMemoryArticleRepository struct {
	mu         sync.Mutex
	byID       map[string]*entities.Article
	bySKU      map[string]*entities.Article
	nextID     int64
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
	// Mimic the system of record minting the id: like the legacy store's
	// AUTO_INCREMENT, an empty id gets the next deterministic decimal string
	// ("1", "2", ...) assigned at first Save.
	if a.ID == "" {
		r.nextID++
		a.ID = strconv.FormatInt(r.nextID, 10)
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
