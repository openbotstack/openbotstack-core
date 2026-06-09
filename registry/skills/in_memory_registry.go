package skills

import (
	"errors"
	"fmt"
	"log"
	"sort"
	"sync"
)

// InMemoryRegistry is an in-memory implementation of SkillRegistry.
// It is safe for concurrent use.
type InMemoryRegistry struct {
	store     *MapStore[Skill]
	cbMu      sync.Mutex
	callbacks []func(ChangeEvent)
}

// NewInMemoryRegistry creates a new empty InMemoryRegistry.
func NewInMemoryRegistry() *InMemoryRegistry {
	return &InMemoryRegistry{
		store: NewMapStore[Skill](),
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

	if ok := r.store.PutIfAbsent(id, skill); !ok {
		return fmt.Errorf("%w: %s", ErrSkillAlreadyExists, id)
	}

	cbs := r.snapshotCallbacks()
	r.fireCallbacks(cbs, ChangeEvent{Type: ChangeEventRegister, SkillID: id})
	return nil
}

// Get retrieves a skill by ID.
func (r *InMemoryRegistry) Get(id string) (Skill, error) {
	s, ok := r.store.Get(id)
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrSkillNotFound, id)
	}
	return s, nil
}

// List returns all registered skill IDs in sorted order.
func (r *InMemoryRegistry) List() []string {
	ids := make([]string, 0, r.store.Len())
	r.store.ForEach(func(id string, _ Skill) {
		ids = append(ids, id)
	})
	sort.Strings(ids)
	return ids
}

// ListByPermission returns skills the caller is allowed to use.
func (r *InMemoryRegistry) ListByPermission(permissions []string) []Skill {
	permSet := make(map[string]struct{}, len(permissions))
	for _, p := range permissions {
		permSet[p] = struct{}{}
	}

	var allowed []Skill
	r.store.ForEach(func(_ string, skill Skill) {
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
	})

	return allowed
}

// Validate checks all registered skills for consistency.
func (r *InMemoryRegistry) Validate() error {
	var err error
	r.store.ForEach(func(id string, skill Skill) {
		if err == nil {
			if vErr := skill.Validate(); vErr != nil {
				err = fmt.Errorf("registry validation failed for skill %s: %w", id, vErr)
			}
		}
	})
	return err
}

// Unregister removes a skill from the registry.
func (r *InMemoryRegistry) Unregister(id string) error {
	if ok := r.store.DeleteIfExists(id); !ok {
		return fmt.Errorf("%w: %s", ErrSkillNotFound, id)
	}

	cbs := r.snapshotCallbacks()
	r.fireCallbacks(cbs, ChangeEvent{Type: ChangeEventUnregister, SkillID: id})
	return nil
}

// Subscribe registers a callback invoked on register/unregister events.
// Returns an unsubscribe function to remove the callback.
func (r *InMemoryRegistry) Subscribe(callback func(event ChangeEvent)) {
	r.cbMu.Lock()
	defer r.cbMu.Unlock()
	r.callbacks = append(r.callbacks, callback)
}

// snapshotCallbacks returns a copy of the current callbacks slice.
func (r *InMemoryRegistry) snapshotCallbacks() []func(ChangeEvent) {
	r.cbMu.Lock()
	defer r.cbMu.Unlock()
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
