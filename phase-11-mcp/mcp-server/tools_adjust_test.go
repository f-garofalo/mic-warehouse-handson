package main

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

// ▸ Flex task specification — these tests ship red. Skip if behind schedule:
// nothing later depends on adjust_inventory.

func TestAdjustInventory_postsToTheArticlePathAndReturnsTheUpdate(t *testing.T) {
	var received map[string]any
	mux := http.NewServeMux()
	mux.HandleFunc("POST /articles/{id}/inventory/adjust", func(w http.ResponseWriter, r *http.Request) {
		if r.PathValue("id") != "art-1" {
			t.Errorf("adjust posted to article %q, want art-1", r.PathValue("id"))
		}
		if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
			t.Fatalf("decode adjust body: %v", err)
		}
		writeJSON(t, w, http.StatusOK, Article{
			ID: "art-1", SKU: "SKU-BOLT-M8", Name: "Bolt M8", Currency: "EUR",
			Inventories: []InventoryLevel{{LocationCode: "MAIN", Quantity: 45, Reserved: 0}},
		})
	})
	handler := adjustInventoryHandler(newTestClient(t, mux))

	_, out, err := handler(context.Background(), nil, AdjustInventoryInput{
		ArticleID:    "art-1",
		LocationCode: "MAIN",
		Delta:        -5,
		Reason:       "damaged goods",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if received["location_code"] != "MAIN" || received["delta"] != float64(-5) || received["reason"] != "damaged goods" {
		t.Errorf("BC received %v, want location_code/delta/reason as sent", received)
	}
	if len(out.Article.Inventories) != 1 || out.Article.Inventories[0].Quantity != 45 {
		t.Errorf("got %+v, want the updated stock level back", out.Article)
	}
}

func TestAdjustInventory_missingFieldsFailWithoutCallingTheBC(t *testing.T) {
	bcCalled := false
	handler := adjustInventoryHandler(newTestClient(t, spyBC(func() { bcCalled = true })))

	for _, in := range []AdjustInventoryInput{
		{LocationCode: "MAIN", Delta: 1},
		{ArticleID: "art-1", Delta: 1},
	} {
		if _, _, err := handler(context.Background(), nil, in); err == nil {
			t.Errorf("input %+v: want a validation error", in)
		}
	}
	if bcCalled {
		t.Error("the BC was called with an incomplete adjustment")
	}
}

func TestAdjustInventory_unknownArticleGuidesTheAgent(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /articles/{id}/inventory/adjust", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, http.StatusNotFound, map[string]string{"error": "article not found"})
	})
	handler := adjustInventoryHandler(newTestClient(t, mux))

	_, _, err := handler(context.Background(), nil, AdjustInventoryInput{
		ArticleID: "art-999", LocationCode: "MAIN", Delta: 1, Reason: "recount",
	})
	if err == nil {
		t.Fatal("want an error for an unknown article")
	}
	if !strings.Contains(err.Error(), "art-999") || !strings.Contains(err.Error(), "list_articles") {
		t.Errorf("error %q should name the id and suggest list_articles", err.Error())
	}
}
