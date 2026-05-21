package main

import (
	"context"
	"log"

	mcptools "github.com/KinjiKawaguchi/sessionctl/internal/mcp"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func main() {
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "sessionctl",
		Version: "0.1.0",
	}, nil)

	deps := mcptools.NewDeps()
	mcptools.RegisterTools(server, deps)

	if err := server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
