package capability

import "context"

// CapabilityRegistry aggregates capabilities from multiple providers.
type CapabilityRegistry interface {
	// Register adds a capability.
	Register(ctx context.Context, cap Capability) error
	// Unregister removes a capability by ID.
	Unregister(ctx context.Context, id string) error
	// Get returns a capability by ID.
	Get(id string) (Capability, error)
	// List returns all capability descriptors.
	List() []CapabilityDescriptor
	// ListByKind returns descriptors filtered by kind.
	ListByKind(kind CapabilityKind) []CapabilityDescriptor
}
