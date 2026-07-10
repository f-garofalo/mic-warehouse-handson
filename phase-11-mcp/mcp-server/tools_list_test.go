package main

import (
	"context"
	"fmt"
	"net/http"
	"testing"
)

// ▸ Task 1 specification — these tests ship red.

func TestListArticles_noQueryReturnsEverything(t *testing.T) {
	handler := listArticlesHandler(newTestClient(t, readOnlyBC(t)))

	_, out, err := handler(context.Background(), nil, ListArticlesInput{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Count != 3 || len(out.Articles) != 3 {
		t.Errorf("got count=%d len=%d, want 3/3", out.Count, len(out.Articles))
	}
}

func TestListArticles_queryFiltersOnSKUAndNameCaseInsensitive(t *testing.T) {
	handler := listArticlesHandler(newTestClient(t, readOnlyBC(t)))

	// "m8" appears in two SKUs and two names, in different case.
	_, out, err := handler(context.Background(), nil, ListArticlesInput{Query: "m8"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Count != 2 {
		t.Errorf("query m8: got count=%d, want 2", out.Count)
	}

	// "wide" matches one article by name only (its SKU is SKU-WASHER).
	_, out, err = handler(context.Background(), nil, ListArticlesInput{Query: "WIDE"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Count != 1 || out.Articles[0].ID != "art-3" {
		t.Errorf("query WIDE: got %+v, want just art-3", out.Articles)
	}
}

func TestListArticles_limitCapsTheResult(t *testing.T) {
	handler := listArticlesHandler(newTestClient(t, readOnlyBC(t)))

	_, out, err := handler(context.Background(), nil, ListArticlesInput{Limit: 2})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Count != 2 || len(out.Articles) != 2 {
		t.Errorf("got count=%d len=%d, want 2/2", out.Count, len(out.Articles))
	}
}

func TestListArticles_defaultLimitIs20(t *testing.T) {
	// A warehouse with 30 articles: with no explicit limit the tool must
	// hand the agent 20, not the whole catalogue.
	big := make([]Article, 30)
	for i := range big {
		big[i] = Article{
			ID:       fmt.Sprintf("art-%03d", i),
			SKU:      fmt.Sprintf("SKU-%03d", i),
			Name:     fmt.Sprintf("Article %03d", i),
			Currency: "EUR",
		}
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /articles", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, http.StatusOK, big)
	})
	handler := listArticlesHandler(newTestClient(t, mux))

	_, out, err := handler(context.Background(), nil, ListArticlesInput{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Count != 20 || len(out.Articles) != 20 {
		t.Errorf("got count=%d len=%d, want the default cap of 20", out.Count, len(out.Articles))
	}
}
