package handlers

import (
	"time"

	"github.com/labstack/echo/v4"
)

// Router registers the Article routes on the Echo instance.
type Router struct {
	articleHandler *ArticleHandler
}

func NewRouter(h *ArticleHandler) *Router {
	return &Router{articleHandler: h}
}

func (r *Router) Register(e *echo.Echo) {
	// GIVEN — the worked-example route.
	e.POST("/articles", r.articleHandler.CreateArticle)

	// Phase 06: paginated list.
	e.GET("/articles", r.articleHandler.ListArticles)

	// Slice 1: GetArticle.
	e.GET("/articles/:id", r.articleHandler.GetArticle)

	// Slice 2: ChangeArticlePrice.
	e.PUT("/articles/:id/price", r.articleHandler.ChangeArticlePrice)

	// Phase 06: deprecation policy (RFC 8594). A deprecated alias of
	// GET /articles/:id that announces its sunset and points to the successor.
	sunset := time.Date(2026, time.December, 31, 0, 0, 0, 0, time.UTC)
	v0 := e.Group("/v0", Deprecation(sunset, "/articles/:id"))
	v0.GET("/articles/:id", r.articleHandler.GetArticle)
}
