package usecases

import (
	"context"
	"errors"
	"testing"

	"warehouse.local/core/dispatcher"
	"warehouse.local/core/events"
)

func seedArticle(t *testing.T, repo *InMemoryArticleRepository, id, skuCode string) {
	t.Helper()
	disp := dispatcher.NewInMemoryDispatcher()
	cuc := NewCreateArticleUseCase(repo, disp)
	if _, err := cuc.Execute(context.Background(), CreateArticleInput{
		ID: id, SKU: skuCode, Name: "Seeded", PriceCents: 1000, Currency: "EUR",
	}); err != nil {
		t.Fatalf("seed Create: %v", err)
	}
}

func TestAdjustInventoryUseCase_appliesDeltaAndDispatches(t *testing.T) {
	ctx := context.Background()
	repo := NewInMemoryArticleRepository()
	seedArticle(t, repo, "id-1", "ABC-001")

	disp := dispatcher.NewInMemoryDispatcher()
	uc := NewAdjustInventoryUseCase(repo, disp)

	out, err := uc.Execute(ctx, AdjustInventoryInput{
		ArticleID:    "id-1",
		LocationCode: "IT-MILANO1",
		Delta:        10,
		Reason:       "initial restock",
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if out.Article == nil || len(out.Article.Inventories) != 1 {
		t.Fatalf("expected one inventory level, got %v", out.Article)
	}
	if out.Article.Inventories[0].Quantity != 10 {
		t.Errorf("expected quantity 10, got %d", out.Article.Inventories[0].Quantity)
	}
	got := disp.Events()
	if len(got) != 1 {
		t.Fatalf("expected 1 event, got %d", len(got))
	}
	adjusted, ok := got[0].(events.InventoryAdjusted)
	if !ok {
		t.Fatalf("expected events.InventoryAdjusted, got %T", got[0])
	}
	if adjusted.Delta != 10 || adjusted.NewQuantity != 10 || adjusted.LocationCode != "IT-MILANO1" {
		t.Errorf("unexpected event payload: %+v", adjusted)
	}
}

func TestAdjustInventoryUseCase_articleNotFound(t *testing.T) {
	ctx := context.Background()
	repo := NewInMemoryArticleRepository()
	disp := dispatcher.NewInMemoryDispatcher()
	uc := NewAdjustInventoryUseCase(repo, disp)

	_, err := uc.Execute(ctx, AdjustInventoryInput{
		ArticleID: "nope", LocationCode: "L1", Delta: 5,
	})
	if err == nil {
		t.Fatal("expected error for missing article")
	}
	if len(disp.Events()) != 0 {
		t.Errorf("expected no events on missing article, got %d", len(disp.Events()))
	}
}

func TestAdjustInventoryUseCase_validationFailureBubblesUp(t *testing.T) {
	ctx := context.Background()
	repo := NewInMemoryArticleRepository()
	seedArticle(t, repo, "id-1", "ABC-001")
	disp := dispatcher.NewInMemoryDispatcher()
	uc := NewAdjustInventoryUseCase(repo, disp)

	// Empty location code should fail aggregate validation.
	_, err := uc.Execute(ctx, AdjustInventoryInput{
		ArticleID: "id-1", LocationCode: "", Delta: 5,
	})
	if err == nil {
		t.Fatal("expected error for empty location code")
	}
	if len(disp.Events()) != 0 {
		t.Errorf("expected no events on validation error, got %d", len(disp.Events()))
	}
}

func TestAdjustInventoryUseCase_dispatchFailureSurfaces(t *testing.T) {
	ctx := context.Background()
	repo := NewInMemoryArticleRepository()
	seedArticle(t, repo, "id-1", "ABC-001")
	disp := dispatcher.NewInMemoryDispatcher()
	disp.Reset()
	disp.FailOnDispatch = errors.New("dispatch down")

	uc := NewAdjustInventoryUseCase(repo, disp)
	_, err := uc.Execute(ctx, AdjustInventoryInput{
		ArticleID: "id-1", LocationCode: "L1", Delta: 5,
	})
	if err == nil {
		t.Fatal("expected dispatch failure to surface")
	}
}
