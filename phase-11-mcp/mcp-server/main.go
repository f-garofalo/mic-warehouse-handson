package main

import (
	"context"
	"log"
	"os"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// The MCP server is a thin translator: JSON-RPC tools/call in, authenticated
// HTTP against the Warehouse BC out. It holds no business logic and no
// authorization logic; the BC's policy decides what this identity may do.
//
// stdout is the JSON-RPC wire. Anything a human should read goes to stderr
// (the log package's default): one stray fmt.Println here breaks the agent's
// connection.
func main() {
	log.SetFlags(0)

	bc := NewBCClient(
		envOr("BC_BASE_URL", "http://warehouse-bc:8081"),
		envOr("IAM_TOKEN_URL", "http://iam-mock:9001/oauth/token"),
		envOr("MCP_CLIENT_ID", "svc-warehouse-agent"),
		envOr("MCP_CLIENT_SECRET", "demo"),
	)

	server := mcp.NewServer(&mcp.Implementation{Name: "warehouse-bc", Version: "0.1.0"}, nil)

	registerGetArticle(server, bc)      // given: the worked example
	registerListArticles(server, bc)    // Task 1
	registerCreateArticle(server, bc)   // Task 2
	registerAdjustInventory(server, bc) // Flex

	log.Printf("warehouse-bc MCP server: serving on stdio, BC at %s", envOr("BC_BASE_URL", "http://warehouse-bc:8081"))
	if err := server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		log.Fatal(err)
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
