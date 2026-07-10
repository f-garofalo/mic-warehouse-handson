package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"

	"warehouse.local/core/dispatcher"
	"warehouse.local/core/middleware"
	"warehouse.local/core/usecases"
)

func setupArticleHandler() (*ArticleHandler, *usecases.InMemoryArticleRepository, *dispatcher.InMemoryDispatcher) {
	repo := usecases.NewInMemoryArticleRepository()
	disp := dispatcher.NewInMemoryDispatcher()
	createUC := usecases.NewCreateArticleUseCase(repo, disp)
	getUC := usecases.NewGetArticleUseCase(repo)
	listUC := usecases.NewListArticlesUseCase(repo)
	return NewArticleHandler(createUC, getUC, listUC), repo, disp
}


// asPrincipal attaches an authenticated identity to the request, the way the
// auth middleware would after validating a token.
func asPrincipal(req *http.Request, ac *middleware.AuthContext) *http.Request {
	return req.WithContext(middleware.WithAuthContext(req.Context(), ac))
}

var testAlice = &middleware.AuthContext{Kind: middleware.AuthKindUser, UserID: "u-alice", Email: "alice@example.com"}
var testBob = &middleware.AuthContext{Kind: middleware.AuthKindUser, UserID: "u-bob", Email: "bob@example.com"}

func TestArticleHandler_Create_returns201WithAggregate(t *testing.T) {
	h, _, disp := setupArticleHandler()

	body := `{"id":"id-1","sku":"ABC-001","name":"Widget","description":"a","price_cents":2999,"currency":"EUR"}`
	req := httptest.NewRequest(http.MethodPost, "/articles", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := echo.New().NewContext(asPrincipal(req, testAlice), rec)

	if err := h.CreateArticle(c); err != nil {
		t.Fatalf("CreateArticle: %v", err)
	}
	if rec.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d (body=%s)", rec.Code, rec.Body.String())
	}

	var resp map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp["id"] != "id-1" {
		t.Errorf("expected id=id-1, got %v", resp["id"])
	}
	if resp["price_cents"] != float64(2999) {
		t.Errorf("expected price_cents=2999, got %v", resp["price_cents"])
	}
	if resp["currency"] != "EUR" {
		t.Errorf("expected currency=EUR, got %v", resp["currency"])
	}
	if got := disp.Events(); len(got) != 1 {
		t.Errorf("expected 1 dispatched event, got %d", len(got))
	}
}

func TestArticleHandler_Create_returns400OnValidationError(t *testing.T) {
	h, _, _ := setupArticleHandler()

	body := `{"id":"id-1","sku":"bad","name":"X","price_cents":2999,"currency":"EUR"}` // SKU lowercase
	req := httptest.NewRequest(http.MethodPost, "/articles", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := echo.New().NewContext(asPrincipal(req, testAlice), rec)

	_ = h.CreateArticle(c)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestArticleHandler_Create_returns400OnMalformedJSON(t *testing.T) {
	h, _, _ := setupArticleHandler()

	req := httptest.NewRequest(http.MethodPost, "/articles", bytes.NewReader([]byte("{not json")))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := echo.New().NewContext(asPrincipal(req, testAlice), rec)

	_ = h.CreateArticle(c)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestArticleHandler_Get_returns200(t *testing.T) {
	h, _, _ := setupArticleHandler()

	// Seed via Create.
	body := `{"id":"id-1","sku":"ABC-001","name":"Widget","price_cents":2999,"currency":"EUR"}`
	createReq := httptest.NewRequest(http.MethodPost, "/articles", strings.NewReader(body))
	createReq.Header.Set("Content-Type", "application/json")
	createRec := httptest.NewRecorder()
	_ = h.CreateArticle(echo.New().NewContext(asPrincipal(createReq, testAlice), createRec))

	// Fetch.
	req := httptest.NewRequest(http.MethodGet, "/articles/id-1", nil)
	rec := httptest.NewRecorder()
	c := echo.New().NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("id-1")
	if err := h.GetArticle(c); err != nil {
		t.Fatalf("GetArticle: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body=%s)", rec.Code, rec.Body.String())
	}

	var resp map[string]any
	_ = json.NewDecoder(rec.Body).Decode(&resp)
	if resp["sku"] != "ABC-001" {
		t.Errorf("expected sku=ABC-001, got %v", resp["sku"])
	}
}

func TestArticleHandler_Get_returns404(t *testing.T) {
	h, _, _ := setupArticleHandler()

	req := httptest.NewRequest(http.MethodGet, "/articles/missing", nil)
	rec := httptest.NewRecorder()
	c := echo.New().NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("missing")
	_ = h.GetArticle(c)
	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}
}

func TestArticleHandler_List_returns200WithArticles(t *testing.T) {
	h, _, _ := setupArticleHandler()

	for _, body := range []string{
		`{"id":"id-1","sku":"ABC-001","name":"Widget A","price_cents":1000,"currency":"EUR"}`,
		`{"id":"id-2","sku":"ABC-002","name":"Widget B","price_cents":1500,"currency":"EUR"}`,
	} {
		req := httptest.NewRequest(http.MethodPost, "/articles", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		if err := h.CreateArticle(echo.New().NewContext(asPrincipal(req, testAlice), rec)); err != nil {
			t.Fatalf("CreateArticle seed: %v", err)
		}
		if rec.Code != http.StatusCreated {
			t.Fatalf("seed expected 201, got %d (body=%s)", rec.Code, rec.Body.String())
		}
	}

	req := httptest.NewRequest(http.MethodGet, "/articles", nil)
	rec := httptest.NewRecorder()
	c := echo.New().NewContext(req, rec)
	if err := h.ListArticles(c); err != nil {
		t.Fatalf("ListArticles: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body=%s)", rec.Code, rec.Body.String())
	}

	var resp []map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(resp) != 2 {
		t.Fatalf("expected 2 articles, got %d", len(resp))
	}
	if resp[0]["price_cents"] == nil || resp[0]["currency"] != "EUR" {
		t.Fatalf("expected clean BC article shape, got %+v", resp[0])
	}
}

func TestArticleHandler_Create_returns403WhenPolicyDenies(t *testing.T) {
	h, _, disp := setupArticleHandler()

	body := `{"sku":"ABC-009","name":"Widget","price_cents":2999,"currency":"EUR"}`
	req := httptest.NewRequest(http.MethodPost, "/articles", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := echo.New().NewContext(asPrincipal(req, testBob), rec)

	_ = h.CreateArticle(c)
	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403 for Bob, got %d (body=%s)", rec.Code, rec.Body.String())
	}
	if got := disp.Events(); len(got) != 0 {
		t.Errorf("a denied create must not dispatch events, got %d", len(got))
	}
}
