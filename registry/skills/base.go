package skills

import "sync"

// MapStore is a generic, goroutine-safe key-value store.
// Domain registries embed it to get mutex-protected CRUD without reimplementing
// the same map+lock boilerplate.
type MapStore[T any] struct {
	mu   sync.RWMutex
	data map[string]T
}

// NewMapStore creates an empty MapStore.
func NewMapStore[T any]() *MapStore[T] {
	return &MapStore[T]{data: make(map[string]T)}
}

// Put stores a value under the given key. Overwrites existing.
func (s *MapStore[T]) Put(key string, val T) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[key] = val
}

// PutIfAbsent stores val under key only if key does not already exist.
// Returns true if the value was stored, false if key already existed.
func (s *MapStore[T]) PutIfAbsent(key string, val T) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.data[key]; ok {
		return false
	}
	s.data[key] = val
	return true
}

// Get retrieves a value by key. Returns the value and whether it was found.
func (s *MapStore[T]) Get(key string) (T, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.data[key]
	return v, ok
}

// Delete removes a key from the store.
func (s *MapStore[T]) Delete(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.data, key)
}

// DeleteIfExists removes a key only if it exists. Returns true if deleted.
func (s *MapStore[T]) DeleteIfExists(key string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.data[key]; !ok {
		return false
	}
	delete(s.data, key)
	return true
}

// Len returns the number of entries.
func (s *MapStore[T]) Len() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.data)
}

// ForEach iterates over all entries while holding the read lock.
// The callback must not modify the store.
func (s *MapStore[T]) ForEach(fn func(key string, val T)) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for k, v := range s.data {
		fn(k, v)
	}
}

// Snapshot returns a shallow copy of the current data.
func (s *MapStore[T]) Snapshot() map[string]T {
	s.mu.RLock()
	defer s.mu.RUnlock()
	cp := make(map[string]T, len(s.data))
	for k, v := range s.data {
		cp[k] = v
	}
	return cp
}
