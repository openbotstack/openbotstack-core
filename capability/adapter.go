package capability

import (
	"fmt"

	aitypes "github.com/openbotstack/openbotstack-core/ai/types"
	"github.com/openbotstack/openbotstack-core/mcp"
	registry "github.com/openbotstack/openbotstack-core/registry/skills"
)

// capabilityAdapter adapts non-Skill sources (MCP tools, native tools) to the
// Capability interface. Skills no longer need adaptation — they implement
// DescriptorProvider and are wrapped by skillCapability which delegates
// directly to the Skill's Descriptor() method.
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

// skillCapability wraps a Skill to satisfy Capability without field copying.
// It delegates directly to registry.GetDescriptor(s) which calls the Skill's
// DescriptorProvider if implemented, or falls back to building from core fields.
type skillCapability struct {
	s registry.Skill
}

func (sc *skillCapability) ID() string                       { return sc.s.ID() }
func (sc *skillCapability) Name() string                     { return sc.s.Name() }
func (sc *skillCapability) Description() string              { return sc.s.Description() }
func (sc *skillCapability) Kind() CapabilityKind             { return CapabilityKindSkill }
func (sc *skillCapability) InputSchema() *aitypes.JSONSchema { return sc.s.InputSchema() }
func (sc *skillCapability) SourceID() string                 { return sc.s.ID() }

// Descriptor returns the planner-facing descriptor via registry.GetDescriptor,
// which checks for DescriptorProvider on the underlying Skill.
func (sc *skillCapability) Descriptor() aitypes.SkillDescriptor {
	return aitypes.SkillDescriptor(registry.GetDescriptor(sc.s))
}

// NewFromSkill wraps a Skill as a Capability using zero-copy delegation.
func NewFromSkill(s registry.Skill) Capability {
	return &skillCapability{s: s}
}

// NewFromMCP creates a Capability from an MCP tool discovery.
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

// NewFromNative creates a Capability for a builtin platform tool.
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

// SkillToDescriptor converts a Skill to its planner-facing descriptor.
// Uses registry.GetDescriptor for the canonical conversion — if the Skill
// implements DescriptorProvider, its Descriptor() method is called directly,
// otherwise a default is built from core fields.
func SkillToDescriptor(s registry.Skill) CapabilityDescriptor {
	return CapabilityDescriptor(registry.GetDescriptor(s))
}

// CapabilityToDescriptor converts any Capability to its planner-facing
// SkillDescriptor. For skills (skillCapability), this delegates to
// registry.GetDescriptor which honors DescriptorProvider. For MCP/native
// capabilities, it builds from the Capability methods directly.
func CapabilityToDescriptor(c Capability) aitypes.SkillDescriptor {
	if dp, ok := c.(interface{ Descriptor() aitypes.SkillDescriptor }); ok {
		return dp.Descriptor()
	}
	return aitypes.SkillDescriptor{
		ID:          c.ID(),
		Name:        c.Name(),
		Description: c.Description(),
		InputSchema: c.InputSchema(),
		Kind:        string(c.Kind()),
		SourceID:    c.SourceID(),
	}
}
