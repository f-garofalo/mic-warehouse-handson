package interfaces_test

import (
	"context"
	"testing"

	"warehouse.local/core/entities"
	"warehouse.local/core/interfaces"
)

// fakeRepo is an in-memory ArticleRepository used only in tests: to prove the
// port is implementable and to encode the idempotent-Save intent. The real
// adapter (MySQL + dual-write) arrives in CP4.
type fakeRepo struct {
	store map[string]*entities.Article
}

func newFakeRepo() *fakeRepo { return &fakeRepo{store: map[string]*entities.Article{}} }

func (r *fakeRepo) Save(ctx context.Context, a *entities.Article) error {
	r.store[a.ID()] = a // upsert by ID -> idempotent for the same state
	return nil
}

func (r *fakeRepo) FindByID(ctx context.Context, id string) (*entities.Article, error) {
	a, ok := r.store[id]
	if !ok {
		return nil, interfaces.ErrArticleNotFound
	}
	return a, nil
}

func (r *fakeRepo) FindBySKU(ctx context.Context, sku entities.SKU) (*entities.Article, error) {
	for _, a := range r.store {
		if a.SKU().Equals(sku) {
			return a, nil
		}
	}
	return nil, interfaces.ErrArticleNotFound
}

func (r *fakeRepo) List(ctx context.Context) ([]*entities.Article, error) {
	out := make([]*entities.Article, 0, len(r.store))
	for _, a := range r.store {
		out = append(out, a)
	}
	return out, nil
}

func (r *fakeRepo) Delete(ctx context.Context, id string) error {
	delete(r.store, id)
	return nil
}

// Compile-time proof that the port is implementable.
var _ interfaces.ArticleRepository = (*fakeRepo)(nil)

func TestSaveIsIdempotentForSameState(t *testing.T) {
	repo := newFakeRepo()
	sku, _ := entities.NewSKU("WID-001")
	price, _ := entities.NewMoney(1000, "EUR")
	a, err := entities.NewArticle("art-1", "Widget", "", sku, price)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()

	if err := repo.Save(ctx, a); err != nil {
		t.Fatal(err)
	}
	if err := repo.Save(ctx, a); err != nil { // same state, twice
		t.Fatal(err)
	}
	all, _ := repo.List(ctx)
	if len(all) != 1 {
		t.Fatalf("saving the same state twice must yield 1 article, got %d", len(all))
	}
	if _, err := repo.FindByID(ctx, "missing"); err != interfaces.ErrArticleNotFound {
		t.Errorf("expected ErrArticleNotFound, got %v", err)
	}
}
