package telemetry

import (
	"time"

	"github.com/google/uuid"
)

// TraceID identifies a distributed trace across span boundaries.
type TraceID string

// SpanID identifies a single span within a trace.
type SpanID string

// NewTraceID generates a unique trace identifier.
func NewTraceID() TraceID {
	return TraceID(uuid.New().String())
}

// NewSpanID generates a unique span identifier.
func NewSpanID() SpanID {
	return SpanID(uuid.New().String())
}

// SpanKind categorizes the type of work a span represents.
type SpanKind string

const (
	SpanKindExecution  SpanKind = "execution"
	SpanKindPlanner    SpanKind = "planner"
	SpanKindToolCall   SpanKind = "tool_call"
	SpanKindProvider   SpanKind = "provider"
	SpanKindWasm       SpanKind = "wasm"
	SpanKindSkill      SpanKind = "skill"
	SpanKindCompaction SpanKind = "compaction"
)

// SpanStatus indicates the outcome of a span.
type SpanStatus string

const (
	SpanStatusOK        SpanStatus = "ok"
	SpanStatusError     SpanStatus = "error"
	SpanStatusTimeout   SpanStatus = "timeout"
	SpanStatusCancelled SpanStatus = "cancelled"
)

// SpanEvent records a timestamped occurrence within a span.
type SpanEvent struct {
	Name       string            `json:"name"`
	Timestamp  time.Time         `json:"timestamp"`
	Attributes map[string]string `json:"attributes,omitempty"`
}

// Span represents a unit of work within a trace.
type Span struct {
	TraceID      TraceID           `json:"trace_id"`
	SpanID       SpanID            `json:"span_id"`
	ParentSpanID SpanID            `json:"parent_span_id,omitempty"`
	Name         string            `json:"name"`
	Kind         SpanKind          `json:"kind"`
	StartTime    time.Time         `json:"start_time"`
	EndTime      time.Time         `json:"end_time"`
	Status       SpanStatus        `json:"status"`
	Attributes   map[string]string `json:"attributes,omitempty"`
	Events       []SpanEvent       `json:"events,omitempty"`
}

// Duration returns the elapsed time between start and end.
func (s *Span) Duration() time.Duration {
	return s.EndTime.Sub(s.StartTime)
}

// AddEvent appends a timestamped event to the span.
func (s *Span) AddEvent(name string, attrs map[string]string) {
	s.Events = append(s.Events, SpanEvent{
		Name:       name,
		Timestamp:  time.Now(),
		Attributes: attrs,
	})
}
