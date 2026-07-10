package main

import (
	"context"
	"errors"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ▸ Task 2 — create_article
//
// This tool WRITES to the warehouse. The BC does not treat the agent
// specially: the request goes through the same auth middleware and the same
// policy you wrote in Phase 07. It succeeds only because the MCP server's
// identity (svc-warehouse-agent) is in the article_creators set; look at
// ../policies/warehouse.rego — that one line is the whole permission.
//
// A write tool raises the bar on the description: the agent must understand
// from your text that this has a side effect, when it is appropriate, and
// exactly which fields are required. The specification is
// tools_create_test.go (ships red).

// CreateArticleInput is the tool's input schema. The BC mints the article id
// at persistence: the agent never supplies one.
//
// TODO Phase 11 Task 2a — write the jsonschema descriptions. Mark what is
// required and what is optional with its default; be precise about
// price_cents (integer cents, not a decimal).
type CreateArticleInput struct {
	SKU         string `json:"sku" jsonschema:"TODO"`
	Name        string `json:"name" jsonschema:"TODO"`
	Description string `json:"description,omitempty" jsonschema:"TODO"`
	PriceCents  int64  `json:"price_cents" jsonschema:"TODO"`
	Currency    string `json:"currency,omitempty" jsonschema:"TODO"`
}

// CreateArticleOutput is the tool's output schema.
type CreateArticleOutput struct {
	Article Article `json:"article" jsonschema:"the persisted article, with the id minted by the BC"`
}

func registerCreateArticle(s *mcp.Server, bc *BCClient) {
	mcp.AddTool(s, &mcp.Tool{
		Name: "create_article",
		// TODO Phase 11 Task 2b — the description must say, for a model
		// reader: this tool CREATES an article (a side effect, not a query),
		// when to use it, and which fields are required.
		Description: "TODO",
	}, createArticleHandler(bc))
}

func createArticleHandler(bc *BCClient) mcp.ToolHandlerFor[CreateArticleInput, CreateArticleOutput] {
	return func(ctx context.Context, req *mcp.CallToolRequest, in CreateArticleInput) (*mcp.CallToolResult, CreateArticleOutput, error) {
		// TODO Phase 11 Task 2c — the handler:
		//   1. reject empty SKU or Name locally, without calling the BC:
		//      the error must tell the agent which field is missing;
		//   2. default Currency to "EUR" when omitted;
		//   3. POST /articles via bc.DoJSON (the BC's CreateArticleRequest
		//      shape: sku, name, description, price_cents, currency; no id);
		//   4. a 403 means the policy said no: return an error that names
		//      the policy, so the agent can explain instead of retrying;
		//   5. on success return the persisted article.
		return nil, CreateArticleOutput{}, errors.New("TODO Phase 11 Task 2: implement create_article")
	}
}
