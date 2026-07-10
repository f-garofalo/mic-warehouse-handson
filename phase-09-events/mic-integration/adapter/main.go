package main

import (
	"database/sql"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

type micArticle struct {
	ID            int64   `json:"id"`
	SKU           string  `json:"sku"`
	Nome          string  `json:"nome"`
	Descrizione   string  `json:"descrizione"`
	PrezzoListino float64 `json:"prezzo_listino"`
	QtaMinima     int     `json:"qta_minima"`
	Categoria     string  `json:"categoria"`
	IVADefault    string  `json:"iva_default"`
	Unita         string  `json:"unita"`
	Status        string  `json:"status"`
	Source        string  `json:"source,omitempty"`
}

type micListResponse struct {
	Data []micArticle   `json:"data"`
	Meta map[string]int `json:"meta"`
}

type legacyListResponse struct {
	Data []micArticle   `json:"data"`
	Meta map[string]int `json:"meta"`
}

func main() {
	monolithURL := envOr("MONOLITH_URL", "http://mic-app")
	db, err := sql.Open("mysql", dsn())
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer db.Close()

	proxy := reverseProxy(monolithURL)
	client := &http.Client{Timeout: 5 * time.Second}

	http.HandleFunc("/api/articles", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("X-Strangler-Route", "mic-monolith")
			proxy.ServeHTTP(w, r)
			return
		}
		handleArticleOptions(w, r, client, db, monolithURL)
	})

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{
			"status":  "ok",
			"adapter": "mic-article-projection",
			"route":   "GET /api/articles -> legacy MIC + warehouse_article_projection",
		})
	})

	log.Printf("MIC article projection adapter listening on :8080 (monolith=%s)", monolithURL)
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func handleArticleOptions(w http.ResponseWriter, r *http.Request, client *http.Client, db *sql.DB, monolithURL string) {
	legacy, err := fetchLegacyArticles(r, client, monolithURL)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": "legacy_articles_unavailable", "detail": err.Error()})
		return
	}
	projected, err := fetchProjectedArticles(db)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": "projection_unavailable", "detail": err.Error()})
		return
	}

	items := append(legacy.Data, projected...)
	items = filter(items, r.URL.Query().Get("q"), r.URL.Query().Get("status"))
	paged := page(items, r.URL.Query().Get("limit"), r.URL.Query().Get("offset"))

	w.Header().Set("X-Strangler-Route", "mic-legacy-plus-hermes-projection")
	writeJSON(w, http.StatusOK, micListResponse{
		Data: paged,
		Meta: map[string]int{
			"total":  len(items),
			"limit":  parseInt(r.URL.Query().Get("limit"), len(items)),
			"offset": parseInt(r.URL.Query().Get("offset"), 0),
		},
	})
}

func fetchLegacyArticles(r *http.Request, client *http.Client, monolithURL string) (legacyListResponse, error) {
	// The adapter fetches the FULL legacy list (the monolith caps at 50 by
	// default) and applies q/status/limit locally, after merging in the
	// projection rows: filtering must see both sources, so forwarding the
	// caller's query string upstream would be wrong (and double-paging with
	// limit would drop rows).
	url := strings.TrimRight(monolithURL, "/") + "/api/articles?limit=10000"
	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, url, nil)
	if err != nil {
		return legacyListResponse{}, err
	}
	res, err := client.Do(req)
	if err != nil {
		return legacyListResponse{}, err
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(res.Body, 512))
		return legacyListResponse{}, &upstreamStatusError{status: res.StatusCode, body: strings.TrimSpace(string(body))}
	}
	var out legacyListResponse
	if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
		return legacyListResponse{}, err
	}
	return out, nil
}

func fetchProjectedArticles(db *sql.DB) ([]micArticle, error) {
	rows, err := db.Query(`
		SELECT id, code, name, description, amount_1, amount_2, text_1, text_2, status
		FROM business_data
		WHERE record_type = 'warehouse_article_projection'
		ORDER BY updated_at DESC, id DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []micArticle
	for rows.Next() {
		var item micArticle
		var desc, category, iva, status sql.NullString
		var price, qta sql.NullFloat64
		if err := rows.Scan(&item.ID, &item.SKU, &item.Nome, &desc, &price, &qta, &category, &iva, &status); err != nil {
			return nil, err
		}
		item.Descrizione = desc.String
		item.PrezzoListino = price.Float64
		item.QtaMinima = int(qta.Float64)
		if item.QtaMinima == 0 {
			item.QtaMinima = 1
		}
		item.Categoria = firstNonEmpty(category.String, "WAREHOUSE-BC")
		item.IVADefault = firstNonEmpty(iva.String, "IVA22")
		item.Unita = "pz"
		item.Status = firstNonEmpty(status.String, "attivo")
		item.Source = "hermes-projection"
		out = append(out, item)
	}
	return out, rows.Err()
}

func filter(items []micArticle, query, status string) []micArticle {
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

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func reverseProxy(raw string) http.Handler {
	target, err := url.Parse(raw)
	if err != nil {
		log.Fatalf("invalid MONOLITH_URL %q: %v", raw, err)
	}
	return httputil.NewSingleHostReverseProxy(target)
}

func dsn() string {
	return envOr("DB_USER", "root") + ":" + envOr("DB_PASS", "root") + "@tcp(" + envOr("DB_HOST", "integration-mysql") + ":" + envOr("DB_PORT", "3306") + ")/" + envOr("DB_NAME", "mic") + "?parseTime=true&charset=utf8mb4&loc=UTC"
}

func envOr(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
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

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

type upstreamStatusError struct {
	status int
	body   string
}

func (e *upstreamStatusError) Error() string {
	return "upstream returned HTTP " + strconv.Itoa(e.status) + ": " + e.body
}
