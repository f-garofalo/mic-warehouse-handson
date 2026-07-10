package main

import (
	"context"
	"errors"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ▸ Task 1 — list_articles
//
// The BC endpoint GET /articles returns EVERY article and takes no filters:
// fine for a program, hostile for a conversation. Your tool shapes the
// surface: filter and cap HERE, so the agent gets what it asked for and
// nothing more. The specification is tools_list_test.go (ships red).

// ListArticlesInput is the tool's input schema.
//
// TODO Phase 11 Task 1a — write the two jsonschema descriptions. Each one
// must say what the field means AND what happens when it is omitted; the
// agent decides how to call you based only on this text.
type ListArticlesInput struct {
	Query string `json:"query,omitempty" jsonschema:"TODO"`
	Limit int    `json:"limit,omitempty" jsonschema:"TODO"`
}

// ListArticlesOutput is the tool's output schema.
type ListArticlesOutput struct {
	Count    int       `json:"count" jsonschema:"number of articles returned, after filter and limit"`
	Articles []Article `json:"articles"`
}

func registerListArticles(s *mcp.Server, bc *BCClient) {
	mcp.AddTool(s, &mcp.Tool{
		Name: "list_articles",
		// TODO Phase 11 Task 1b — write the description for a model reader:
		// what the tool returns, when to prefer it over get_article, how
		// query and limit behave. A vague description produces wrong calls.
		Description: "TODO",
	}, listArticlesHandler(bc))
}

func listArticlesHandler(bc *BCClient) mcp.ToolHandlerFor[ListArticlesInput, ListArticlesOutput] {
	return func(ctx context.Context, req *mcp.CallToolRequest, in ListArticlesInput) (*mcp.CallToolResult, ListArticlesOutput, error) {
		// TODO Phase 11 Task 1c — the handler:
		//   1. fetch all articles: bc.DoJSON GET /articles into []Article;
		//   2. if in.Query is set, keep only articles whose SKU or Name
		//      contains it, case-insensitive;
		//   3. cap the result at in.Limit (treat 0 as the default, 20);
		//   4. return Count and Articles.
		return nil, ListArticlesOutput{}, errors.New("TODO Phase 11 Task 1: implement list_articles")
	}
}
