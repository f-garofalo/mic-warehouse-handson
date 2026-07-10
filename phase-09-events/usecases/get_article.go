package usecases

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"warehouse.local/core/entities"
	"warehouse.local/core/interfaces"
)

// GetArticleInput identifies the aggregate to load.
type GetArticleInput struct {
	ID string
}

// GetArticleOutput returns the aggregate. Mapping to a transport DTO is the
// HTTP layer's responsibility (phase 06).
type GetArticleOutput struct {
	Article *entities.Article
}

// GetArticleUseCase loads an Article by id.
type GetArticleUseCase struct {
	repo interfaces.ArticleRepository
}

func NewGetArticleUseCase(repo interfaces.ArticleRepository) *GetArticleUseCase {
	return &GetArticleUseCase{repo: repo}
}

func (uc *GetArticleUseCase) Execute(ctx context.Context, in GetArticleInput) (*GetArticleOutput, error) {
	if strings.TrimSpace(in.ID) == "" {
		return nil, errors.New("GetArticle: ID is required")
	}
	a, err := uc.repo.FindByID(ctx, in.ID)
	if err != nil {
		return nil, fmt.Errorf("GetArticle: %w", err)
	}
	return &GetArticleOutput{Article: a}, nil
}
