package usecases

import (
	"context"
	"errors"
	"testing"

	"warehouse.local/core/dispatcher"
)

// Acceptance spec for the GetArticle slice (use case side). RED until you
// implement GetArticleUseCase.Execute in get_article.go. Make it green, then
// wire the handler + route and verify GET /articles/:id with curl.

func TestGetArticleUseCase_returnsSeededArticle(t *testing.T) {
	ctx := context.Background()
	repo := NewInMemoryArticleRepository()
	create := NewCreateArticleUseCase(repo, dispatcher.NewInMemoryDispatcher())
	if _, err := create.Execute(ctx, CreateArticleInput{
		ID: "id-1", SKU: "GET-001", Name: "Widget", PriceCents: 1000, Currency: "EUR",
	}); err != nil {
		t.Fatalf("seed: %v", err)
	}

	uc := NewGetArticleUseCase(repo)
	out, err := uc.Execute(ctx, GetArticleInput{ID: "id-1"})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if out.Article == nil || out.Article.ID != "id-1" || out.Article.Name != "Widget" {
		t.Fatalf("expected article id-1 Widget, got %+v", out.Article)
	}
}

func TestGetArticleUseCase_missingReturnsNotFound(t *testing.T) {
	uc := NewGetArticleUseCase(NewInMemoryArticleRepository())
	_, err := uc.Execute(context.Background(), GetArticleInput{ID: "nope"})
	if err == nil {
		t.Fatal("expected error for a missing article")
	}
	// The handler relies on this: the use case must wrap the repo error so
	// errors.Is still finds ErrArticleNotFound.
	if !errors.Is(err, ErrArticleNotFound) {
		t.Errorf("expected ErrArticleNotFound (wrapped), got %v", err)
	}
}

func TestGetArticleUseCase_emptyIDRejected(t *testing.T) {
	uc := NewGetArticleUseCase(NewInMemoryArticleRepository())
	if _, err := uc.Execute(context.Background(), GetArticleInput{ID: ""}); err == nil {
		t.Fatal("expected error for an empty ID")
	}
}
