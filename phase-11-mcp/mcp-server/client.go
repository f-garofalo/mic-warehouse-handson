package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// Article mirrors the BC's ArticleResponse contract (the HTTP API you built
// in Phase 06). The jsonschema tags are read by the agent when it interprets
// tool results.
type Article struct {
	ID          string           `json:"id" jsonschema:"the article id, unique in the warehouse"`
	SKU         string           `json:"sku" jsonschema:"the stock keeping unit code"`
	Name        string           `json:"name"`
	Description string           `json:"description,omitempty"`
	PriceCents  int64            `json:"price_cents" jsonschema:"price in euro cents: 1299 means 12.99"`
	Currency    string           `json:"currency" jsonschema:"ISO 4217 code, e.g. EUR"`
	Inventories []InventoryLevel `json:"inventories,omitempty" jsonschema:"per-location stock levels"`
}

// InventoryLevel is the per-location stock view inside an Article.
type InventoryLevel struct {
	LocationCode string `json:"location_code"`
	Quantity     int32  `json:"quantity"`
	Reserved     int32  `json:"reserved"`
}

// BCClient calls the Warehouse BC HTTP API as a machine caller. It obtains a
// token from the IAM mock via the client_credentials grant (Phase 07): the
// BC sees a Bearer token like any other service traffic and never learns
// that an AI agent is behind it.
type BCClient struct {
	baseURL      string
	tokenURL     string
	clientID     string
	clientSecret string
	httpClient   *http.Client

	mu       sync.Mutex
	token    string
	tokenExp time.Time
}

func NewBCClient(baseURL, tokenURL, clientID, clientSecret string) *BCClient {
	return &BCClient{
		baseURL:      strings.TrimRight(baseURL, "/"),
		tokenURL:     tokenURL,
		clientID:     clientID,
		clientSecret: clientSecret,
		httpClient:   &http.Client{Timeout: 10 * time.Second},
	}
}

// APIError is a non-2xx response from the BC. Status lets a tool handler map
// specific cases (404, 403) to messages the agent can act on.
type APIError struct {
	Status int
	Body   string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("the warehouse BC returned %d: %s", e.Status, e.Body)
}

// DoJSON performs an authenticated request against the BC. A non-nil body is
// sent as JSON; a 2xx response is decoded into out when out is non-nil.
// Non-2xx responses return *APIError.
func (c *BCClient) DoJSON(ctx context.Context, method, path string, body any, out any) error {
	token, err := c.accessToken(ctx)
	if err != nil {
		return fmt.Errorf("obtain M2M token: %w", err)
	}

	var reader io.Reader
	if body != nil {
		payload, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("encode request body: %w", err)
		}
		reader = bytes.NewReader(payload)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reader)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("call warehouse BC: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read BC response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return &APIError{Status: resp.StatusCode, Body: strings.TrimSpace(string(data))}
	}
	if out != nil {
		if err := json.Unmarshal(data, out); err != nil {
			return fmt.Errorf("decode BC response: %w", err)
		}
	}
	return nil
}

// accessToken returns a cached M2M token, requesting a fresh one through the
// client_credentials grant when missing or close to expiry.
func (c *BCClient) accessToken(ctx context.Context) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.token != "" && time.Now().Before(c.tokenExp.Add(-30*time.Second)) {
		return c.token, nil
	}

	form := url.Values{
		"grant_type":    {"client_credentials"},
		"client_id":     {c.clientID},
		"client_secret": {c.clientSecret},
		"scope":         {"platform.m2m"},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("call IAM token endpoint: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("IAM token endpoint returned %d: %s", resp.StatusCode, strings.TrimSpace(string(data)))
	}

	var tokenResp struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.Unmarshal(data, &tokenResp); err != nil {
		return "", fmt.Errorf("decode token response: %w", err)
	}
	if tokenResp.AccessToken == "" {
		return "", fmt.Errorf("IAM returned an empty access_token")
	}

	c.token = tokenResp.AccessToken
	c.tokenExp = time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)
	return c.token, nil
}
