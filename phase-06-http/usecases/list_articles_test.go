package usecases

import (
	"context"
	"fmt"
	"testing"

	"warehouse.local/core/dispatcher"
)

func seedN(t *testing.T, repo *InMemoryArticleRepository, n int) {
	t.Helper()
	create := NewCreateArticleUseCase(repo, dispatcher.NewInMemoryDispatcher())
	for i := 1; i <= n; i++ {
		if _, err := create.Execute(context.Background(), CreateArticleInput{
			ID: fmt.Sprintf("id-%d", i), SKU: fmt.Sprintf("SKU-%03d", i),
			Name: "N", PriceCents: 1000, Currency: "EUR",
		}); err != nil {
			t.Fatalf("seed: %v", err)
		}
	}
}

func TestListArticlesUseCase_paginates(t *testing.T) {
	repo := NewInMemoryArticleRepository()
	seedN(t, repo, 5)
	out, err := NewListArticlesUseCase(repo).Execute(context.Background(), ListArticlesInput{Limit: 2, Offset: 0})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if out.Total != 5 {
		t.Errorf("total = %d, want 5", out.Total)
	}
	if len(out.Articles) != 2 {
		t.Errorf("page len = %d, want 2", len(out.Articles))
	}
	if out.Limit != 2 || out.Offset != 0 {
		t.Errorf("meta wrong: limit=%d offset=%d", out.Limit, out.Offset)
	}
}

func TestListArticlesUseCase_defaultsAndClamp(t *testing.T) {
	out, err := NewListArticlesUseCase(NewInMemoryArticleRepository()).
		Execute(context.Background(), ListArticlesInput{Limit: 0, Offset: -5})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if out.Limit != defaultListLimit || out.Offset != 0 || out.Total != 0 || len(out.Articles) != 0 {
		t.Errorf("defaults/clamp wrong: %+v", out)
	}
}

func TestListArticlesUseCase_offsetBeyondTotal(t *testing.T) {
	repo := NewInMemoryArticleRepository()
	seedN(t, repo, 3)
	out, err := NewListArticlesUseCase(repo).Execute(context.Background(), ListArticlesInput{Limit: 10, Offset: 100})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if out.Total != 3 || len(out.Articles) != 0 {
		t.Errorf("offset beyond total: total=%d len=%d", out.Total, len(out.Articles))
	}
}
