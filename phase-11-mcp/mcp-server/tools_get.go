package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// GIVEN — the worked example. Every tool in this server has the same three
// parts; pattern-match them in your tasks:
//
//  1. typed input/output structs: the SDK turns them into the JSON schema
//     the agent reads BEFORE deciding whether and how to call the tool;
//  2. a registration with name + description: the description is the "when
//     and how to use me" contract, written for a model, not for a human;
//  3. a handler: translate the call into an authenticated HTTP request to
//     the BC, and shape errors so the agent can recover on its own.

// GetArticleInput is the tool's input schema.
type GetArticleInput struct {
	ID string `json:"id" jsonschema:"the article id, as returned by list_articles or create_article"`
}

// GetArticleOutput is the tool's output schema.
type GetArticleOutput struct {
	Article Article `json:"article"`
}

func registerGetArticle(s *mcp.Server, bc *BCClient) {
	mcp.AddTool(s, &mcp.Tool{
		Name: "get_article",
		Description: "Fetch one article from the warehouse by its id, including " +
			"per-location stock levels. Use this when you already know the id; " +
			"to search by name or SKU, use list_articles instead.",
	}, getArticleHandler(bc))
}

func getArticleHandler(bc *BCClient) mcp.ToolHandlerFor[GetArticleInput, GetArticleOutput] {
	return func(ctx context.Context, req *mcp.CallToolRequest, in GetArticleInput) (*mcp.CallToolResult, GetArticleOutput, error) {
		if strings.TrimSpace(in.ID) == "" {
			return nil, GetArticleOutput{}, errors.New("id is required: pass an article id, or use list_articles to browse the warehouse")
		}

		var article Article
		err := bc.DoJSON(ctx, http.MethodGet, "/articles/"+url.PathEscape(in.ID), nil, &article)

		// Error text is agent-facing: say what happened AND what to try next.
		var apiErr *APIError
		if errors.As(err, &apiErr) && apiErr.Status == http.StatusNotFound {
			return nil, GetArticleOutput{}, fmt.Errorf("no article with id %q exists; list_articles shows the ids that do", in.ID)
		}
		if err != nil {
			return nil, GetArticleOutput{}, err
		}
		return nil, GetArticleOutput{Article: article}, nil
	}
}
