package capability

import (
	"fmt"

	aitypes "github.com/openbotstack/openbotstack-core/ai/types"
	"github.com/openbotstack/openbotstack-core/mcp"
	registry "github.com/openbotstack/openbotstack-core/registry/skills"
)

type capabilityAdapter struct {
	id          string
	name        string
	description string
	kind        CapabilityKind
	inputSchema *aitypes.JSONSchema
	sourceID    string
}

func (a *capabilityAdapter) ID() string                       { return a.id }
func (a *capabilityAdapter) Name() string                     { return a.name }
func (a *capabilityAdapter) Description() string              { return a.description }
func (a *capabilityAdapter) Kind() CapabilityKind             { return a.kind }
func (a *capabilityAdapter) InputSchema() *aitypes.JSONSchema { return a.inputSchema }
func (a *capabilityAdapter) SourceID() string                 { return a.sourceID }

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

func NewFromNative(id, name, desc string, schema *aitypes.JSONSchema) Capability {
	return &capabilityAdapter{
		id:          id,
		name:        name,
		description: desc,
		kind:        CapabilityKindNative,
		inputSchema: schema,
		sourceID:    "builtin",
	}
}

func SkillToDescriptor(s registry.Skill) CapabilityDescriptor {
	return CapabilityDescriptor{
		ID:          s.ID(),
		Name:        s.Name(),
		Description: s.Description(),
		InputSchema: s.InputSchema(),
		Kind:        string(CapabilityKindSkill),
		SourceID:    s.ID(),
	}
}
