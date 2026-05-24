package capability

import (
	"context"
	"fmt"
	"sync"
)

// MemoryCapabilityRegistry is an in-memory, thread-safe implementation of CapabilityRegistry.
type MemoryCapabilityRegistry struct {
	mu   sync.RWMutex
	caps map[string]Capability
}

// NewMemoryCapabilityRegistry creates a new empty registry.
func NewMemoryCapabilityRegistry() *MemoryCapabilityRegistry {
	return &MemoryCapabilityRegistry{
		caps: make(map[string]Capability),
	}
}

func (r *MemoryCapabilityRegistry) Register(_ context.Context, cap Capability) error {
	if cap == nil {
		return fmt.Errorf("capability must not be nil")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.caps[cap.ID()] = cap
	return nil
}

func (r *MemoryCapabilityRegistry) Unregister(_ context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.caps, id)
	return nil
}

func (r *MemoryCapabilityRegistry) Get(id string) (Capability, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	c, ok := r.caps[id]
	if !ok {
		return nil, fmt.Errorf("capability %q not found", id)
	}
	return c, nil
}

func (r *MemoryCapabilityRegistry) List() []CapabilityDescriptor {
	r.mu.RLock()
	defer r.mu.RUnlock()
	descs := make([]CapabilityDescriptor, 0, len(r.caps))
	for _, c := range r.caps {
		descs = append(descs, capToDescriptor(c))
	}
	return descs
}

func (r *MemoryCapabilityRegistry) ListByKind(kind CapabilityKind) []CapabilityDescriptor {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var descs []CapabilityDescriptor
	for _, c := range r.caps {
		if c.Kind() == kind {
			descs = append(descs, capToDescriptor(c))
		}
	}
	return descs
}

func capToDescriptor(c Capability) CapabilityDescriptor {
	return CapabilityDescriptor{
		ID:          c.ID(),
		Name:        c.Name(),
		Description: c.Description(),
		InputSchema: c.InputSchema(),
		Kind:        c.Kind(),
		SourceID:    c.SourceID(),
	}
}
