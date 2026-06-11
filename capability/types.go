package capability

import (
	aitypes "github.com/openbotstack/openbotstack-core/ai/types"
)

// CapabilityKind classifies the source of a capability.
type CapabilityKind string

const (
	CapabilityKindSkill    CapabilityKind = "skill"
	CapabilityKindMCP      CapabilityKind = "mcp"
	CapabilityKindNative CapabilityKind = "native"
)

// Capability is the universal interface for anything the registry can hold.
type Capability interface {
	ID() string
	Name() string
	Description() string
	Kind() CapabilityKind
	InputSchema() *aitypes.JSONSchema
	SourceID() string
}
