package audit

import (
	"encoding/json"
	"testing"
	"time"
)

func TestToEnvelope_StepCompleted(t *testing.T) {
	now := time.Now()
	e := AuditEvent{
		ID:        "evt-1",
		TenantID:  "t1",
		UserID:    "u1",
		RequestID: "req-1",
		Action:    "skills.execute",
		Resource:  "tool.search",
		Outcome:   "success",
		Duration:  150 * time.Millisecond,
		Timestamp: now,
		StepID:    "step-0",
		StepName:  "search",
		StepType:  "tool",
		Status:    "completed",
		TraceID:   "abc123",
	}

	env := e.ToEnvelope()

	if env.EventID != "evt-1" {
		t.Errorf("EventID = %q, want %q", env.EventID, "evt-1")
	}
	if env.EventType != EventStepCompleted {
		t.Errorf("EventType = %q, want %q", env.EventType, EventStepCompleted)
	}
	if env.Severity != SeverityInfo {
		t.Errorf("Severity = %q, want %q", env.Severity, SeverityInfo)
	}
	if env.Source != SourceExecutor {
		t.Errorf("Source = %q, want %q", env.Source, SourceExecutor)
	}
	if env.DurationMs != 150 {
		t.Errorf("DurationMs = %d, want 150", env.DurationMs)
	}
	if env.TraceID != "abc123" {
		t.Errorf("TraceID = %q, want %q", env.TraceID, "abc123")
	}
	if env.ExecutionID != "req-1" {
		t.Errorf("ExecutionID = %q, want %q", env.ExecutionID, "req-1")
	}
}

func TestToEnvelope_StepFailed(t *testing.T) {
	e := AuditEvent{
		ID:       "evt-2",
		Action:   "skills.execute",
		Outcome:  "failure",
		Error:    "connection refused",
		Status:   "failed",
		StepID:   "step-1",
		StepType: "tool",
	}

	env := e.ToEnvelope()

	if env.EventType != EventStepFailed {
		t.Errorf("EventType = %q, want %q", env.EventType, EventStepFailed)
	}
	if env.Severity != SeverityError {
		t.Errorf("Severity = %q, want %q", env.Severity, SeverityError)
	}
	if env.Error != "connection refused" {
		t.Errorf("Error = %q, want %q", env.Error, "connection refused")
	}
}

func TestToEnvelope_PolicyDenied(t *testing.T) {
	e := AuditEvent{
		ID:       "evt-3",
		Action:   "policy.enforce",
		Outcome:  "denied",
		Resource: "skill/prescribe_medication",
	}

	env := e.ToEnvelope()

	if env.EventType != EventPolicyDenied {
		t.Errorf("EventType = %q, want %q", env.EventType, EventPolicyDenied)
	}
	if env.Severity != SeverityWarn {
		t.Errorf("Severity = %q, want %q", env.Severity, SeverityWarn)
	}
	if env.Source != SourcePolicy {
		t.Errorf("Source = %q, want %q", env.Source, SourcePolicy)
	}
}

func TestToEnvelope_PayloadWithToolIO(t *testing.T) {
	e := AuditEvent{
		ID:        "evt-4",
		Action:    "tool.call",
		Outcome:   "success",
		ToolInput: map[string]any{"query": "test"},
		ToolOutput: map[string]any{"result": "found"},
	}

	env := e.ToEnvelope()

	if env.Payload == nil {
		t.Fatal("Payload should not be nil when ToolInput or ToolOutput is set")
	}

	var payload map[string]any
	if err := json.Unmarshal(env.Payload, &payload); err != nil {
		t.Fatalf("Payload should be valid JSON: %v", err)
	}
	if _, ok := payload["tool_input"]; !ok {
		t.Error("Payload should contain tool_input")
	}
	if _, ok := payload["tool_output"]; !ok {
		t.Error("Payload should contain tool_output")
	}
}

func TestToEnvelope_JSONSerialization(t *testing.T) {
	e := AuditEvent{
		ID:        "evt-5",
		TenantID:  "t1",
		Action:    "skills.execute",
		Outcome:   "success",
		Duration:  50 * time.Millisecond,
		Timestamp: time.Now(),
		StepID:    "step-0",
		Status:    "completed",
		Metadata:  map[string]string{"key": "value"},
	}

	env := e.ToEnvelope()
	data, err := json.Marshal(env)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}

	if decoded["event_type"] != "execution.step.completed" {
		t.Errorf("event_type = %v, want execution.step.completed", decoded["event_type"])
	}
	if decoded["severity"] != "info" {
		t.Errorf("severity = %v, want info", decoded["severity"])
	}
	if decoded["duration_ms"] != float64(50) {
		t.Errorf("duration_ms = %v, want 50", decoded["duration_ms"])
	}
}
