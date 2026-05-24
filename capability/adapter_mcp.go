package capability

import (
	"fmt"

	skills "github.com/openbotstack/openbotstack-core/control/skills"
	"github.com/openbotstack/openbotstack-core/mcp"
)

// MCPToolAdapter wraps an MCP ClientTool to satisfy the Capability interface.
type MCPToolAdapter struct {
	Tool     mcp.ClientTool
	ServerID string
}

func (a *MCPToolAdapter) ID() string                    { return fmt.Sprintf("mcp.%s.%s", a.ServerID, a.Tool.Name) }
func (a *MCPToolAdapter) Name() string                  { return a.Tool.Name }
func (a *MCPToolAdapter) Description() string           { return a.Tool.Description }
func (a *MCPToolAdapter) Kind() CapabilityKind          { return CapabilityKindMCP }
func (a *MCPToolAdapter) InputSchema() *skills.JSONSchema { return a.Tool.InputSchema }
func (a *MCPToolAdapter) SourceID() string              { return a.ServerID }
