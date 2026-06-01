package telemetry

import (
	"testing"
	"time"
)

func TestTelemetryEventCreation(t *testing.T) {
	now := time.Now()
	evt := TelemetryEvent{
		Timestamp:  now,
		TraceID:    NewTraceID(),
		SpanID:     NewSpanID(),
		Component:  "planner",
		Operation:  "plan",
		Level:      EventLevelInfo,
		Message:    "plan generated successfully",
		Attributes: map[string]string{"step_count": "3"},
	}
	if evt.Component != "planner" {
		t.Fatalf("Component = %q, want %q", evt.Component, "planner")
	}
	if evt.Level != EventLevelInfo {
		t.Fatalf("Level = %q, want %q", evt.Level, EventLevelInfo)
	}
}

func TestEventLevels(t *testing.T) {
	levels := []EventLevel{EventLevelInfo, EventLevelWarn, EventLevelError}
	for _, l := range levels {
		if string(l) == "" {
			t.Fatal("EventLevel must have non-empty string representation")
		}
	}
}

func TestTelemetryContextCreation(t *testing.T) {
	tc := TelemetryContext{
		TraceID:     NewTraceID(),
		SpanID:      NewSpanID(),
		ExecutionID: "exec-123",
		RequestID:   "req-456",
		TenantID:    "tenant-1",
	}
	if tc.TraceID == "" {
		t.Fatal("TraceID must not be empty")
	}
	if tc.ExecutionID == "exec-123" && tc.TraceID == TraceID("exec-123") {
		t.Fatal("TraceID must differ from ExecutionID")
	}
}
