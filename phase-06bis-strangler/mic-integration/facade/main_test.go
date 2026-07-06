package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// The table below is the Part 1 specification of the routing decision.
// Every row here stays true for the whole phase: Part 3 only changes what
// happens to the create (see TestDecideUpstream_part3CreateCutover).
func TestDecideUpstreamPart1(t *testing.T) {
	tests := []struct {
		name   string
		method string
		mode   string
		want   string
	}{
		{"legacy mode keeps the list read on the monolith", http.MethodGet, modeLegacy, upstreamMonolith},
		{"warehouse-bc mode moves the list read to the BC", http.MethodGet, modeWarehouseBC, upstreamWarehouseBC},
		{"an unknown mode fails safe to the monolith", http.MethodGet, "typo-mode", upstreamMonolith},
		{"the update is not migrated", http.MethodPut, modeWarehouseBC, upstreamMonolith},
		{"the delete is not migrated", http.MethodDelete, modeWarehouseBC, upstreamMonolith},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := decideUpstream(tt.method, tt.mode); got != tt.want {
				t.Errorf("decideUpstream(%s, %s) = %q, want %q", tt.method, tt.mode, got, tt.want)
			}
		})
	}
}

// Part 3: the article CREATE migrates too. Remove the t.Skip line when you
// take on Part 3, then update decideUpstream until this is green.
func TestDecideUpstream_part3CreateCutover(t *testing.T) {
	t.Skip("Part 3: remove this skip when you migrate the create")

	tests := []struct {
		name   string
		method string
		mode   string
		want   string
	}{
		{"warehouse-bc mode moves the create to the BC", http.MethodPost, modeWarehouseBC, upstreamWarehouseBC},
		{"the facade dial pulls the create back too (rollback covers writes)", http.MethodPost, modeLegacy, upstreamMonolith},
		{"the update is still not migrated", http.MethodPut, modeWarehouseBC, upstreamMonolith},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := decideUpstream(tt.method, tt.mode); got != tt.want {
				t.Errorf("decideUpstream(%s, %s) = %q, want %q", tt.method, tt.mode, got, tt.want)
			}
		})
	}
}

// End-to-end through the facade: the /api/articles route obeys the mode and
// stamps X-Strangler-Route; item routes and other MIC paths always reach the
// monolith.
func TestArticleRouteHeaderAndScope(t *testing.T) {
	monolith := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("monolith"))
	}))
	defer monolith.Close()
	adapter := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("adapter"))
	}))
	defer adapter.Close()

	checks := []struct {
		name       string
		mode       string
		method     string
		path       string
		wantHeader string
	}{
		{"legacy mode: list read answered by monolith", modeLegacy, http.MethodGet, articleRoute, upstreamMonolith},
		{"warehouse-bc mode: list read answered by the BC path", modeWarehouseBC, http.MethodGet, articleRoute, upstreamWarehouseBC},
		{"warehouse-bc mode: item routes stay on the monolith", modeWarehouseBC, http.MethodGet, articleRoute + "/42", upstreamMonolith},
		{"warehouse-bc mode: UI assets stay on the monolith", modeWarehouseBC, http.MethodGet, "/js/articles.js", upstreamMonolith},
	}

	for _, tt := range checks {
		t.Run(tt.name, func(t *testing.T) {
			f, err := newFacade(tt.mode, monolith.URL, adapter.URL)
			if err != nil {
				t.Fatalf("newFacade: %v", err)
			}
			srv := httptest.NewServer(f.routes())
			defer srv.Close()

			req, err := http.NewRequest(tt.method, srv.URL+tt.path, nil)
			if err != nil {
				t.Fatalf("build request: %v", err)
			}
			res, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatalf("do request: %v", err)
			}
			defer res.Body.Close()

			if got := res.Header.Get(routeHeader); got != tt.wantHeader {
				t.Errorf("%s %s in mode %s: %s = %q, want %q",
					tt.method, tt.path, tt.mode, routeHeader, got, tt.wantHeader)
			}
		})
	}
}
