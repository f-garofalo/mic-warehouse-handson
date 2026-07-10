package handlers

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"

	"warehouse.local/core/entities"
	"warehouse.local/core/middleware"
	"warehouse.local/core/repositories"
	"warehouse.local/core/usecases"
)

// ArticleHandler handles HTTP requests for the Article aggregate.
// Request/response shape exposes the BC's clean contract (price_cents + currency),
// not the legacy schema's decimal price. ADR-016 governs the deprecation
// policy for breaking changes.
type ArticleHandler struct {
	createArticleUC *usecases.CreateArticleUseCase
	getArticleUC    *usecases.GetArticleUseCase
	listArticlesUC  *usecases.ListArticlesUseCase
}

// CreateArticleRequest is the inbound shape for POST /articles.
type CreateArticleRequest struct {
	ID          string `json:"id"`
	SKU         string `json:"sku"`
	Name        string `json:"name"`
	Description string `json:"description"`
	PriceCents  int64  `json:"price_cents"`
	Currency    string `json:"currency"`
}

// ArticleResponse is the outbound shape for both POST and GET.
type ArticleResponse struct {
	ID          string           `json:"id"`
	SKU         string           `json:"sku"`
	Name        string           `json:"name"`
	Description string           `json:"description"`
	PriceCents  int64            `json:"price_cents"`
	Currency    string           `json:"currency"`
	Inventories []InventorySlice `json:"inventories,omitempty"`
}

// InventorySlice is the per-location stock view exposed in the article response.
type InventorySlice struct {
	LocationCode string `json:"location_code"`
	Quantity     int32  `json:"quantity"`
	Reserved     int32  `json:"reserved"`
}

func NewArticleHandler(createUC *usecases.CreateArticleUseCase, getUC *usecases.GetArticleUseCase, listUC *usecases.ListArticlesUseCase) *ArticleHandler {
	return &ArticleHandler{createArticleUC: createUC, getArticleUC: getUC, listArticlesUC: listUC}
}

func (h *ArticleHandler) CreateArticle(c echo.Context) error {
	// Authentication said who the caller is; the policy says whether THIS
	// caller may create articles. 403 is "I know you, and no".
	if !middleware.Can(middleware.AuthContextFrom(c.Request().Context()), middleware.ActionArticleCreate) {
		return c.JSON(http.StatusForbidden, map[string]string{
			"error": "forbidden",
			"hint":  "authenticated, but this principal is not allowed to create articles (see middleware/policy.go)",
		})
	}

	req := new(CreateArticleRequest)
	if err := c.Bind(req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}
	out, err := h.createArticleUC.Execute(c.Request().Context(), usecases.CreateArticleInput{
		ID:          req.ID,
		SKU:         req.SKU,
		Name:        req.Name,
		Description: req.Description,
		PriceCents:  req.PriceCents,
		Currency:    req.Currency,
	})
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusCreated, toArticleResponse(out.Article))
}

func (h *ArticleHandler) GetArticle(c echo.Context) error {
	id := c.Param("id")
	out, err := h.getArticleUC.Execute(c.Request().Context(), usecases.GetArticleInput{ID: id})
	if err != nil {
		// Use case wraps the repository's ErrArticleNotFound via fmt.Errorf("...%w", err);
		// errors.Is unwraps it. Both sentinels (usecases + repositories) are checked
		// because production wires MySQLArticleRepository (returns repositories sentinel)
		// while tests wire usecases.InMemoryArticleRepository (returns usecases sentinel).
		if errors.Is(err, usecases.ErrArticleNotFound) || errors.Is(err, repositories.ErrArticleNotFound) {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "article not found"})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, toArticleResponse(out.Article))
}

func (h *ArticleHandler) ListArticles(c echo.Context) error {
	out, err := h.listArticlesUC.Execute(c.Request().Context(), usecases.ListArticlesInput{})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	responses := make([]ArticleResponse, 0, len(out.Articles))
	for _, article := range out.Articles {
		responses = append(responses, toArticleResponse(article))
	}
	return c.JSON(http.StatusOK, responses)
}

func toArticleResponse(a *entities.Article) ArticleResponse {
	out := ArticleResponse{
		ID:          a.ID,
		SKU:         a.SKU.Code,
		Name:        a.Name,
		Description: a.Description,
		PriceCents:  a.Price.AmountCents,
		Currency:    a.Price.Currency,
	}
	for _, lvl := range a.Inventories {
		out.Inventories = append(out.Inventories, InventorySlice{
			LocationCode: lvl.LocationCode,
			Quantity:     lvl.Quantity,
			Reserved:     lvl.Reserved,
		})
	}
	return out
}
