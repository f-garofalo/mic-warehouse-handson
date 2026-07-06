package usecases

import (
	"context"
	"fmt"

	"warehouse.local/core/entities"
	"warehouse.local/core/interfaces"
)

// ListArticlesInput is intentionally empty for now. Future phases can add
// pagination or filters without changing the handler shape introduced here.
type ListArticlesInput struct{}

// ListArticlesOutput returns the aggregates loaded from the repository.
type ListArticlesOutput struct {
	Articles []*entities.Article
}

// ListArticlesUseCase loads Article aggregates for the teaching GET /articles
// endpoint. The route is intentionally small and not the final platform list
// contract.
type ListArticlesUseCase struct {
	repo interfaces.ArticleRepository
}

func NewListArticlesUseCase(repo interfaces.ArticleRepository) *ListArticlesUseCase {
	return &ListArticlesUseCase{repo: repo}
}

func (uc *ListArticlesUseCase) Execute(ctx context.Context, in ListArticlesInput) (*ListArticlesOutput, error) {
	articles, err := uc.repo.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("ListArticles: list: %w", err)
	}
	return &ListArticlesOutput{Articles: articles}, nil
}
