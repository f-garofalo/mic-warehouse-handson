package handlers

import (
	"github.com/labstack/echo/v4"

	"warehouse.local/core/usecases"
)

// Router wires handlers to Echo routes. The legacy `PUT /articles/:id/inventory`
// path is replaced by `POST /articles/:article_id/inventory/adjust` matching
// the aggregate's AdjustInventory semantics.
type Router struct {
	articleHandler   *ArticleHandler
	inventoryHandler *InventoryHandler
}

func NewRouter(
	createUC *usecases.CreateArticleUseCase,
	getUC *usecases.GetArticleUseCase,
	listUC *usecases.ListArticlesUseCase,
	adjustUC *usecases.AdjustInventoryUseCase,
) *Router {
	return &Router{
		articleHandler:   NewArticleHandler(createUC, getUC, listUC),
		inventoryHandler: NewInventoryHandler(adjustUC),
	}
}

func (r *Router) Register(e *echo.Echo) {
	e.POST("/articles", r.articleHandler.CreateArticle)
	e.GET("/articles", r.articleHandler.ListArticles)
	e.GET("/articles/:id", r.articleHandler.GetArticle)
	e.POST("/articles/:article_id/inventory/adjust", r.inventoryHandler.AdjustInventory)
}
