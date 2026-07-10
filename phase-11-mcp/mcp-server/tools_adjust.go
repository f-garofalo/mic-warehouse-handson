package main

import (
	"context"
	"errors"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ▸ Flex task — adjust_inventory (skip if behind schedule)
//
// Same pattern a third time, on the inventory endpoint. Nothing later
// depends on this tool: if the hour is running out, stop here and go to the
// restitution questions in the README. The specification is
// tools_adjust_test.go (ships red).

// AdjustInventoryInput is the tool's input schema.
//
// TODO Phase 11 Flex a — write the jsonschema descriptions. Delta is signed:
// say so, with an example in each direction.
type AdjustInventoryInput struct {
	ArticleID    string `json:"article_id" jsonschema:"TODO"`
	LocationCode string `json:"location_code" jsonschema:"TODO"`
	Delta        int32  `json:"delta" jsonschema:"TODO"`
	Reason       string `json:"reason" jsonschema:"TODO"`
}

// AdjustInventoryOutput is the tool's output schema.
type AdjustInventoryOutput struct {
	Article Article `json:"article" jsonschema:"the article after the adjustment, with updated stock levels"`
}

func registerAdjustInventory(s *mcp.Server, bc *BCClient) {
	mcp.AddTool(s, &mcp.Tool{
		Name: "adjust_inventory",
		// TODO Phase 11 Flex b — a write tool again: side effect, when to
		// use it, required fields, and what delta means.
		Description: "TODO",
	}, adjustInventoryHandler(bc))
}

func adjustInventoryHandler(bc *BCClient) mcp.ToolHandlerFor[AdjustInventoryInput, AdjustInventoryOutput] {
	return func(ctx context.Context, req *mcp.CallToolRequest, in AdjustInventoryInput) (*mcp.CallToolResult, AdjustInventoryOutput, error) {
		// TODO Phase 11 Flex c — the handler:
		//   1. reject empty ArticleID or LocationCode locally;
		//   2. POST /articles/{article_id}/inventory/adjust with the BC's
		//      AdjustInventoryRequest shape (location_code, delta, reason);
		//   3. map a 404 to an error that points the agent at list_articles;
		//   4. on success return the updated article.
		return nil, AdjustInventoryOutput{}, errors.New("TODO Phase 11 Flex: implement adjust_inventory")
	}
}
