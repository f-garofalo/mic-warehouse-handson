package usecases

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"warehouse.local/core/dispatcher"
	"warehouse.local/core/entities"
	"warehouse.local/core/events"
	"warehouse.local/core/interfaces"
)

// AdjustInventoryInput carries the parameters for a stock-level change.
// Delta may be positive (restock) or negative (decrement).
type AdjustInventoryInput struct {
	ArticleID    string
	LocationCode string
	Delta        int32
	Reason       string
}

// AdjustInventoryResult returns the mutated aggregate.
type AdjustInventoryResult struct {
	Article *entities.Article
}

// AdjustInventoryUseCase loads the aggregate, applies the delta, persists, and
// dispatches the canonical InventoryAdjusted event.
type AdjustInventoryUseCase struct {
	repo       interfaces.ArticleRepository
	dispatcher dispatcher.EventDispatcher
}

func NewAdjustInventoryUseCase(repo interfaces.ArticleRepository, d dispatcher.EventDispatcher) *AdjustInventoryUseCase {
	return &AdjustInventoryUseCase{repo: repo, dispatcher: d}
}

func (uc *AdjustInventoryUseCase) Execute(ctx context.Context, in AdjustInventoryInput) (*AdjustInventoryResult, error) {
	if strings.TrimSpace(in.ArticleID) == "" {
		return nil, errors.New("AdjustInventory: ArticleID is required")
	}
	a, err := uc.repo.FindByID(ctx, in.ArticleID)
	if err != nil {
		return nil, fmt.Errorf("AdjustInventory: load: %w", err)
	}
	if err := a.AdjustInventory(in.LocationCode, in.Delta, in.Reason); err != nil {
		return nil, fmt.Errorf("AdjustInventory: %w", err)
	}
	if err := uc.repo.Save(ctx, a); err != nil {
		return nil, fmt.Errorf("AdjustInventory: save: %w", err)
	}

	var newQty int32
	for _, lvl := range a.Inventories {
		if lvl.LocationCode == in.LocationCode {
			newQty = lvl.Quantity
			break
		}
	}
	canonical := events.InventoryAdjusted{
		ArticleID:    a.ID,
		LocationCode: in.LocationCode,
		Delta:        in.Delta,
		NewQuantity:  newQty,
		Reason:       in.Reason,
		At:           time.Now().UTC(),
	}
	if err := uc.dispatcher.Dispatch(ctx, []events.DomainEvent{canonical}); err != nil {
		return nil, fmt.Errorf("AdjustInventory: dispatch: %w", err)
	}
	a.ClearPendingEvents()
	return &AdjustInventoryResult{Article: a}, nil
}
