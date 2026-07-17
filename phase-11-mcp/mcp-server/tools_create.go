package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ▸ Task 2 — create_article
//
// This tool WRITES to the warehouse. The BC does not treat the agent
// specially: the request goes through the same auth middleware and the same
// policy written in Phase 07. It succeeds only because the MCP server's
// identity (svc-warehouse-agent) is in the article_creators set of
// ../policies/warehouse.rego — that one line is the whole permission.

// CreateArticleInput is the tool's input schema. The BC mints the article id
// at persistence: the agent never supplies one.
type CreateArticleInput struct {
	SKU         string `json:"sku" jsonschema:"Required. Stock keeping unit: the article's unique business code, e.g. SKU-BOLT-M8."`
	Name        string `json:"name" jsonschema:"Required. Human-readable article name, e.g. Hex bolt M8."`
	Description string `json:"description,omitempty" jsonschema:"Optional free-text description; omit when unknown."`
	PriceCents  int64  `json:"price_cents" jsonschema:"Required. Unit price in integer euro cents: 1299 means 12.99. Never pass a decimal or a euro amount."`
	Currency    string `json:"currency,omitempty" jsonschema:"Optional ISO 4217 currency code; defaults to EUR when omitted."`
}

// CreateArticleOutput is the tool's output schema.
type CreateArticleOutput struct {
	Article Article `json:"article" jsonschema:"the persisted article, with the id minted by the BC"`
}

func registerCreateArticle(s *mcp.Server, bc *BCClient) {
	mcp.AddTool(s, &mcp.Tool{
		Name: "create_article",
		Description: "Create a NEW article in the warehouse. This is a WRITE with a " +
			"permanent side effect: it persists a new article, so use it only " +
			"when the user wants to add an article that does not exist yet, " +
			"never to look one up (use list_articles or get_article for that). " +
			"Required: sku, name and price_cents (integer euro cents, e.g. " +
			"1299 = 12.99). The warehouse assigns the id; never send one. " +
			"Returns the persisted article, including its minted id.",
	}, createArticleHandler(bc))
}

func createArticleHandler(bc *BCClient) mcp.ToolHandlerFor[CreateArticleInput, CreateArticleOutput] {
	return func(ctx context.Context, req *mcp.CallToolRequest, in CreateArticleInput) (*mcp.CallToolResult, CreateArticleOutput, error) {
		// Reject obviously invalid input before spending a BC round-trip; the
		// error names the missing field so the agent can fix its own call.
		if strings.TrimSpace(in.SKU) == "" {
			return nil, CreateArticleOutput{}, errors.New("sku is required to create an article")
		}
		if strings.TrimSpace(in.Name) == "" {
			return nil, CreateArticleOutput{}, errors.New("name is required to create an article")
		}
		if strings.TrimSpace(in.Currency) == "" {
			in.Currency = "EUR"
		}

		// The input struct already serialises to the BC's CreateArticleRequest
		// shape (sku, name, description, price_cents, currency) with no id: the
		// system of record mints the id (Phase 04).
		var article Article
		err := bc.DoJSON(ctx, http.MethodPost, "/articles", in, &article)

		// The BC's answer carries the outcome. A policy denial (403) and a
		// rejected body (400) are both decisions the agent should surface, not
		// retry — so give each a message it can act on.
		var apiErr *APIError
		if errors.As(err, &apiErr) {
			switch apiErr.Status {
			case http.StatusForbidden:
				return nil, CreateArticleOutput{}, fmt.Errorf("the warehouse policy denied creating this article: the svc-warehouse-agent identity is not allowed to create articles (policies/warehouse.rego decides who may). Explain the denial to the user; do not retry (%s)", apiErr.Body)
			case http.StatusBadRequest:
				return nil, CreateArticleOutput{}, fmt.Errorf("the warehouse rejected the article as invalid: %s", apiErr.Body)
			}
		}
		if err != nil {
			return nil, CreateArticleOutput{}, err
		}
		return nil, CreateArticleOutput{Article: article}, nil
	}
}
