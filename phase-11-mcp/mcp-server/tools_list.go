package main

import (
	"context"
	"net/http"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ▸ Task 1 — list_articles
//
// The BC endpoint GET /articles returns EVERY article and takes no filters:
// fine for a program, hostile for a conversation. This tool shapes the
// surface — filter and cap HERE — so the agent gets what it asked for and
// nothing more.

// ListArticlesInput is the tool's input schema. These descriptions are the
// only thing the agent reads before choosing query and limit.
type ListArticlesInput struct {
	Query string `json:"query,omitempty" jsonschema:"Optional. Case-insensitive text matched against each article's SKU and name; return only matching articles. Omit to browse the whole catalogue (still capped by limit)."`
	Limit int    `json:"limit,omitempty" jsonschema:"Optional. Maximum number of articles to return. Omit or pass 0 for the default of 20."`
}

// ListArticlesOutput is the tool's output schema.
type ListArticlesOutput struct {
	Count    int       `json:"count" jsonschema:"number of articles returned, after filter and limit"`
	Articles []Article `json:"articles"`
}

func registerListArticles(s *mcp.Server, bc *BCClient) {
	mcp.AddTool(s, &mcp.Tool{
		Name: "list_articles",
		Description: "List articles in the warehouse and return their id, SKU, name, " +
			"price and stock. Prefer this over get_article whenever you do not " +
			"already know an id: pass query to search by SKU or name " +
			"(case-insensitive), and limit to cap how many rows come back " +
			"(default 20). Use the ids it returns to call get_article for detail, " +
			"or create_article / adjust_inventory to act.",
	}, listArticlesHandler(bc))
}

func listArticlesHandler(bc *BCClient) mcp.ToolHandlerFor[ListArticlesInput, ListArticlesOutput] {
	return func(ctx context.Context, req *mcp.CallToolRequest, in ListArticlesInput) (*mcp.CallToolResult, ListArticlesOutput, error) {
		// The BC has no filters: fetch everything, then shape it here.
		var articles []Article
		if err := bc.DoJSON(ctx, http.MethodGet, "/articles", nil, &articles); err != nil {
			return nil, ListArticlesOutput{}, err
		}

		if needle := strings.ToLower(strings.TrimSpace(in.Query)); needle != "" {
			var matched []Article
			for _, a := range articles {
				if strings.Contains(strings.ToLower(a.SKU), needle) ||
					strings.Contains(strings.ToLower(a.Name), needle) {
					matched = append(matched, a)
				}
			}
			articles = matched
		}

		limit := in.Limit
		if limit <= 0 {
			limit = 20
		}
		if len(articles) > limit {
			articles = articles[:limit]
		}

		return nil, ListArticlesOutput{Count: len(articles), Articles: articles}, nil
	}
}
