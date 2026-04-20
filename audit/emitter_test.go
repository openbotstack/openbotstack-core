package audit

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
)

var ErrSubscriberFailed = errors.New("subscriber failed")

// mockSubscriber is a test subscriber.
type mockSubscriber struct {
	id     string
	events []AuditEvent
	err    error
	panic  bool
	mu     sync.Mutex
}

func (m *mockSubscriber) OnEvent(_ context.Context, event AuditEvent) error {
	if m.panic {
		panic("subscriber panic")
	}
	m.mu.Lock()
	m.events = append(m.events, event)
	m.mu.Unlock()
	return m.err
}

func (m *mockSubscriber) ID() string { return m.id }

// --- Normal Cases (4) ---

func TestEmit_DeliversToSingleSubscriber(t *testing.T) {
	e := NewEmitter()
	s := &mockSubscriber{id: "sub-1"}
	_ = e.Subscribe(s)

	event := AuditEvent{ID: "evt-1", Action: "test", TenantID: "t1"}
	err := e.Emit(context.Background(), event)
	if err != nil {
		t.Fatalf("Emit: %v", err)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(s.events))
	}
	if s.events[0].ID != "evt-1" {
		t.Errorf("event ID = %q, want %q", s.events[0].ID, "evt-1")
	}
}

func TestEmit_DeliversToMultipleSubscribers(t *testing.T) {
	e := NewEmitter()
	s1 := &mockSubscriber{id: "sub-1"}
	s2 := &mockSubscriber{id: "sub-2"}
	_ = e.Subscribe(s1)
	_ = e.Subscribe(s2)

	event := AuditEvent{ID: "evt-1", Action: "test"}
	_ = e.Emit(context.Background(), event)

	s1.mu.Lock()
	s2.mu.Lock()
	defer s1.mu.Unlock()
	defer s2.mu.Unlock()
	if len(s1.events) != 1 || len(s2.events) != 1 {
		t.Errorf("both subscribers should receive 1 event, got %d and %d", len(s1.events), len(s2.events))
	}
}

func TestSubscribe_AddsSubscriber(t *testing.T) {
	e := NewEmitter()
	_ = e.Subscribe(&mockSubscriber{id: "sub-1"})
	_ = e.Subscribe(&mockSubscriber{id: "sub-2"})
	if e.Subscribers() != 2 {
		t.Errorf("Subscribers() = %d, want 2", e.Subscribers())
	}
}

func TestUnsubscribe_RemovesSubscriber(t *testing.T) {
	e := NewEmitter()
	_ = e.Subscribe(&mockSubscriber{id: "sub-1"})
	_ = e.Subscribe(&mockSubscriber{id: "sub-2"})
	_ = e.Unsubscribe("sub-1")
	if e.Subscribers() != 1 {
		t.Errorf("Subscribers() = %d, want 1", e.Subscribers())
	}
}

// --- Abnormal / Edge Cases (13) ---

func TestEmit_NoSubscribers(t *testing.T) {
	e := NewEmitter()
	err := e.Emit(context.Background(), AuditEvent{ID: "evt-1"})
	if err != nil {
		t.Errorf("Emit with no subscribers should succeed, got: %v", err)
	}
}

func TestEmit_SubscriberReturnsError(t *testing.T) {
	e := NewEmitter()
	s1 := &mockSubscriber{id: "failing", err: ErrSubscriberFailed}
	s2 := &mockSubscriber{id: "working"}
	_ = e.Subscribe(s1)
	_ = e.Subscribe(s2)

	err := e.Emit(context.Background(), AuditEvent{ID: "evt-1"})
	if err != nil {
		t.Errorf("Emit should not fail when one subscriber errors, got: %v", err)
	}
	s2.mu.Lock()
	defer s2.mu.Unlock()
	if len(s2.events) != 1 {
		t.Error("working subscriber should still receive event despite other failing")
	}
}

func TestEmit_NilContext(t *testing.T) {
	e := NewEmitter()
	err := e.Emit(nil, AuditEvent{ID: "evt-1"})
	if err == nil {
		t.Error("expected error for nil context")
	}
}

