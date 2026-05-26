// Package capability defines the unified capability abstraction layer.
//
// CapabilityDescriptor is what the planner and executor see, regardless of
// whether the capability comes from a skill, an MCP server, a native function,
// or an external service.
package capability

import (
	skills "github.com/openbotstack/openbotstack-core/control/skills"
)

// CapabilityKind classifies the source of a capability.
type CapabilityKind string

const (
	// CapabilityKindSkill is a built-in or declarative skill.
	CapabilityKindSkill CapabilityKind = "skill"

	// CapabilityKindMCP is a tool discovered from an MCP server.
	CapabilityKindMCP CapabilityKind = "mcp"

	// CapabilityKindNative is a host-process capability.
	CapabilityKindNative CapabilityKind = "native"

	// CapabilityKindExternal is an external service capability.
	CapabilityKindExternal CapabilityKind = "external"
)

// CapabilityDescriptor describes a discrete capability that can be presented
// to the planner as an available tool. It is a type alias for skills.SkillDescriptor,
// which now carries Kind and SourceID fields.
type CapabilityDescriptor = skills.SkillDescriptor

// Capability is the universal interface for anything the registry can hold.
// Adapters wrap domain-specific types (Skill, MCP Tool) to satisfy this.
type Capability interface {
	ID() string
	Name() string
	Description() string
	Kind() CapabilityKind
	InputSchema() *skills.JSONSchema
	SourceID() string
}
