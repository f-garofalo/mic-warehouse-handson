package handlers

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

// RegisterDocs serves an interactive API explorer (Swagger UI, like FastAPI's
// /docs) at GET /docs, backed by the OpenAPI spec at GET /openapi.yaml.
//
// This is the black-box way to test the API: open http://localhost:8081/docs in
// a browser, pick an endpoint, click "Try it out", send a request, read the
// response. No curl, no Go knowledge needed. (Swagger UI assets load from a CDN,
// so the browser needs internet the first time.)
func RegisterDocs(e *echo.Echo) {
	e.GET("/openapi.yaml", func(c echo.Context) error {
		return c.Blob(http.StatusOK, "application/yaml", []byte(openapiYAML))
	})
	e.GET("/docs", func(c echo.Context) error {
		return c.HTML(http.StatusOK, swaggerUIHTML)
	})
}

const swaggerUIHTML = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <title>Warehouse BC API</title>
  <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css">
</head>
<body>
  <div id="swagger-ui"></div>
  <script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
  <script>
    window.onload = function () {
      SwaggerUIBundle({ url: '/openapi.yaml', dom_id: '#swagger-ui' });
    };
  </script>
</body>
</html>`

const openapiYAML = `openapi: 3.0.3
info:
  title: Warehouse BC (Phase 05)
  version: 0.1.0
  description: >
    In-memory Warehouse API. POST is given; GET and PUT come alive as you build
    the two slices. Use "Try it out" on each endpoint to test it black-box.
paths:
  /articles:
    post:
      summary: Create an article (GIVEN)
      requestBody:
        required: true
        content:
          application/json:
            schema: { $ref: '#/components/schemas/CreateArticle' }
      responses:
        '201': { description: created, content: { application/json: { schema: { $ref: '#/components/schemas/Article' } } } }
        '400': { description: bad request }
  /articles/{id}:
    get:
      summary: Get an article by id (SLICE 1 - you build this)
      parameters:
        - { name: id, in: path, required: true, schema: { type: string } }
      responses:
        '200': { description: ok, content: { application/json: { schema: { $ref: '#/components/schemas/Article' } } } }
        '404': { description: not found }
  /articles/{id}/price:
    put:
      summary: Change an article's price (SLICE 2 - you build this)
      parameters:
        - { name: id, in: path, required: true, schema: { type: string } }
      requestBody:
        required: true
        content:
          application/json:
            schema: { $ref: '#/components/schemas/ChangePrice' }
      responses:
        '200': { description: ok, content: { application/json: { schema: { $ref: '#/components/schemas/Article' } } } }
        '400': { description: bad request }
components:
  schemas:
    CreateArticle:
      type: object
      properties:
        id: { type: string }
        sku: { type: string }
        name: { type: string }
        description: { type: string }
        price_cents: { type: integer, format: int64 }
        currency: { type: string }
      example: { id: a1, sku: ABC-001, name: Widget, description: "", price_cents: 1000, currency: EUR }
    ChangePrice:
      type: object
      properties:
        price_cents: { type: integer, format: int64 }
        currency: { type: string }
      example: { price_cents: 1500, currency: EUR }
    Article:
      type: object
      properties:
        id: { type: string }
        sku: { type: string }
        name: { type: string }
        description: { type: string }
        price_cents: { type: integer, format: int64 }
        currency: { type: string }
`
