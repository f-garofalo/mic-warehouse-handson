package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"

	"warehouse.local/core/dispatcher"
	"warehouse.local/core/usecases"
)

func setupInventoryHandler() (*ArticleHandler, *InventoryHandler, *usecases.InMemoryArticleRepository, *dispatcher.InMemoryDispatcher) {
	repo := usecases.NewInMemoryArticleRepository()
	disp := dispatcher.NewInMemoryDispatcher()
	createUC := usecases.NewCreateArticleUseCase(repo, disp)
	getUC := usecases.NewGetArticleUseCase(repo)
	listUC := usecases.NewListArticlesUseCase(repo)
	adjustUC := usecases.NewAdjustInventoryUseCase(repo, disp)
	return NewArticleHandler(createUC, getUC, listUC), NewInventoryHandler(adjustUC), repo, disp
}

func TestInventoryHandler_Adjust_appliesDelta(t *testing.T) {
	ah, ih, _, disp := setupInventoryHandler()

	// Seed an article via the article handler.
	body := `{"id":"id-1","sku":"ABC-001","name":"Widget","price_cents":1000,"currency":"EUR"}`
	req := httptest.NewRequest(http.MethodPost, "/articles", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	_ = ah.CreateArticle(echo.New().NewContext(asPrincipal(req, testAlice), rec))
	disp.Reset()

	// Adjust.
	adjBody := `{"location_code":"IT-MILANO1","delta":15,"reason":"restock"}`
	req2 := httptest.NewRequest(http.MethodPost, "/articles/id-1/inventory/adjust", strings.NewReader(adjBody))
	req2.Header.Set("Content-Type", "application/json")
	rec2 := httptest.NewRecorder()
	c := echo.New().NewContext(req2, rec2)
	c.SetParamNames("article_id")
	c.SetParamValues("id-1")
	if err := ih.AdjustInventory(c); err != nil {
		t.Fatalf("AdjustInventory: %v", err)
	}
	if rec2.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body=%s)", rec2.Code, rec2.Body.String())
	}

	var resp map[string]any
	_ = json.NewDecoder(rec2.Body).Decode(&resp)
	invs, ok := resp["inventories"].([]any)
	if !ok || len(invs) != 1 {
		t.Fatalf("expected 1 inventory level in response, got %v", resp["inventories"])
	}
	if got := disp.Events(); len(got) != 1 {
		t.Errorf("expected 1 dispatched event, got %d", len(got))
	}
}

func TestInventoryHandler_Adjust_returns404OnMissingArticle(t *testing.T) {
	_, ih, _, _ := setupInventoryHandler()

	body := `{"location_code":"L1","delta":5}`
	req := httptest.NewRequest(http.MethodPost, "/articles/missing/inventory/adjust", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := echo.New().NewContext(req, rec)
	c.SetParamNames("article_id")
	c.SetParamValues("missing")
	_ = ih.AdjustInventory(c)
	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404 for missing article, got %d (body=%s)", rec.Code, rec.Body.String())
	}
}

func TestInventoryHandler_Adjust_returns400OnEmptyLocation(t *testing.T) {
	ah, ih, _, _ := setupInventoryHandler()

	// Seed.
	body := `{"id":"id-1","sku":"ABC-001","name":"X","price_cents":100,"currency":"EUR"}`
	r1 := httptest.NewRequest(http.MethodPost, "/articles", strings.NewReader(body))
	r1.Header.Set("Content-Type", "application/json")
	rec1 := httptest.NewRecorder()
	_ = ah.CreateArticle(echo.New().NewContext(asPrincipal(r1, testAlice), rec1))

	// Adjust with empty location.
	adj := `{"location_code":"","delta":5}`
	r2 := httptest.NewRequest(http.MethodPost, "/articles/id-1/inventory/adjust", strings.NewReader(adj))
	r2.Header.Set("Content-Type", "application/json")
	rec2 := httptest.NewRecorder()
	c := echo.New().NewContext(r2, rec2)
	c.SetParamNames("article_id")
	c.SetParamValues("id-1")
	_ = ih.AdjustInventory(c)
	if rec2.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec2.Code)
	}
}
