package router

import (
	"github.com/openbotstack/openbotstack-core/ai"
	"github.com/openbotstack/openbotstack-core/ai/providers"
	"sync"
	"github.com/openbotstack/openbotstack-core/control/skills"
)

// DefaultRouter implements ModelRouter with capability-based routing.
type DefaultRouter struct {
	mu        sync.RWMutex
	providers map[string]providers.ModelProvider
}

// NewDefaultRouter creates a new router.
func NewDefaultRouter() *DefaultRouter {
	return &DefaultRouter{
		providers: make(map[string]providers.ModelProvider),
	}
}

// Register adds a provider to the router.
func (r *DefaultRouter) Register(provider providers.ModelProvider) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	id := provider.ID()
	if _, exists := r.providers[id]; exists {
		return ai.ErrProviderAlreadyExists
	}
	r.providers[id] = provider
	return nil
}

// Route selects the best provider for the given requirements.
func (r *DefaultRouter) Route(requirements []skills.CapabilityType, constraints skills.ModelConstraints) (providers.ModelProvider, error) {
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

	return nil, ai.ErrNoMatchingProvider
}

// Replace registers or replaces a provider in the router.
// Unlike Register, this overwrites any existing provider with the same ID.
// Use for runtime reconfiguration (e.g., admin API updates).
func (r *DefaultRouter) Replace(provider providers.ModelProvider) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.providers[provider.ID()] = provider
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
func hasAllCapabilities(provider providers.ModelProvider, requirements []skills.CapabilityType) bool {
	caps := provider.Capabilities()
	capSet := make(map[skills.CapabilityType]bool)
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
