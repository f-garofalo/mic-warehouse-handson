package usecases

import (
	"context"
	"fmt"

	"warehouse.local/core/entities"
	"warehouse.local/core/interfaces"
)

const (
	defaultListLimit = 50
	maxListLimit     = 200
)

// ListArticlesInput carries pagination. Zero/negative values are normalized:
// limit -> default (capped at max), offset -> 0.
type ListArticlesInput struct {
	Limit  int
	Offset int
}

// ListArticlesOutput is one page plus the total count for the meta envelope.
type ListArticlesOutput struct {
	Articles []*entities.Article
	Total    int
	Limit    int
	Offset   int
}

// ListArticlesUseCase returns a page of Articles. Pagination is applied at the
// application layer (fetch-all + slice); DB-side LIMIT/OFFSET would be a later
// optimization touching the repository port and every adapter.
type ListArticlesUseCase struct {
	repo interfaces.ArticleRepository
}

func NewListArticlesUseCase(repo interfaces.ArticleRepository) *ListArticlesUseCase {
	return &ListArticlesUseCase{repo: repo}
}

func (uc *ListArticlesUseCase) Execute(ctx context.Context, in ListArticlesInput) (*ListArticlesOutput, error) {
	all, err := uc.repo.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("ListArticles: %w", err)
	}

	limit := in.Limit
	if limit <= 0 {
		limit = defaultListLimit
	}
	if limit > maxListLimit {
		limit = maxListLimit
	}
	offset := in.Offset
	if offset < 0 {
		offset = 0
	}

	total := len(all)
	start := offset
	if start > total {
		start = total
	}
	end := start + limit
	if end > total {
		end = total
	}

	return &ListArticlesOutput{
		Articles: all[start:end],
		Total:    total,
		Limit:    limit,
		Offset:   offset,
	}, nil
}
