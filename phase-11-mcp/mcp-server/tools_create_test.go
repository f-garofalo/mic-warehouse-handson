package main

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

// ▸ Task 2 specification — these tests ship red.

func TestCreateArticle_postsTheBCShapeAndReturnsTheMintedArticle(t *testing.T) {
	var received map[string]any
	mux := http.NewServeMux()
	mux.HandleFunc("POST /articles", func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
			t.Fatalf("decode create body: %v", err)
		}
		writeJSON(t, w, http.StatusCreated, Article{
			ID:         "art-minted-1",
			SKU:        received["sku"].(string),
			Name:       received["name"].(string),
			PriceCents: 1299,
			Currency:   received["currency"].(string),
		})
	})
	handler := createArticleHandler(newTestClient(t, mux))

	_, out, err := handler(context.Background(), nil, CreateArticleInput{
		SKU:        "SKU-NEW-01",
		Name:       "Agent-created article",
		PriceCents: 1299,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if received["sku"] != "SKU-NEW-01" || received["name"] != "Agent-created article" {
		t.Errorf("BC received %v, want the input sku and name", received)
	}
	if received["currency"] != "EUR" {
		t.Errorf("currency sent %v, want the default EUR when omitted", received["currency"])
	}
	if _, hasID := received["id"]; hasID && received["id"] != "" {
		t.Errorf("the request carries id %v: the BC mints the id, the tool must not", received["id"])
	}
	if out.Article.ID != "art-minted-1" {
		t.Errorf("got article %+v, want the BC-minted id art-minted-1", out.Article)
	}
}

func TestCreateArticle_missingFieldsFailWithoutCallingTheBC(t *testing.T) {
	bcCalled := false
	handler := createArticleHandler(newTestClient(t, spyBC(func() { bcCalled = true })))

	for _, in := range []CreateArticleInput{
		{Name: "No SKU", PriceCents: 100},
		{SKU: "SKU-X", PriceCents: 100},
	} {
		if _, _, err := handler(context.Background(), nil, in); err == nil {
			t.Errorf("input %+v: want a validation error", in)
		}
	}
	if bcCalled {
		t.Error("the BC was called with an incomplete article")
	}
}

func TestCreateArticle_forbiddenErrorNamesThePolicy(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /articles", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, http.StatusForbidden, map[string]string{
			"error": "forbidden",
			"hint":  "authenticated, but this principal is not allowed to create articles",
		})
	})
	handler := createArticleHandler(newTestClient(t, mux))

	_, _, err := handler(context.Background(), nil, CreateArticleInput{
		SKU: "SKU-DENIED", Name: "Denied", PriceCents: 100,
	})
	if err == nil {
		t.Fatal("want an error when the BC answers 403")
	}
	// 403 is a decision, not a malfunction: the agent should explain it,
	// not retry it. The error must say the policy denied the action.
	if !strings.Contains(strings.ToLower(err.Error()), "policy") {
		t.Errorf("error %q should name the policy as the reason", err.Error())
	}
}
