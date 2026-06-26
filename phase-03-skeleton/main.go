package main

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

// main boots the Warehouse service. At CP3 it exposes only a health probe; the
// real /articles HTTP surface arrives in CP6. It imports no domain package, so
// the clean-architecture boundary (domain free of HTTP) stays intact.
func main() {
	e := echo.New()
	e.HideBanner = true
	e.GET("/health", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})
	e.Logger.Fatal(e.Start(":8081"))
}
