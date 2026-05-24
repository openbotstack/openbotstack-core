package telemetry

import (
	"testing"
	"time"
)

func TestSpanLifecycle(t *testing.T) {
	start := time.Now()
	span := Span{
		TraceID:    NewTraceID(),
		SpanID:     NewSpanID(),
		Name:       "execution.run",
		Kind:       SpanKindExecution,
		StartTime:  start,
		EndTime:    start.Add(100 * time.Millisecond),
		Status:     SpanStatusOK,
		Attributes: map[string]string{"skill": "summarize"},
	}
	if span.TraceID == "" {
		t.Fatal("TraceID must not be empty")
	}
	if span.SpanID == "" {
		t.Fatal("SpanID must not be empty")
	}
	if span.Name != "execution.run" {
		t.Fatalf("Name = %q, want %q", span.Name, "execution.run")
	}
	if span.Duration() != 100*time.Millisecond {
		t.Fatalf("Duration = %v, want %v", span.Duration(), 100*time.Millisecond)
	}
}

func TestSpanWithParent(t *testing.T) {
	traceID := NewTraceID()
	parent := Span{
		TraceID: traceID,
		SpanID:  NewSpanID(),
		Name:    "execution.run",
		Kind:    SpanKindExecution,
	}
	child := Span{
		TraceID:      traceID,
		SpanID:       NewSpanID(),
		ParentSpanID: parent.SpanID,
		Name:         "tool.call",
		Kind:         SpanKindToolCall,
	}
	if child.TraceID != parent.TraceID {
		t.Fatal("child trace ID must match parent")
	}
	if child.ParentSpanID != parent.SpanID {
		t.Fatal("child parent span ID must match parent span ID")
	}
}

func TestSpanEvents(t *testing.T) {
	span := Span{
		TraceID: NewTraceID(),
		SpanID:  NewSpanID(),
		Name:    "planner.plan",
		Kind:    SpanKindPlanner,
	}
	span.AddEvent("plan_generated", map[string]string{"step_count": "3"})
	if len(span.Events) != 1 {
		t.Fatalf("Events count = %d, want 1", len(span.Events))
	}
	if span.Events[0].Name != "plan_generated" {
		t.Fatalf("Event name = %q, want %q", span.Events[0].Name, "plan_generated")
	}
}

func TestSpanStatuses(t *testing.T) {
	statuses := []SpanStatus{SpanStatusOK, SpanStatusError, SpanStatusTimeout, SpanStatusCancelled}
	for _, s := range statuses {
		if string(s) == "" {
			t.Fatal("SpanStatus must have non-empty string representation")
		}
	}
}

func TestSpanKinds(t *testing.T) {
	kinds := []SpanKind{
		SpanKindExecution,
		SpanKindPlanner,
		SpanKindToolCall,
		SpanKindProvider,
		SpanKindWasm,
		SpanKindSkill,
		SpanKindCompaction,
	}
	for _, k := range kinds {
		if string(k) == "" {
			t.Fatal("SpanKind must have non-empty string representation")
		}
	}
}

func TestTraceIDsAreUnique(t *testing.T) {
	ids := make(map[TraceID]bool)
	for i := 0; i < 100; i++ {
		id := NewTraceID()
		if ids[id] {
			t.Fatal("TraceID collision detected")
		}
		ids[id] = true
	}
}

func TestSpanIDsAreUnique(t *testing.T) {
	ids := make(map[SpanID]bool)
	for i := 0; i < 100; i++ {
		id := NewSpanID()
		if ids[id] {
			t.Fatal("SpanID collision detected")
		}
		ids[id] = true
	}
}
