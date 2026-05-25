package capability

import (
	"fmt"

	skills "github.com/openbotstack/openbotstack-core/control/skills"
	"github.com/openbotstack/openbotstack-core/mcp"
	registry "github.com/openbotstack/openbotstack-core/registry/skills"
)

// capabilityAdapter is a private struct that satisfies the Capability interface.
// Use the NewFromSkill, NewFromMCP, or NewFromNative factory functions to create instances.
type capabilityAdapter struct {
	id          string
	name        string
	description string
	kind        CapabilityKind
	inputSchema *skills.JSONSchema
	sourceID    string
}

func (a *capabilityAdapter) ID() string                      { return a.id }
func (a *capabilityAdapter) Name() string                    { return a.name }
func (a *capabilityAdapter) Description() string             { return a.description }
func (a *capabilityAdapter) Kind() CapabilityKind            { return a.kind }
func (a *capabilityAdapter) InputSchema() *skills.JSONSchema { return a.inputSchema }
func (a *capabilityAdapter) SourceID() string                { return a.sourceID }

// NewFromSkill wraps a registry.Skill as a Capability.
func NewFromSkill(s registry.Skill) Capability {
	return &capabilityAdapter{
		id:          s.ID(),
		name:        s.Name(),
		description: s.Description(),
		kind:        CapabilityKindSkill,
		inputSchema: s.InputSchema(),
		sourceID:    s.ID(),
	}
}

// NewFromMCP wraps an MCP ClientTool as a Capability.
// The capability ID is formatted as "mcp.{serverID}.{toolName}".
func NewFromMCP(serverID string, tool mcp.ClientTool) Capability {
	return &capabilityAdapter{
		id:          fmt.Sprintf("mcp.%s.%s", serverID, tool.Name),
		name:        tool.Name,
		description: tool.Description,
		kind:        CapabilityKindMCP,
		inputSchema: tool.InputSchema,
		sourceID:    serverID,
	}
}

// NewFromNative wraps a built-in tool description as a Capability.
// The source ID is always "builtin".
func NewFromNative(id, name, desc string, schema *skills.JSONSchema) Capability {
	return &capabilityAdapter{
		id:          id,
		name:        name,
		description: desc,
		kind:        CapabilityKindNative,
		inputSchema: schema,
		sourceID:    "builtin",
	}
}

// SkillToDescriptor converts a Skill directly to a CapabilityDescriptor.
func SkillToDescriptor(s registry.Skill) CapabilityDescriptor {
	return CapabilityDescriptor{
		ID:          s.ID(),
		Name:        s.Name(),
		Description: s.Description(),
		InputSchema: s.InputSchema(),
		Kind:        CapabilityKindSkill,
		SourceID:    s.ID(),
	}
}
