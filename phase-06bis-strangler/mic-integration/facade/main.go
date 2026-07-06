// Phase 06bis Strangler facade.
//
// This service owns the routing decision of the cutover. It has no data and
// no business logic: for every incoming request it picks ONE of two backends
// and forwards the request there, then relays the response back.
//
//   - upstream "monolith":     the legacy PHP MIC (serves the UI and every
//     route we have not migrated)
//   - upstream "warehouse-bc": the compatibility adapter in front of the new
//     Warehouse BC (serves the migrated article-list read)
//
// The current dial position comes from the ROUTE_MODE environment variable
// ("legacy" or "warehouse-bc") and is fixed for the lifetime of the process:
// moving through the migration is a restart, exactly like the read-mode dial
// of Phase 04.
package main

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
)

const (
	articleRoute     = "/api/articles"
	routeHeader      = "X-Strangler-Route"

	modeLegacy      = "legacy"
	modeWarehouseBC = "warehouse-bc"

	upstreamMonolith    = "monolith"
	upstreamWarehouseBC = "warehouse-bc"
)

type facade struct {
	mode     string
	monolith http.Handler
	adapter  http.Handler
}

func main() {
	mode := envOr("ROUTE_MODE", modeLegacy)
	if mode != modeLegacy && mode != modeWarehouseBC {
		log.Fatalf("ROUTE_MODE must be %q or %q, got %q", modeLegacy, modeWarehouseBC, mode)
	}

	f, err := newFacade(
		mode,
		envOr("MONOLITH_URL", "http://mic-app:80"),
		envOr("WAREHOUSE_ADAPTER_URL", "http://mic-compat-adapter:8080"),
	)
	if err != nil {
		log.Fatalf("configure facade: %v", err)
	}

	addr := envOr("LISTEN_ADDR", ":80")
	log.Printf("Phase 06bis strangler facade listening on %s (ROUTE_MODE=%s)", addr, mode)
	log.Fatal(http.ListenAndServe(addr, f.routes()))
}

func newFacade(mode, monolithURL, adapterURL string) (*facade, error) {
	monolithProxy, err := proxyTo(monolithURL, upstreamMonolith)
	if err != nil {
		return nil, err
	}
	adapterProxy, err := proxyTo(adapterURL, upstreamWarehouseBC)
	if err != nil {
		return nil, err
	}
	return &facade{mode: mode, monolith: monolithProxy, adapter: adapterProxy}, nil
}

func (f *facade) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/facade/health", f.handleHealth)
	mux.HandleFunc("/", f.handleProxy)
	return mux
}

func (f *facade) handleHealth(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(`{"status":"ok","facade":"strangler","mode":"` + f.mode + `"}`))
}

func (f *facade) handleProxy(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == articleRoute {
		if decideUpstream(r.Method, f.mode) == upstreamWarehouseBC {
			f.adapter.ServeHTTP(w, r)
			return
		}
		f.monolith.ServeHTTP(w, r)
		return
	}

	// Item routes, SPA assets, and every other MIC path stay legacy-owned.
	f.monolith.ServeHTTP(w, r)
}

// decideUpstream is the routing decision of the cutover: given the HTTP
// method of the request and the facade's mode, it names the backend that
// must answer the /api/articles route ("monolith" or "warehouse-bc").
//
// We migrate operation by operation. Today's subset is the article READ
// (the list) and, in Part 3, the article CREATE. Everything else is simply
// not migrated YET.
//
// TODO Phase 06bis Part 1:
//   - GET (the article list): mode "warehouse-bc" sends it to the
//     warehouse-bc upstream; mode "legacy", or anything unexpected, keeps
//     it on the monolith (fail safe, toward the old system);
//   - any other method (create, update, delete): monolith — those
//     operations are not migrated yet. Part 3 will change this rule for
//     the create.
func decideUpstream(method, mode string) string {
	return upstreamMonolith // TODO: everything stays legacy until you implement the decision
}

func proxyTo(rawURL, routeName string) (http.Handler, error) {
	target, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}
	proxy := httputil.NewSingleHostReverseProxy(target)
	proxy.ModifyResponse = func(res *http.Response) error {
		res.Header.Del(routeHeader)
		res.Header.Set(routeHeader, routeName)
		return nil
	}
	proxy.ErrorHandler = func(w http.ResponseWriter, _ *http.Request, err error) {
		log.Printf("proxy %s failed: %v", routeName, err)
		w.Header().Set(routeHeader, routeName)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte(`{"error":"upstream_unavailable","upstream":"` + routeName + `"}`))
	}
	return proxy, nil
}

func envOr(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}
