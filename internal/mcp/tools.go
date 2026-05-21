package mcptools

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// RegisterTools registers all session management tools on the MCP server.
func RegisterTools(server *mcp.Server, deps *Deps) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "session_open",
		Description: "Open a new SSH or telnet session to a device",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input SessionOpenInput) (*mcp.CallToolResult, SessionOpenOutput, error) {
		return HandleSessionOpen(ctx, req, input, deps)
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "session_exec",
		Description: "Execute a command on an existing session and wait for the prompt",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input SessionExecInput) (*mcp.CallToolResult, SessionExecOutput, error) {
		return HandleSessionExec(ctx, req, input, deps)
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "session_interact",
		Description: "Send input to a session (e.g. password response) and optionally wait for a pattern",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input SessionInteractInput) (*mcp.CallToolResult, SessionInteractOutput, error) {
		return HandleSessionInteract(ctx, req, input, deps)
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "session_chain",
		Description: "Start a sub-session within an existing session (e.g. telnet from SSH)",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input SessionChainInput) (*mcp.CallToolResult, SessionChainOutput, error) {
		return HandleSessionChain(ctx, req, input, deps)
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "session_list",
		Description: "List all active sessions",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input SessionListInput) (*mcp.CallToolResult, SessionListOutput, error) {
		return HandleSessionList(ctx, req, input, deps)
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "session_close",
		Description: "Close a session (sends exit for chained sessions)",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input SessionCloseInput) (*mcp.CallToolResult, SessionCloseOutput, error) {
		return HandleSessionClose(ctx, req, input, deps)
	})
}
