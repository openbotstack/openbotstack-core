package assistant

import (
	"errors"
	"sync"

	"github.com/openbotstack/openbotstack-core/registry/skills"
)

var (
	ErrAssistantNotFound = errors.New("assistant: not found")
)

// AssistantRegistry manages the registration and lookup of assistant definitions.
type AssistantRegistry struct {
	mu      sync.Mutex // guards atomic paired writes to profile+config
	profiles *skills.MapStore[AssistantProfile]
	configs  *skills.MapStore[AssistantConfig]
}

// NewAssistantRegistry creates a new in-memory registry.
func NewAssistantRegistry() *AssistantRegistry {
	return &AssistantRegistry{
		profiles: skills.NewMapStore[AssistantProfile](),
		configs:  skills.NewMapStore[AssistantConfig](),
	}
}

// Register adds an assistant profile and its default configuration to the registry.
func (r *AssistantRegistry) Register(profile AssistantProfile, config AssistantConfig) {
	r.mu.Lock()
	r.profiles.Put(profile.ID, profile)
	r.configs.Put(profile.ID, config)
	r.mu.Unlock()
}

// GetProfile retrieves an assistant profile by ID.
func (r *AssistantRegistry) GetProfile(id string) (AssistantProfile, error) {
	p, ok := r.profiles.Get(id)
	if !ok {
		return AssistantProfile{}, ErrAssistantNotFound
	}
	return p, nil
}

// GetConfig retrieves an assistant's default configuration by ID.
func (r *AssistantRegistry) GetConfig(id string) (AssistantConfig, error) {
	c, ok := r.configs.Get(id)
	if !ok {
		return AssistantConfig{}, ErrAssistantNotFound
	}
	return c, nil
}
