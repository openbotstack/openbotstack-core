package skills

import (
	"errors"
	"fmt"
	"sync"
)

// InMemoryRegistry is an in-memory implementation of SkillRegistry.
// It is safe for concurrent use.
type InMemoryRegistry struct {
	mu     sync.RWMutex
	skills map[string]Skill
}

// NewInMemoryRegistry creates a new empty InMemoryRegistry.
func NewInMemoryRegistry() *InMemoryRegistry {
	return &InMemoryRegistry{
		skills: make(map[string]Skill),
	}
}

// Register adds a skill to the registry.
func (r *InMemoryRegistry) Register(skill Skill) error {
	if skill == nil {
		return errors.New("cannot register nil skill")
	}

	id := skill.ID()
	if id == "" {
		return errors.New("skill ID cannot be empty")
	}

	if err := skill.Validate(); err != nil {
		return fmt.Errorf("invalid skill %s: %w", id, err)
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.skills[id]; exists {
		return fmt.Errorf("%w: %s", ErrSkillAlreadyExists, id)
	}

	r.skills[id] = skill
	return nil
}

// Get retrieves a skill by ID.
func (r *InMemoryRegistry) Get(id string) (Skill, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if skill, exists := r.skills[id]; exists {
		return skill, nil
	}

	return nil, fmt.Errorf("%w: %s", ErrSkillNotFound, id)
}

// List returns all registered skill IDs.
func (r *InMemoryRegistry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	ids := make([]string, 0, len(r.skills))
	for id := range r.skills {
		ids = append(ids, id)
	}
	return ids
}

// ListByPermission returns skills the caller is allowed to use.
// A skill is included if the provided permissions set contains ALL of its RequiredPermissions.
func (r *InMemoryRegistry) ListByPermission(permissions []string) []Skill {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Build a fast lookup set for provided permissions
	permSet := make(map[string]struct{}, len(permissions))
	for _, p := range permissions {
		permSet[p] = struct{}{}
	}

	var allowed []Skill
	for _, skill := range r.skills {
		reqPerms := skill.RequiredPermissions()
		canUse := true
		for _, req := range reqPerms {
			if _, ok := permSet[req]; !ok {
				canUse = false
				break
			}
		}
		if canUse {
			allowed = append(allowed, skill)
		}
	}

	return allowed
}

// Validate checks all registered skills for consistency.
func (r *InMemoryRegistry) Validate() error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for id, skill := range r.skills {
		if err := skill.Validate(); err != nil {
			return fmt.Errorf("registry validation failed for skill %s: %w", id, err)
		}
	}
	return nil
}
