package handlers

import (
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
)

// Deprecation returns middleware that stamps RFC 8594 sunset headers on every
// response of the routes it wraps:
//
//	Deprecation: true
//	Sunset: <HTTP-date>                       (when the endpoint will be removed)
//	Link: <successor>; rel="successor-version"
//
// This is how the Warehouse BC announces a breaking-changed endpoint to clients
// before removing it — a deprecation policy, not just a comment.
func Deprecation(sunset time.Time, successor string) echo.MiddlewareFunc {
	sunsetHeader := sunset.UTC().Format(http.TimeFormat) // RFC 1123 GMT, per RFC 8594
	link := "<" + successor + ">; rel=\"successor-version\""
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			h := c.Response().Header()
			h.Set("Deprecation", "true")
			h.Set("Sunset", sunsetHeader)
			h.Set("Link", link)
			return next(c)
		}
	}
}
