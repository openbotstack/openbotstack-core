package telemetry

import "time"

// EventLevel classifies telemetry event severity.
type EventLevel string

const (
	EventLevelInfo  EventLevel = "info"
	EventLevelWarn  EventLevel = "warn"
	EventLevelError EventLevel = "error"
)

// TelemetryEvent records a runtime observation for debugging and monitoring.
type TelemetryEvent struct {
	Timestamp  time.Time         `json:"timestamp"`
	TraceID    TraceID           `json:"trace_id"`
	SpanID     SpanID            `json:"span_id"`
	Component  string            `json:"component"`
	Operation  string            `json:"operation"`
	Level      EventLevel        `json:"level"`
	Message    string            `json:"message"`
	Attributes map[string]string `json:"attributes,omitempty"`
}

// TelemetryContext propagates trace and correlation identifiers through the call stack.
type TelemetryContext struct {
	TraceID     TraceID `json:"trace_id"`
	SpanID      SpanID  `json:"span_id"`
	ExecutionID string  `json:"execution_id"`
	RequestID   string  `json:"request_id"`
	TenantID    string  `json:"tenant_id"`
}