func TestEmit_EmptyEventID(t *testing.T) {
	e := NewEmitter()
	s := &mockSubscriber{id: "sub-1"}
	_ = e.Subscribe(s)

	err := e.Emit(context.Background(), AuditEvent{ID: ""})
	if err != nil {
		t.Fatalf("Emit: %v", err)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.events) != 1 {
		t.Fatal("subscriber should receive event even with empty ID")
	}
	if s.events[0].ID == "" {
		t.Error("empty event ID should be auto-generated")
	}
}

func TestSubscribe_DuplicateID(t *testing.T) {
	e := NewEmitter()
	_ = e.Subscribe(&mockSubscriber{id: "sub-1"})
	err := e.Subscribe(&mockSubscriber{id: "sub-1"})
	if err == nil {
		t.Error("expected error for duplicate subscriber ID")
	}
}

func TestSubscribe_NilSubscriber(t *testing.T) {
	e := NewEmitter()
	err := e.Subscribe(nil)
	if err == nil {
		t.Error("expected error for nil subscriber")
	}
}

func TestUnsubscribe_NotFound(t *testing.T) {
	e := NewEmitter()
	err := e.Unsubscribe("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent subscriber")
	}
}

func TestUnsubscribe_EmptyID(t *testing.T) {
	e := NewEmitter()
	err := e.Unsubscribe("")
	if err == nil {
		t.Error("expected error for empty subscriber ID")
	}
}

func TestEmit_LargeMetadata(t *testing.T) {
	e := NewEmitter()
	s := &mockSubscriber{id: "sub-1"}
	_ = e.Subscribe(s)

	meta := make(map[string]string)
	for i := 0; i < 100; i++ {
		meta[fmt.Sprintf("key_%d", i)] = fmt.Sprintf("value_%d", i)
	}
	err := e.Emit(context.Background(), AuditEvent{ID: "evt-1", Metadata: meta})
	if err != nil {
		t.Errorf("Emit with large metadata: %v", err)
	}
}

func TestEmit_ConcurrentEmitAndSubscribe(t *testing.T) {
	e := NewEmitter()
	var wg sync.WaitGroup

	for i := 0; i < 10; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			_ = e.Emit(context.Background(), AuditEvent{ID: "evt"})
		}()
		go func(i int) {
			defer wg.Done()
			_ = e.Subscribe(&mockSubscriber{id: fmt.Sprintf("sub-%d", i)})
		}(i)
	}
	wg.Wait()
}

func TestEmit_SubscriberPanic(t *testing.T) {
	e := NewEmitter()
	s1 := &mockSubscriber{id: "panicker", panic: true}
	s2 := &mockSubscriber{id: "working"}
	_ = e.Subscribe(s1)
	_ = e.Subscribe(s2)

	err := e.Emit(context.Background(), AuditEvent{ID: "evt-1"})
	if err != nil {
		t.Errorf("Emit should recover from panic, got: %v", err)
	}
	s2.mu.Lock()
	defer s2.mu.Unlock()
	if len(s2.events) != 1 {
		t.Error("working subscriber should receive event despite other panicking")
	}
}

func TestSubscribers_AfterMultipleOps(t *testing.T) {
	e := NewEmitter()
	_ = e.Subscribe(&mockSubscriber{id: "s1"})
	_ = e.Subscribe(&mockSubscriber{id: "s2"})
	_ = e.Subscribe(&mockSubscriber{id: "s3"})
	_ = e.Unsubscribe("s2")
	_ = e.Subscribe(&mockSubscriber{id: "s4"})
	_ = e.Unsubscribe("s1")
	if e.Subscribers() != 2 {
		t.Errorf("Subscribers() = %d, want 2", e.Subscribers())
	}
}

func TestEmit_FilterByTenant(t *testing.T) {
	e := NewEmitter()
	s := &mockSubscriber{id: "sub-1"}
	_ = e.Subscribe(s)

	_ = e.Emit(context.Background(), AuditEvent{ID: "evt-1", TenantID: "t1"})
	_ = e.Emit(context.Background(), AuditEvent{ID: "evt-2", TenantID: "t2"})

	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.events) != 2 {
		t.Errorf("expected 2 events (no tenant filter on basic emitter), got %d", len(s.events))
	}
}
