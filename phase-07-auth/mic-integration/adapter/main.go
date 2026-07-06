package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log"
	"math"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

type bcArticle struct {
	ID          string `json:"id"`
	SKU         string `json:"sku"`
	Name        string `json:"name"`
	Description string `json:"description"`
	PriceCents  int64  `json:"price_cents"`
	Currency    string `json:"currency"`
}

type micArticle struct {
	ID            string  `json:"id"`
	SKU           string  `json:"sku"`
	Nome          string  `json:"nome"`
	Descrizione   string  `json:"descrizione"`
	PrezzoListino float64 `json:"prezzo_listino"`
	QtaMinima     int     `json:"qta_minima"`
	Categoria     string  `json:"categoria"`
	IVADefault    string  `json:"iva_default"`
	Unita         string  `json:"unita"`
	Status        string  `json:"status"`
}

type micListResponse struct {
	Data []micArticle   `json:"data"`
	Meta map[string]int `json:"meta"`
}

func main() {
	warehouseURL := envOr("WAREHOUSE_BC_URL", "http://warehouse-bc:8081")
	monolithURL := envOr("MONOLITH_URL", "http://mic-app")

	monolithProxy := reverseProxy(monolithURL)
	client := &http.Client{Timeout: 5 * time.Second}

	http.HandleFunc("/api/articles", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			// fall through to the list translation below
		case http.MethodPost:
			handleCreate(client, warehouseURL, w, r)
			return
		default:
			// PUT/DELETE/... still belong to the monolith for now.
			w.Header().Set("X-Strangler-Route", "monolith")
			monolithProxy.ServeHTTP(w, r)
			return
		}

		articles, correlation, err := fetchBCArticles(r.Context(), client, warehouseURL, r.Header.Get("Authorization"), r.Header.Get("X-Workspace-ID"))
		if err != nil {
			log.Printf("fetch warehouse articles: %v", err)
			w.Header().Set("Content-Type", "application/json")
			status := http.StatusBadGateway
			body := map[string]any{
				"error": "warehouse_bc_unavailable",
				"hint":  "Warehouse BC rejected or failed the authenticated read. Log in with the Phase 07 demo IAM flow, then retry.",
			}
			if statusErr, ok := err.(*upstreamStatusError); ok {
				body["upstream_status"] = statusErr.status
				body["upstream_body"] = statusErr.body
				if statusErr.status == http.StatusUnauthorized {
					status = http.StatusUnauthorized
					body["error"] = "warehouse_bc_auth_required"
					body["hint"] = "MIC is routed to the authenticated Warehouse BC, but no valid Bearer token reached it. Complete the demo login and Authorization propagation step."
				}
			}
			w.WriteHeader(status)
			_ = json.NewEncoder(w).Encode(body)
			return
		}

		mapped := make([]micArticle, 0, len(articles))
		for _, a := range articles {
			mapped = append(mapped, toMICArticle(a))
		}
		mapped = filterMICArticles(mapped, r.URL.Query().Get("q"), r.URL.Query().Get("status"))
		paged := page(mapped, r.URL.Query().Get("limit"), r.URL.Query().Get("offset"))

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Strangler-Route", "warehouse-bc")
		// Mirror the BC's correlation headers so identity stays observable
		// from the browser (DevTools) even through the compatibility adapter.
		for _, h := range []string{"X-TS-ID", "X-Workspace-ID"} {
			if v := correlation.Get(h); v != "" {
				w.Header().Set(h, v)
			}
		}
		_ = json.NewEncoder(w).Encode(micListResponse{
			Data: paged,
			Meta: map[string]int{
				"total":  len(mapped),
				"limit":  parseInt(r.URL.Query().Get("limit"), len(mapped)),
				"offset": parseInt(r.URL.Query().Get("offset"), 0),
			},
		})
	})

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok","adapter":"mic-compat","read_route":"warehouse-bc","auth":"bearer-forwarded"}`))
	})

	log.Printf("MIC compatibility adapter listening on :8080 (warehouse=%s, monolith=%s)", warehouseURL, monolithURL)
	log.Fatal(http.ListenAndServe(":8080", nil))
}

// micCreateRequest is the MIC-shaped create payload the monolith's clients
// send. Fields the BC does not model (qta_minima, categoria, iva_default,
// unita, peso_kg, status) are ignored here: the legacy ACL inside the BC
// writes their defaults on insert (the write-side seam).
type micCreateRequest struct {
	SKU           string  `json:"sku"`
	Nome          string  `json:"nome"`
	Descrizione   string  `json:"descrizione"`
	PrezzoListino float64 `json:"prezzo_listino"`
}

