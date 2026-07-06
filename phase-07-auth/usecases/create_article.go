package usecases

import (
	"context"
	"fmt"

	"warehouse.local/core/dispatcher"
	"warehouse.local/core/entities"
	"warehouse.local/core/events"
	"warehouse.local/core/interfaces"
)

// CreateArticleInput carries the primitive request shape; the use case wraps
// these into the aggregate's value objects and validates via NewArticle.
//
// ID is optional: empty means "minted at persistence" (the system of record
// assigns the id on first Save and the output carries it back).
type CreateArticleInput struct {
	ID          string
	SKU         string
	Name        string
	Description string
	PriceCents  int64
	Currency    string
}

// CreateArticleOutput exposes the persisted aggregate to the caller.
type CreateArticleOutput struct {
	Article *entities.Article
}

// CreateArticleUseCase persists a new Article and dispatches its creation event.
// It owns the workflow: build aggregate → Save → translate pending events to
// canonical events → Dispatch → ClearPendingEvents.
type CreateArticleUseCase struct {
	repo       interfaces.ArticleRepository
	dispatcher dispatcher.EventDispatcher
}

func NewCreateArticleUseCase(repo interfaces.ArticleRepository, d dispatcher.EventDispatcher) *CreateArticleUseCase {
	return &CreateArticleUseCase{repo: repo, dispatcher: d}
}

func (uc *CreateArticleUseCase) Execute(ctx context.Context, in CreateArticleInput) (*CreateArticleOutput, error) {
	// in.ID may be empty: the system of record mints the id at first Save.
	sku, err := entities.NewSKU(in.SKU)
	if err != nil {
		return nil, fmt.Errorf("CreateArticle: %w", err)
	}
	price, err := entities.NewMoney(in.PriceCents, in.Currency)
	if err != nil {
		return nil, fmt.Errorf("CreateArticle: %w", err)
	}
	a, err := entities.NewArticle(in.ID, *sku, in.Name, in.Description, *price)
	if err != nil {
		return nil, fmt.Errorf("CreateArticle: %w", err)
	}

	if err := uc.repo.Save(ctx, a); err != nil {
		return nil, fmt.Errorf("CreateArticle: save: %w", err)
	}

	// The canonical creation event is built AFTER Save: when the input carries
	// no id, Save is where the system of record mints it, and the event must
	// name the minted id, not the empty placeholder.
	canonical := events.ArticleCreated{
		ArticleID:   a.ID,
		SKU:         a.SKU.Code,
		ArticleName: a.Name,
		PriceCents:  a.Price.AmountCents,
		Currency:    a.Price.Currency,
		At:          a.CreatedAt,
	}
	if err := uc.dispatcher.Dispatch(ctx, []events.DomainEvent{canonical}); err != nil {
		return nil, fmt.Errorf("CreateArticle: dispatch: %w", err)
	}
	a.ClearPendingEvents()

	return &CreateArticleOutput{Article: a}, nil
}
