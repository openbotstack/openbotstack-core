package assistant

import (
	"errors"
	"sync"
)

var (
	ErrAssistantNotFound = errors.New("assistant: not found")
)

// AssistantRegistry manages the registration and lookup of assistant definitions.
type AssistantRegistry struct {
	mu       sync.RWMutex
	profiles map[string]AssistantProfile
	configs  map[string]AssistantConfig
}

// NewAssistantRegistry creates a new in-memory registry.
func NewAssistantRegistry() *AssistantRegistry {
	return &AssistantRegistry{
		profiles: make(map[string]AssistantProfile),
		configs:  make(map[string]AssistantConfig),
	}
}

// Register adds an assistant profile and its default configuration to the registry.
func (r *AssistantRegistry) Register(profile AssistantProfile, config AssistantConfig) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.profiles[profile.ID] = profile
	r.configs[profile.ID] = config
}

// GetProfile retrieves an assistant profile by ID.
func (r *AssistantRegistry) GetProfile(id string) (AssistantProfile, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	p, ok := r.profiles[id]
	if !ok {
		return AssistantProfile{}, ErrAssistantNotFound
	}
	return p, nil
}

// GetConfig retrieves an assistant's default configuration by ID.
func (r *AssistantRegistry) GetConfig(id string) (AssistantConfig, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	c, ok := r.configs[id]
	if !ok {
		return AssistantConfig{}, ErrAssistantNotFound
	}
	return c, nil
}
