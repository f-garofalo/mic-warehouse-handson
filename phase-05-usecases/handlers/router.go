package handlers

import (
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

	// Slice 1: GetArticle.
	e.GET("/articles/:id", r.articleHandler.GetArticle)

	// Slice 2: ChangeArticlePrice.
	e.PUT("/articles/:id/price", r.articleHandler.ChangeArticlePrice)
}
