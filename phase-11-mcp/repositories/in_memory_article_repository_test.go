package repositories

import (
	"context"
	"errors"
	"testing"

	"warehouse.local/core/entities"
)

func mustArticle(t *testing.T, id, skuCode string, priceCents int64) *entities.Article {
	t.Helper()
	sku, err := entities.NewSKU(skuCode)
	if err != nil {
		t.Fatalf("NewSKU: %v", err)
	}
	price, err := entities.NewMoney(priceCents, "EUR")
	if err != nil {
		t.Fatalf("NewMoney: %v", err)
	}
	a, err := entities.NewArticle(id, *sku, "Test "+skuCode, "test", *price)
	if err != nil {
		t.Fatalf("NewArticle: %v", err)
	}
	return a
}

func TestInMemoryArticleRepository_saveAndFind(t *testing.T) {
	ctx := context.Background()
	r := NewInMemoryArticleRepository()

	a := mustArticle(t, "id-1", "ABC-001", 1000)
	if err := r.Save(ctx, a); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := r.FindByID(ctx, "id-1")
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	if got.ID != "id-1" {
		t.Errorf("expected id id-1, got %s", got.ID)
	}

	bySKU, err := r.FindBySKU(ctx, "ABC-001")
	if err != nil {
		t.Fatalf("FindBySKU: %v", err)
	}
	if bySKU.ID != "id-1" {
		t.Errorf("expected id id-1 by SKU, got %s", bySKU.ID)
	}
}

func TestInMemoryArticleRepository_findMissingReturnsErrNotFound(t *testing.T) {
	ctx := context.Background()
	r := NewInMemoryArticleRepository()

	_, err := r.FindByID(ctx, "nope")
	if !errors.Is(err, ErrArticleNotFound) {
		t.Errorf("expected ErrArticleNotFound, got %v", err)
	}
}

func TestInMemoryArticleRepository_listAndDelete(t *testing.T) {
	ctx := context.Background()
	r := NewInMemoryArticleRepository()

	a := mustArticle(t, "id-1", "ABC-001", 1000)
	b := mustArticle(t, "id-2", "ABC-002", 2000)
	_ = r.Save(ctx, a)
	_ = r.Save(ctx, b)

	all, err := r.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(all) != 2 {
		t.Errorf("expected 2 articles, got %d", len(all))
	}

	if err := r.Delete(ctx, "id-1"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if err := r.Delete(ctx, "id-1"); !errors.Is(err, ErrArticleNotFound) {
		t.Errorf("expected ErrArticleNotFound on second delete, got %v", err)
	}
}

func TestInMemoryArticleRepository_failOnSave_simulatesError(t *testing.T) {
	ctx := context.Background()
	r := NewInMemoryArticleRepository()
	r.FailOnSave = errors.New("simulated save failure")

	a := mustArticle(t, "id-1", "ABC-001", 1000)
	if err := r.Save(ctx, a); err == nil {
		t.Fatal("expected simulated failure, got nil")
	}
}
