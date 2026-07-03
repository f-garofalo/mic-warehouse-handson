package usecases

import (
	"context"
	"errors"
	"testing"

	"warehouse.local/core/dispatcher"
	"warehouse.local/core/events"
)

// This file is the acceptance spec for the Phase 05 task. It runs against
// in-memory fakes (no MySQL) and is RED until you implement
// ChangeArticlePriceUseCase.Execute in change_article_price.go. Make it green.

func seedPriceArticle(t *testing.T, repo *InMemoryArticleRepository, id, skuCode string, priceCents int64) {
	t.Helper()
	disp := dispatcher.NewInMemoryDispatcher()
	cuc := NewCreateArticleUseCase(repo, disp)
	if _, err := cuc.Execute(context.Background(), CreateArticleInput{
		ID: id, SKU: skuCode, Name: "Seeded", PriceCents: priceCents, Currency: "EUR",
	}); err != nil {
		t.Fatalf("seed Create: %v", err)
	}
}

func TestChangeArticlePriceUseCase_changesPriceSavesAndDispatches(t *testing.T) {
	ctx := context.Background()
	repo := NewInMemoryArticleRepository()
	seedPriceArticle(t, repo, "id-1", "PRICE-001", 1000)
	disp := dispatcher.NewInMemoryDispatcher()
	uc := NewChangeArticlePriceUseCase(repo, disp)

	out, err := uc.Execute(ctx, ChangeArticlePriceInput{
		ArticleID: "id-1", NewPriceCents: 1500, Currency: "EUR",
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if out.Article == nil || out.Article.Price.AmountCents != 1500 {
		t.Fatalf("expected price 1500, got %+v", out.Article)
	}
	got := disp.Events()
	if len(got) != 1 {
		t.Fatalf("expected 1 event, got %d", len(got))
	}
	changed, ok := got[0].(events.ArticlePriceChanged)
	if !ok {
		t.Fatalf("expected events.ArticlePriceChanged, got %T", got[0])
	}
	if changed.OldPriceCents != 1000 || changed.NewPriceCents != 1500 || changed.Currency != "EUR" {
		t.Errorf("unexpected event payload: %+v", changed)
	}
}

func TestChangeArticlePriceUseCase_samePriceIsNoOp(t *testing.T) {
	ctx := context.Background()
	repo := NewInMemoryArticleRepository()
	seedPriceArticle(t, repo, "id-1", "PRICE-001", 1000)
	repo.FailOnSave = errors.New("save should not be called")
	disp := dispatcher.NewInMemoryDispatcher()
	uc := NewChangeArticlePriceUseCase(repo, disp)

	out, err := uc.Execute(ctx, ChangeArticlePriceInput{
		ArticleID: "id-1", NewPriceCents: 1000, Currency: "EUR",
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if out.Article == nil || out.Article.Price.AmountCents != 1000 {
		t.Fatalf("expected unchanged price 1000, got %+v", out.Article)
	}
	if len(disp.Events()) != 0 {
		t.Errorf("expected no event on same price, got %d", len(disp.Events()))
	}
}

func TestChangeArticlePriceUseCase_rejectsCurrencyChange(t *testing.T) {
	ctx := context.Background()
	repo := NewInMemoryArticleRepository()
	seedPriceArticle(t, repo, "id-1", "PRICE-001", 1000)
	disp := dispatcher.NewInMemoryDispatcher()
	uc := NewChangeArticlePriceUseCase(repo, disp)

	_, err := uc.Execute(ctx, ChangeArticlePriceInput{
		ArticleID: "id-1", NewPriceCents: 1500, Currency: "USD",
	})
	if err == nil {
		t.Fatal("expected currency change error")
	}
	if len(disp.Events()) != 0 {
		t.Errorf("expected no events on currency change, got %d", len(disp.Events()))
	}
}

func TestChangeArticlePriceUseCase_findFailureSurfacesAndDoesNotDispatch(t *testing.T) {
	repo := NewInMemoryArticleRepository()
	repo.FailOnFind = errors.New("repo down")
	disp := dispatcher.NewInMemoryDispatcher()
	uc := NewChangeArticlePriceUseCase(repo, disp)

	_, err := uc.Execute(context.Background(), ChangeArticlePriceInput{
		ArticleID: "id-1", NewPriceCents: 1500, Currency: "EUR",
	})
	if err == nil {
		t.Fatal("expected find failure to surface")
	}
	if len(disp.Events()) != 0 {
		t.Errorf("expected no dispatch when find fails, got %d", len(disp.Events()))
	}
}

func TestChangeArticlePriceUseCase_saveFailureSurfacesAndDoesNotDispatch(t *testing.T) {
	ctx := context.Background()
	repo := NewInMemoryArticleRepository()
	seedPriceArticle(t, repo, "id-1", "PRICE-001", 1000)
	repo.FailOnSave = errors.New("save down")
	disp := dispatcher.NewInMemoryDispatcher()
	uc := NewChangeArticlePriceUseCase(repo, disp)

	_, err := uc.Execute(ctx, ChangeArticlePriceInput{
		ArticleID: "id-1", NewPriceCents: 1500, Currency: "EUR",
	})
	if err == nil {
		t.Fatal("expected save failure to surface")
	}
	if len(disp.Events()) != 0 {
		t.Errorf("expected no dispatch when save fails, got %d", len(disp.Events()))
	}
}

func TestChangeArticlePriceUseCase_dispatchFailureSurfaces(t *testing.T) {
	ctx := context.Background()
	repo := NewInMemoryArticleRepository()
	seedPriceArticle(t, repo, "id-1", "PRICE-001", 1000)
	disp := dispatcher.NewInMemoryDispatcher()
	disp.FailOnDispatch = errors.New("hermes unreachable")
	uc := NewChangeArticlePriceUseCase(repo, disp)

	_, err := uc.Execute(ctx, ChangeArticlePriceInput{
		ArticleID: "id-1", NewPriceCents: 1500, Currency: "EUR",
	})
	if err == nil {
		t.Fatal("expected dispatch failure to surface")
	}
}
