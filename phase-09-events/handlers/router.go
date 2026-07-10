package handlers

import (
	"github.com/labstack/echo/v4"

	"warehouse.local/core/middleware"
	"warehouse.local/core/usecases"
)

// Router wires handlers + auth + correlation. Public routes (/health) skip auth;
// every other route requires an AuthContext (user JWT or M2M token).
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

// Register installs middleware (auth + correlation) and routes on the Echo instance.
func (r *Router) Register(e *echo.Echo) {
	// AuthMiddlewareWithSkip honors SkipAuthPath ({/health, /metrics, /api/v1/auth}).
	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if err := middleware.AuthMiddlewareWithSkip(c); err != nil {
				return err
			}
			if c.Response().Committed {
				// AuthMiddleware writes a 401 body via c.JSON(...) and returns nil
				// (because c.JSON succeeded). Without this guard the chain would
				// continue into the business handler, which would append a second
				// body (e.g. "article not found") under the already-sent 401 status.
				return nil
			}
			return next(c)
		}
	})
	e.Use(middleware.CorrelationMiddleware)

	e.POST("/articles", r.articleHandler.CreateArticle)
	e.GET("/articles", r.articleHandler.ListArticles)
	e.GET("/articles/:id", r.articleHandler.GetArticle)
	e.POST("/articles/:article_id/inventory/adjust", r.inventoryHandler.AdjustInventory)
}
