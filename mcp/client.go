package mcp

import "context"

// Client communicates with an MCP server.
type Client interface {
	// Initialize performs the MCP handshake with the server.
	Initialize(ctx context.Context) error

	// ListTools discovers all tools available on the server.
	ListTools(ctx context.Context) ([]ClientTool, error)

	// CallTool invokes a tool on the server with the given arguments.
	CallTool(ctx context.Context, toolName string, arguments map[string]any) (*CallToolResult, error)

	// Close shuts down the client connection.
	Close() error
}
