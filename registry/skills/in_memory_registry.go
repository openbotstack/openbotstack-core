package skills

import (
	"errors"
	"fmt"
	"log"
	"sync"
)

// InMemoryRegistry is an in-memory implementation of SkillRegistry.
// It is safe for concurrent use.
type InMemoryRegistry struct {
	mu        sync.RWMutex
	skills    map[string]Skill
	callbacks []func(ChangeEvent)
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
	if _, exists := r.skills[id]; exists {
		r.mu.Unlock()
		return fmt.Errorf("%w: %s", ErrSkillAlreadyExists, id)
	}
	r.skills[id] = skill
	cbs := r.snapshotCallbacks()
	r.mu.Unlock()

	r.fireCallbacks(cbs, ChangeEvent{Type: ChangeEventRegister, SkillID: id})
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
func (r *InMemoryRegistry) ListByPermission(permissions []string) []Skill {
	r.mu.RLock()
	defer r.mu.RUnlock()

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

// Unregister removes a skill from the registry.
func (r *InMemoryRegistry) Unregister(id string) error {
	r.mu.Lock()
	if _, exists := r.skills[id]; !exists {
		r.mu.Unlock()
		return fmt.Errorf("%w: %s", ErrSkillNotFound, id)
	}
	delete(r.skills, id)
	cbs := r.snapshotCallbacks()
	r.mu.Unlock()

	r.fireCallbacks(cbs, ChangeEvent{Type: ChangeEventUnregister, SkillID: id})
	return nil
}

// Subscribe registers a callback invoked on register/unregister events.
// Returns an unsubscribe function to remove the callback.
func (r *InMemoryRegistry) Subscribe(callback func(event ChangeEvent)) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.callbacks = append(r.callbacks, callback)
}

// snapshotCallbacks returns a copy of the current callbacks slice.
// Must be called while holding r.mu.
func (r *InMemoryRegistry) snapshotCallbacks() []func(ChangeEvent) {
	cbs := make([]func(ChangeEvent), len(r.callbacks))
	copy(cbs, r.callbacks)
	return cbs
}

// fireCallbacks invokes callbacks outside the lock with panic recovery.
func (r *InMemoryRegistry) fireCallbacks(cbs []func(ChangeEvent), event ChangeEvent) {
	for _, cb := range cbs {
		func() {
			defer func() {
				if rec := recover(); rec != nil {
					log.Printf("registry: callback panic for event %v: %v", event, rec)
				}
			}()
			cb(event)
		}()
	}
}