// handleCreate translates a MIC create into the BC contract (POST /articles
// with price_cents + currency), then translates the BC response back into the
// MIC create-response shape {"data": {...}} carrying the minted id. The HTTP
// status mirrors the BC's (201 on create).
//
// Phase 07: the BC sits behind IAM, so the incoming Authorization and
// X-Workspace-ID headers are forwarded on the POST, and the BC's correlation
// headers (X-TS-ID / X-Workspace-ID) are mirrored back, exactly like the GET
// path does.
func handleCreate(client *http.Client, base string, w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	var in micCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid JSON body"})
		return
	}

	bcReq := bcArticle{
		ID:          "", // minted by the system of record at persistence
		SKU:         in.SKU,
		Name:        in.Nome,
		Description: in.Descrizione,
		PriceCents:  int64(math.Round(in.PrezzoListino * 100)),
		Currency:    "EUR",
	}
	payload, err := json.Marshal(bcReq)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	req, err := http.NewRequestWithContext(r.Context(), http.MethodPost,
		strings.TrimRight(base, "/")+"/articles", bytes.NewReader(payload))
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	req.Header.Set("Content-Type", "application/json")
	if v := r.Header.Get("Authorization"); strings.TrimSpace(v) != "" {
		req.Header.Set("Authorization", v)
	}
	if v := r.Header.Get("X-Workspace-ID"); strings.TrimSpace(v) != "" {
		req.Header.Set("X-Workspace-ID", v)
	}

	res, err := client.Do(req)
	if err != nil {
		log.Printf("create warehouse article: %v", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadGateway)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"error": "warehouse_bc_unavailable",
			"hint":  "Warehouse BC POST /articles is unavailable. Phase 07 keeps this failure visible so you can discuss blast radius and rollback.",
		})
		return
	}
	defer res.Body.Close()

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		// Mirror the BC's error verbatim: the adapter translates shapes, it
		// does not paper over failures (401/403 from the auth middleware
		// included).
		body, _ := io.ReadAll(io.LimitReader(res.Body, 4096))
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Strangler-Route", "warehouse-bc")
		w.WriteHeader(res.StatusCode)
		_, _ = w.Write(body)
		return
	}

	var created bcArticle
	if err := json.NewDecoder(res.Body).Decode(&created); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadGateway)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid response from warehouse BC"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Strangler-Route", "warehouse-bc")
	// Mirror the BC's correlation headers so identity stays observable from
	// the browser (DevTools) even through the compatibility adapter.
	for _, h := range []string{"X-TS-ID", "X-Workspace-ID"} {
		if v := res.Header.Get(h); v != "" {
			w.Header().Set(h, v)
		}
	}
	w.WriteHeader(res.StatusCode) // mirror the BC: 201 on create
	_ = json.NewEncoder(w).Encode(map[string]any{
		"data": toMICArticle(created),
	})
}

func fetchBCArticles(ctx context.Context, client *http.Client, base, authorization, workspaceID string) ([]bcArticle, http.Header, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(base, "/")+"/articles", nil)
	if err != nil {
		return nil, nil, err
	}
	if strings.TrimSpace(authorization) != "" {
		req.Header.Set("Authorization", authorization)
	}
	if strings.TrimSpace(workspaceID) != "" {
		req.Header.Set("X-Workspace-ID", workspaceID)
	}
	res, err := client.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(res.Body, 512))
		return nil, nil, &upstreamStatusError{
			status: res.StatusCode,
			body:   strings.TrimSpace(string(body)),
		}
	}
	var out []bcArticle
	if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
		return nil, nil, err
	}
	return out, res.Header, nil
}

func toMICArticle(a bcArticle) micArticle {
	return micArticle{
		ID:            a.ID,
		SKU:           a.SKU,
		Nome:          a.Name,
		Descrizione:   a.Description,
		PrezzoListino: float64(a.PriceCents) / 100,
		QtaMinima:     1,
		Categoria:     "WAREHOUSE",
		IVADefault:    "IVA22",
		Unita:         "pz",
		Status:        "attivo",
	}
}

func filterMICArticles(items []micArticle, query, status string) []micArticle {
	query = strings.ToLower(strings.TrimSpace(query))
	status = strings.TrimSpace(status)
	if query == "" && status == "" {
		return items
	}
	out := make([]micArticle, 0, len(items))
	for _, item := range items {
		if status != "" && item.Status != status {
			continue
		}
		if query != "" {
			haystack := strings.ToLower(item.SKU + " " + item.Nome + " " + item.Descrizione)
			if !strings.Contains(haystack, query) {
				continue
			}
		}
		out = append(out, item)
	}
	return out
}

func page(items []micArticle, limitRaw, offsetRaw string) []micArticle {
	limit := parseInt(limitRaw, len(items))
	offset := parseInt(offsetRaw, 0)
	if offset < 0 {
		offset = 0
	}
	if limit <= 0 {
		limit = len(items)
	}
	if offset >= len(items) {
		return []micArticle{}
	}
	end := offset + limit
	if end > len(items) {
		end = len(items)
	}
	return items[offset:end]
}

func reverseProxy(raw string) http.Handler {
	target, err := url.Parse(raw)
	if err != nil {
		log.Fatalf("invalid MONOLITH_URL %q: %v", raw, err)
	}
	return httputil.NewSingleHostReverseProxy(target)
}

func parseInt(raw string, fallback int) int {
	if strings.TrimSpace(raw) == "" {
		return fallback
	}
	v, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return v
}

func envOr(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

type upstreamStatusError struct {
	status int
	body   string
}

func (e *upstreamStatusError) Error() string {
	return "upstream returned HTTP " + strconv.Itoa(e.status)
}
