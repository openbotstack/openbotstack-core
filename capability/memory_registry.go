package capability

import (
	"context"
	"fmt"

	registry "github.com/openbotstack/openbotstack-core/registry/skills"
)

// MemoryCapabilityRegistry is an in-memory, thread-safe implementation of CapabilityRegistry.
type MemoryCapabilityRegistry struct {
	store *registry.MapStore[Capability]
}

// NewMemoryCapabilityRegistry creates a new empty registry.
func NewMemoryCapabilityRegistry() *MemoryCapabilityRegistry {
	return &MemoryCapabilityRegistry{
		store: registry.NewMapStore[Capability](),
	}
}

func (r *MemoryCapabilityRegistry) Register(_ context.Context, cap Capability) error {
	if cap == nil {
		return fmt.Errorf("capability must not be nil")
	}
	r.store.Put(cap.ID(), cap)
	return nil
}

func (r *MemoryCapabilityRegistry) Unregister(_ context.Context, id string) error {
	r.store.Delete(id)
	return nil
}

func (r *MemoryCapabilityRegistry) Get(id string) (Capability, error) {
	c, ok := r.store.Get(id)
	if !ok {
		return nil, fmt.Errorf("capability %q not found", id)
	}
	return c, nil
}

func (r *MemoryCapabilityRegistry) List() []CapabilityDescriptor {
	var descs []CapabilityDescriptor
	r.store.ForEach(func(_ string, c Capability) {
		descs = append(descs, capToDescriptor(c))
	})
	return descs
}

func (r *MemoryCapabilityRegistry) ListByKind(kind CapabilityKind) []CapabilityDescriptor {
	var descs []CapabilityDescriptor
	r.store.ForEach(func(_ string, c Capability) {
		if c.Kind() == kind {
			descs = append(descs, capToDescriptor(c))
		}
	})
	return descs
}

func capToDescriptor(c Capability) CapabilityDescriptor {
	return CapabilityDescriptor{
		ID:          c.ID(),
		Name:        c.Name(),
		Description: c.Description(),
		InputSchema: c.InputSchema(),
		Kind:        string(c.Kind()),
		SourceID:    c.SourceID(),
	}
}
