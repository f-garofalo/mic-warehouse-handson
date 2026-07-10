package main

import (
	"context"
	"strings"
	"testing"
)

// get_article is the given worked example: this file is green from the
// start. Read it together with tools_get.go to see what the specification
// of a tool looks like before writing your own.

func TestGetArticle_returnsTheArticle(t *testing.T) {
	handler := getArticleHandler(newTestClient(t, readOnlyBC(t)))

	_, out, err := handler(context.Background(), nil, GetArticleInput{ID: "art-1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Article.ID != "art-1" || out.Article.SKU != "SKU-BOLT-M8" {
		t.Errorf("got %+v, want art-1 / SKU-BOLT-M8", out.Article)
	}
}

func TestGetArticle_missingIDFailsWithoutCallingTheBC(t *testing.T) {
	bcCalled := false
	handler := getArticleHandler(newTestClient(t, spyBC(func() { bcCalled = true })))

	_, _, err := handler(context.Background(), nil, GetArticleInput{ID: "  "})
	if err == nil {
		t.Fatal("want an error for a blank id")
	}
	if bcCalled {
		t.Error("the BC was called for an obviously invalid input")
	}
}

func TestGetArticle_notFoundErrorGuidesTheAgent(t *testing.T) {
	handler := getArticleHandler(newTestClient(t, readOnlyBC(t)))

	_, _, err := handler(context.Background(), nil, GetArticleInput{ID: "art-999"})
	if err == nil {
		t.Fatal("want an error for an unknown id")
	}
	// The error is read by a model: it must carry the failing id and point
	// at the tool that can recover the situation.
	if !strings.Contains(err.Error(), "art-999") || !strings.Contains(err.Error(), "list_articles") {
		t.Errorf("error %q should name the id and suggest list_articles", err.Error())
	}
}
