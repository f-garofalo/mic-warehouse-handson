package main

import (
	"log"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"warehouse.local/core/dispatcher"
	"warehouse.local/core/handlers"
	"warehouse.local/core/usecases"
)

func main() {
	// Composition root. Phase 05 runs on in-memory adapters: the point is the
	// application + HTTP layers, not persistence (that was Phase 04). One shared
	// repository instance keeps data alive across requests for the whole run.
	repo := usecases.NewInMemoryArticleRepository()
	disp := dispatcher.NewInMemoryDispatcher()

	createUC := usecases.NewCreateArticleUseCase(repo, disp)
	getUC := usecases.NewGetArticleUseCase(repo)
	changePriceUC := usecases.NewChangeArticlePriceUseCase(repo, disp)

	articleHandler := handlers.NewArticleHandler(createUC, getUC, changePriceUC)
	router := handlers.NewRouter(articleHandler)

	e := echo.New()
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.GET("/health", func(c echo.Context) error {
		return c.JSON(200, map[string]string{"status": "ok"})
	})
	router.Register(e)
	handlers.RegisterDocs(e) // Swagger UI at /docs — test the API from the browser

	log.Printf("Starting server on :8081 (API explorer at http://localhost:8081/docs)")
	e.Logger.Fatal(e.Start(":8081"))
}
