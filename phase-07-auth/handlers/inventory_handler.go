package handlers

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"

	"warehouse.local/core/repositories"
	"warehouse.local/core/usecases"
)

// InventoryHandler exposes inventory adjustment via the Article aggregate.
type InventoryHandler struct {
	adjustInventoryUC *usecases.AdjustInventoryUseCase
}

// AdjustInventoryRequest is the inbound shape for POST /articles/:article_id/inventory/adjust.
type AdjustInventoryRequest struct {
	LocationCode string `json:"location_code"`
	Delta        int32  `json:"delta"`
	Reason       string `json:"reason"`
}

func NewInventoryHandler(uc *usecases.AdjustInventoryUseCase) *InventoryHandler {
	return &InventoryHandler{adjustInventoryUC: uc}
}

func (h *InventoryHandler) AdjustInventory(c echo.Context) error {
	articleID := c.Param("article_id")
	req := new(AdjustInventoryRequest)
	if err := c.Bind(req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}
	out, err := h.adjustInventoryUC.Execute(c.Request().Context(), usecases.AdjustInventoryInput{
		ArticleID:    articleID,
		LocationCode: req.LocationCode,
		Delta:        req.Delta,
		Reason:       req.Reason,
	})
	if err != nil {
		// Both sentinels checked: usecases.InMemoryArticleRepository (tests) returns
		// usecases.ErrArticleNotFound; repositories.MySQLArticleRepository (production)
		// returns repositories.ErrArticleNotFound. Use case wraps via %w.
		if errors.Is(err, usecases.ErrArticleNotFound) || errors.Is(err, repositories.ErrArticleNotFound) {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "article not found"})
		}
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, toArticleResponse(out.Article))
}
