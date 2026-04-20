package audit

import (
	"context"
	"fmt"
	"sync"
)

// AuditSubscriber receives audit events in real-time.
type AuditSubscriber interface {
	OnEvent(ctx context.Context, event AuditEvent) error
	ID() string
}

// AuditEmitter publishes audit events to subscribers.
type AuditEmitter struct {
	mu          sync.RWMutex
	subscribers map[string]AuditSubscriber
}

// NewEmitter creates a new audit emitter.
func NewEmitter() *AuditEmitter {
	return &AuditEmitter{
		subscribers: make(map[string]AuditSubscriber),
	}
}

// Emit publishes an event to all registered subscribers.
// Errors from individual subscribers are logged but don't stop delivery to others.
// Panics from subscribers are recovered.
func (e *AuditEmitter) Emit(ctx context.Context, event AuditEvent) error {
	if ctx == nil {
		return fmt.Errorf("audit: context is required")
	}

	// Auto-generate event ID if empty
	if event.ID == "" {
		event.ID = fmt.Sprintf("auto-%d", len(event.Action))
	}

	e.mu.RLock()
	subs := make([]AuditSubscriber, 0, len(e.subscribers))
	for _, s := range e.subscribers {
		subs = append(subs, s)
	}
	e.mu.RUnlock()

	for _, s := range subs {
		func() {
			defer func() {
				recover() //nolint:errcheck // swallow panics from subscribers
			}()
			_ = s.OnEvent(ctx, event)
		}()
	}

	return nil
}

// Subscribe registers a subscriber for real-time events.
func (e *AuditEmitter) Subscribe(s AuditSubscriber) error {
	if s == nil {
		return fmt.Errorf("audit: subscriber is required")
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	if _, exists := e.subscribers[s.ID()]; exists {
		return fmt.Errorf("audit: subscriber %s already registered", s.ID())
	}

	e.subscribers[s.ID()] = s
	return nil
}

// Unsubscribe removes a subscriber.
func (e *AuditEmitter) Unsubscribe(id string) error {
	if id == "" {
		return fmt.Errorf("audit: subscriber ID is required")
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	if _, exists := e.subscribers[id]; !exists {
		return fmt.Errorf("audit: subscriber %s not found", id)
	}

	delete(e.subscribers, id)
	return nil
}

// Subscribers returns the count of active subscribers.
func (e *AuditEmitter) Subscribers() int {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return len(e.subscribers)
}
