package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// The suite runs the tool handlers against a fake BC (an httptest server)
// and a fake IAM issuing "test-token". No Docker stack needed: run it with
//
//	docker compose run --build --rm test

// newTestClient wires a BCClient against the given fake BC handler. Every
// request to the fake BC must carry the M2M token, exactly like the real BC
// would demand: a handler that forgets the client is authenticated fails
// here first.
func newTestClient(t *testing.T, bcHandler http.Handler) *BCClient {
	t.Helper()

	iam := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/oauth/token" {
			http.NotFound(w, r)
			return
		}
		if err := r.ParseForm(); err != nil || r.PostForm.Get("grant_type") != "client_credentials" {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error":"unsupported_grant_type"}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"access_token": "test-token",
			"expires_in":   3600,
			"token_type":   "Bearer",
		})
	}))
	t.Cleanup(iam.Close)

	authenticated := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-token" {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error":"warehouse_bc_auth_required"}`))
			return
		}
		bcHandler.ServeHTTP(w, r)
	})
	bcSrv := httptest.NewServer(authenticated)
	t.Cleanup(bcSrv.Close)

	return NewBCClient(bcSrv.URL, iam.URL+"/oauth/token", "svc-warehouse-agent", "demo")
}

// testArticles is the fixed warehouse the read tests run against.
var testArticles = []Article{
	{ID: "art-1", SKU: "SKU-BOLT-M8", Name: "Bolt M8", Description: "Hex bolt, zinc plated", PriceCents: 250, Currency: "EUR"},
	{ID: "art-2", SKU: "SKU-NUT-M8", Name: "Nut M8", Description: "Hex nut", PriceCents: 120, Currency: "EUR"},
	{ID: "art-3", SKU: "SKU-WASHER", Name: "Wide washer", Description: "Flat washer, large series", PriceCents: 80, Currency: "EUR"},
}

// readOnlyBC fakes GET /articles and GET /articles/{id} over testArticles.
func readOnlyBC(t *testing.T) http.Handler {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("GET /articles", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, http.StatusOK, testArticles)
	})
	mux.HandleFunc("GET /articles/{id}", func(w http.ResponseWriter, r *http.Request) {
		for _, a := range testArticles {
			if a.ID == r.PathValue("id") {
				writeJSON(t, w, http.StatusOK, a)
				return
			}
		}
		writeJSON(t, w, http.StatusNotFound, map[string]string{"error": "article not found"})
	})
	return mux
}

// spyBC records that the BC was reached and answers 500: used to prove a
// handler rejected an invalid input WITHOUT calling the BC.
func spyBC(onCall func()) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		onCall()
		w.WriteHeader(http.StatusInternalServerError)
	})
}

func writeJSON(t *testing.T, w http.ResponseWriter, status int, v any) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		t.Fatalf("encode fake BC response: %v", err)
	}
}
