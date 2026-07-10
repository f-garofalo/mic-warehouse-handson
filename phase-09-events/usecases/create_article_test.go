package usecases

import (
	"context"
	"errors"
	"testing"

	"warehouse.local/core/dispatcher"
	"warehouse.local/core/events"
)

func TestCreateArticleUseCase_savesAndDispatches(t *testing.T) {
	repo := NewInMemoryArticleRepository()
	disp := dispatcher.NewInMemoryDispatcher()
	uc := NewCreateArticleUseCase(repo, disp)

	out, err := uc.Execute(context.Background(), CreateArticleInput{
		ID:          "id-1",
		SKU:         "TEST-001",
		Name:        "Test Article",
		Description: "A test article",
		PriceCents:  9999,
		Currency:    "EUR",
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if out.Article == nil || out.Article.ID != "id-1" {
		t.Fatalf("expected aggregate id-1, got %v", out.Article)
	}
	if repo.Count() != 1 {
		t.Errorf("expected 1 article persisted, got %d", repo.Count())
	}
	got := disp.Events()
	if len(got) != 1 {
		t.Fatalf("expected 1 dispatched event, got %d", len(got))
	}
	created, ok := got[0].(events.ArticleCreated)
	if !ok {
		t.Fatalf("expected events.ArticleCreated, got %T", got[0])
	}
	if created.SKU != "TEST-001" || created.PriceCents != 9999 || created.Currency != "EUR" {
		t.Errorf("unexpected event payload: %+v", created)
	}
}

func TestCreateArticleUseCase_mintsIDWhenEmpty(t *testing.T) {
	repo := NewInMemoryArticleRepository()
	disp := dispatcher.NewInMemoryDispatcher()
	uc := NewCreateArticleUseCase(repo, disp)

	out, err := uc.Execute(context.Background(), CreateArticleInput{
		SKU:        "TEST-003",
		Name:       "Minted",
		PriceCents: 990,
		Currency:   "EUR",
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	// The in-memory fake mints deterministic ids starting at "1", mimicking
	// the legacy store's AUTO_INCREMENT.
	if out.Article.ID != "1" {
		t.Errorf("expected minted id %q, got %q", "1", out.Article.ID)
	}
	got := disp.Events()
	if len(got) != 1 {
		t.Fatalf("expected 1 dispatched event, got %d", len(got))
	}
	created, ok := got[0].(events.ArticleCreated)
	if !ok {
		t.Fatalf("expected events.ArticleCreated, got %T", got[0])
	}
	if created.ArticleID != "1" {
		t.Errorf("expected event to carry the minted id, got %q", created.ArticleID)
	}
}

func TestCreateArticleUseCase_validationErrors(t *testing.T) {
	repo := NewInMemoryArticleRepository()
	disp := dispatcher.NewInMemoryDispatcher()
	uc := NewCreateArticleUseCase(repo, disp)

	// Note: a missing id is NOT a validation error; it means "minted at
	// persistence" (see TestCreateArticleUseCase_mintsIDWhenEmpty).
	cases := []struct {
		name string
		in   CreateArticleInput
	}{
		{"missing SKU", CreateArticleInput{ID: "id", Name: "X", PriceCents: 100, Currency: "EUR"}},
		{"missing Name", CreateArticleInput{ID: "id", SKU: "TEST-001", PriceCents: 100, Currency: "EUR"}},
		{"zero price", CreateArticleInput{ID: "id", SKU: "TEST-001", Name: "X", PriceCents: 0, Currency: "EUR"}},
		{"missing currency", CreateArticleInput{ID: "id", SKU: "TEST-001", Name: "X", PriceCents: 100}},
		{"bad SKU", CreateArticleInput{ID: "id", SKU: "bad", Name: "X", PriceCents: 100, Currency: "EUR"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := uc.Execute(context.Background(), c.in)
			if err == nil {
				t.Errorf("expected error for case %q", c.name)
			}
		})
	}
	if repo.Count() != 0 {
		t.Errorf("expected no articles persisted on validation error, got %d", repo.Count())
	}
	if len(disp.Events()) != 0 {
		t.Errorf("expected no events dispatched on validation error, got %d", len(disp.Events()))
	}
}

func TestCreateArticleUseCase_repoFailureSurfacesAndDoesNotDispatch(t *testing.T) {
	repo := NewInMemoryArticleRepository()
	repo.FailOnSave = errors.New("repo down")
	disp := dispatcher.NewInMemoryDispatcher()
	uc := NewCreateArticleUseCase(repo, disp)

	_, err := uc.Execute(context.Background(), CreateArticleInput{
		ID: "id-1", SKU: "TEST-001", Name: "X", PriceCents: 100, Currency: "EUR",
	})
	if err == nil {
		t.Fatal("expected repo failure to surface")
	}
	if len(disp.Events()) != 0 {
		t.Errorf("expected no dispatch when repo fails, got %d", len(disp.Events()))
	}
}

func TestGetArticleUseCase_returnsAggregate(t *testing.T) {
	ctx := context.Background()
	repo := NewInMemoryArticleRepository()
	disp := dispatcher.NewInMemoryDispatcher()

	cuc := NewCreateArticleUseCase(repo, disp)
	_, err := cuc.Execute(ctx, CreateArticleInput{
		ID: "id-2", SKU: "TEST-002", Name: "Test 2", PriceCents: 5000, Currency: "EUR",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	uc := NewGetArticleUseCase(repo)
	got, err := uc.Execute(ctx, GetArticleInput{ID: "id-2"})
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Article == nil || got.Article.SKU.Code != "TEST-002" {
		t.Errorf("expected SKU TEST-002, got %v", got.Article)
	}
}
