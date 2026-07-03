// Package handlers is the HTTP layer for Phase 05. It is a deliberately reduced
// version of the Phase 06 layer: no auth, no middleware beyond logger/recover,
// no list/pagination, no dual-write/MySQL, no error-sentinel juggling. A handler
// does exactly one job: translate HTTP <-> use case. Business logic stays in the
// use cases; rules stay in the aggregate.
package handlers

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"

	"warehouse.local/core/entities"
	"warehouse.local/core/usecases"
)

// ArticleHandler exposes the Article use cases over HTTP.
type ArticleHandler struct {
	createUC      *usecases.CreateArticleUseCase
	getUC         *usecases.GetArticleUseCase
	changePriceUC *usecases.ChangeArticlePriceUseCase
}

func NewArticleHandler(
	create *usecases.CreateArticleUseCase,
	get *usecases.GetArticleUseCase,
	changePrice *usecases.ChangeArticlePriceUseCase,
) *ArticleHandler {
	return &ArticleHandler{createUC: create, getUC: get, changePriceUC: changePrice}
}

// --- request / response DTOs (the BC's clean contract: price_cents + currency) ---

type CreateArticleRequest struct {
	ID          string `json:"id"`
	SKU         string `json:"sku"`
	Name        string `json:"name"`
	Description string `json:"description"`
	PriceCents  int64  `json:"price_cents"`
	Currency    string `json:"currency"`
}

type ChangePriceRequest struct {
	PriceCents int64  `json:"price_cents"`
	Currency   string `json:"currency"`
}

type ArticleResponse struct {
	ID          string `json:"id"`
	SKU         string `json:"sku"`
	Name        string `json:"name"`
	Description string `json:"description"`
	PriceCents  int64  `json:"price_cents"`
	Currency    string `json:"currency"`
}

// ============================================================================
// GIVEN — the worked example. POST /articles, wired end to end.
// Read this: it is the exact shape your two slices must follow.
// ============================================================================

func (h *ArticleHandler) CreateArticle(c echo.Context) error {
	req := new(CreateArticleRequest)
	if err := c.Bind(req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}
	out, err := h.createUC.Execute(c.Request().Context(), usecases.CreateArticleInput{
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

// ============================================================================
// YOUR TASK — Slice 1. GET /articles/:id
// ============================================================================

func (h *ArticleHandler) GetArticle(c echo.Context) error {
	out, err := h.getUC.Execute(c.Request().Context(), usecases.GetArticleInput{ID: c.Param("id")})
	if err != nil {
		if errors.Is(err, usecases.ErrArticleNotFound) {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "article not found"})
		}
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, toArticleResponse(out.Article))
}

// ============================================================================
// YOUR TASK — Slice 2. PUT /articles/:id/price   body: {price_cents, currency}
// ============================================================================

func (h *ArticleHandler) ChangeArticlePrice(c echo.Context) error {
	req := new(ChangePriceRequest)
	if err := c.Bind(req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}
	out, err := h.changePriceUC.Execute(c.Request().Context(), usecases.ChangeArticlePriceInput{
		ArticleID:     c.Param("id"),
		NewPriceCents: req.PriceCents,
		Currency:      req.Currency,
	})
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, toArticleResponse(out.Article))
}

// toArticleResponse maps the aggregate to the transport DTO. Mapping is the
// handler's job; the use case never sees JSON.
func toArticleResponse(a *entities.Article) ArticleResponse {
	return ArticleResponse{
		ID:          a.ID,
		SKU:         a.SKU.Code,
		Name:        a.Name,
		Description: a.Description,
		PriceCents:  a.Price.AmountCents,
		Currency:    a.Price.Currency,
	}
}
