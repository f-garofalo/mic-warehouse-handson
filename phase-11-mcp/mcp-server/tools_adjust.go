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

// ▸ Flex task — adjust_inventory
//
// Same pattern a third time, on the inventory endpoint: a write tool whose
// description must make the signed delta unambiguous to a model reader.

// AdjustInventoryInput is the tool's input schema.
type AdjustInventoryInput struct {
	ArticleID    string `json:"article_id" jsonschema:"Required. Id of the article whose stock changes, as returned by list_articles or create_article."`
	LocationCode string `json:"location_code" jsonschema:"Required. Warehouse location whose stock is adjusted, e.g. MAIN."`
	Delta        int32  `json:"delta" jsonschema:"Required signed change in units: positive adds stock (e.g. 10 = ten units received), negative removes it (e.g. -5 = five units written off). Do not pass 0."`
	Reason       string `json:"reason" jsonschema:"Required short human-readable reason for the adjustment, e.g. 'stocktake correction' or 'damaged goods'."`
}

// AdjustInventoryOutput is the tool's output schema.
type AdjustInventoryOutput struct {
	Article Article `json:"article" jsonschema:"the article after the adjustment, with updated stock levels"`
}

func registerAdjustInventory(s *mcp.Server, bc *BCClient) {
	mcp.AddTool(s, &mcp.Tool{
		Name: "adjust_inventory",
		Description: "Adjust the stock level of one article at one location. This is a " +
			"WRITE with a permanent side effect on inventory. Provide article_id " +
			"(from list_articles or create_article), location_code, a signed " +
			"delta (positive receives units, negative removes them) and a short " +
			"reason. Returns the article with its updated per-location stock levels.",
	}, adjustInventoryHandler(bc))
}

func adjustInventoryHandler(bc *BCClient) mcp.ToolHandlerFor[AdjustInventoryInput, AdjustInventoryOutput] {
	return func(ctx context.Context, req *mcp.CallToolRequest, in AdjustInventoryInput) (*mcp.CallToolResult, AdjustInventoryOutput, error) {
		if strings.TrimSpace(in.ArticleID) == "" {
			return nil, AdjustInventoryOutput{}, errors.New("article_id is required: use list_articles to find the article to adjust")
		}
		if strings.TrimSpace(in.LocationCode) == "" {
			return nil, AdjustInventoryOutput{}, errors.New("location_code is required: name the warehouse location whose stock changes")
		}

		// The BC's AdjustInventoryRequest carries only location_code, delta and
		// reason; the article id travels in the path.
		payload := struct {
			LocationCode string `json:"location_code"`
			Delta        int32  `json:"delta"`
			Reason       string `json:"reason"`
		}{in.LocationCode, in.Delta, in.Reason}

		var article Article
		err := bc.DoJSON(ctx, http.MethodPost, "/articles/"+url.PathEscape(in.ArticleID)+"/inventory/adjust", payload, &article)

		// An unknown article is recoverable: name the id and point the agent at
		// the tool that lists the ids that do exist.
		var apiErr *APIError
		if errors.As(err, &apiErr) && apiErr.Status == http.StatusNotFound {
			return nil, AdjustInventoryOutput{}, fmt.Errorf("no article with id %q exists; list_articles shows the ids that do", in.ArticleID)
		}
		if err != nil {
			return nil, AdjustInventoryOutput{}, err
		}
		return nil, AdjustInventoryOutput{Article: article}, nil
	}
}
