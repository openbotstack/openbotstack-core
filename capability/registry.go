package capability

import (
	"context"

	aitypes "github.com/openbotstack/openbotstack-core/ai/types"
)

// CapabilityRegistry aggregates capabilities from multiple providers.
type CapabilityRegistry interface {
	// Register adds a capability.
	Register(ctx context.Context, cap Capability) error
	// Unregister removes a capability by ID.
	Unregister(ctx context.Context, id string) error
	// Get returns a capability by ID.
	Get(id string) (Capability, error)
	// List returns all capability descriptors.
	List() []aitypes.SkillDescriptor
	// ListByKind returns descriptors filtered by kind.
	ListByKind(kind CapabilityKind) []aitypes.SkillDescriptor
}
