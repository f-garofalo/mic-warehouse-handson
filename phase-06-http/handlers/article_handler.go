// Package handlers is the HTTP layer. In Phase 06 it grows list + pagination, is
// wired to the real dual-write repository, and matches the unified not-found
// sentinel; auth stays out (that is CP7). A handler does exactly one job:
// translate HTTP <-> use case. Business logic stays in the use cases; rules in
// the aggregate.
package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"

	"warehouse.local/core/entities"
	"warehouse.local/core/usecases"
)

// ArticleHandler exposes the Article use cases over HTTP.
type ArticleHandler struct {
	createUC      *usecases.CreateArticleUseCase
	getUC         *usecases.GetArticleUseCase
	changePriceUC *usecases.ChangeArticlePriceUseCase
	listUC        *usecases.ListArticlesUseCase
}

func NewArticleHandler(
	create *usecases.CreateArticleUseCase,
	get *usecases.GetArticleUseCase,
	changePrice *usecases.ChangeArticlePriceUseCase,
	list *usecases.ListArticlesUseCase,
) *ArticleHandler {
	return &ArticleHandler{createUC: create, getUC: get, changePriceUC: changePrice, listUC: list}
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

// ListResponse mirrors the MIC monolith's list envelope: { data, meta }.
type ListResponse struct {
	Data []ArticleResponse `json:"data"`
	Meta ListMeta          `json:"meta"`
}

type ListMeta struct {
	Total  int `json:"total"`
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
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

// ============================================================================
// Phase 06 — List. GET /articles?limit=&offset=  ->  { data, meta }
// ============================================================================

func (h *ArticleHandler) ListArticles(c echo.Context) error {
	limit, _ := strconv.Atoi(c.QueryParam("limit"))
	offset, _ := strconv.Atoi(c.QueryParam("offset"))
	out, err := h.listUC.Execute(c.Request().Context(), usecases.ListArticlesInput{Limit: limit, Offset: offset})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	items := make([]ArticleResponse, 0, len(out.Articles))
	for _, a := range out.Articles {
		items = append(items, toArticleResponse(a))
	}
	return c.JSON(http.StatusOK, ListResponse{
		Data: items,
		Meta: ListMeta{Total: out.Total, Limit: out.Limit, Offset: out.Offset},
	})
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
