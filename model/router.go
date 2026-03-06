package model

import "sync"

// DefaultRouter implements ModelRouter with capability-based routing.
type DefaultRouter struct {
	mu        sync.RWMutex
	providers map[string]ModelProvider
}

// NewDefaultRouter creates a new router.
func NewDefaultRouter() *DefaultRouter {
	return &DefaultRouter{
		providers: make(map[string]ModelProvider),
	}
}

// Register adds a provider to the router.
func (r *DefaultRouter) Register(provider ModelProvider) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	id := provider.ID()
	if _, exists := r.providers[id]; exists {
		return ErrProviderAlreadyExists
	}
	r.providers[id] = provider
	return nil
}

// Route selects the best provider for the given requirements.
func (r *DefaultRouter) Route(requirements []CapabilityType, constraints ModelConstraints) (ModelProvider, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Check preferred provider first
	if constraints.PreferredProvider != "" {
		if provider, ok := r.providers[constraints.PreferredProvider]; ok {
			if hasAllCapabilities(provider, requirements) {
				return provider, nil
			}
		}
	}

	// Find first provider matching all requirements
	for _, provider := range r.providers {
		if hasAllCapabilities(provider, requirements) {
			return provider, nil
		}
	}

	return nil, ErrNoMatchingProvider
}

// List returns all registered provider IDs.
func (r *DefaultRouter) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	ids := make([]string, 0, len(r.providers))
	for id := range r.providers {
		ids = append(ids, id)
	}
	return ids
}

// hasAllCapabilities checks if provider supports all required capabilities.
func hasAllCapabilities(provider ModelProvider, requirements []CapabilityType) bool {
	caps := provider.Capabilities()
	capSet := make(map[CapabilityType]bool)
	for _, c := range caps {
		capSet[c] = true
	}

	for _, req := range requirements {
		if !capSet[req] {
			return false
		}
	}
	return true
}
