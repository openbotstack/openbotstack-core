// Package audit provides audit event schemas for OpenBotStack.
//
// All significant actions in the control plane must emit audit events
// for traceability, compliance, and debugging.
package audit

import (
	"time"
)

// AuditEvent represents a single auditable action.
//
// This is the single canonical audit event type used across both core and
// runtime. It supersedes the former harness.AuditEntry and
// execution_logs.Event types. Fields fall into two groups:
//
// Core fields (ID..ActorID) are always populated. Step/harness context
// fields (StepID..TraceID) are zero-value when not applicable.
type AuditEvent struct {
	// ID is a unique identifier for this event.
	ID string

	// TenantID identifies the tenant (resource isolation).
	TenantID string

	// UserID identifies the user who triggered the event.
	UserID string

	// RequestID links the event to a specific request for tracing.
	RequestID string

	// Action categorizes the event (e.g., "skills.execute", "model.generate").
	Action string

	// Resource identifies the target resource (e.g., "skill/search", "model/claude").
	Resource string

	// Outcome indicates the result: "success", "failure", "timeout", "started".
	Outcome string

	// Duration is how long the action took.
	Duration time.Duration

	// Metadata contains event-specific key-value data.
	Metadata map[string]string

	// Timestamp is when the event occurred.
	Timestamp time.Time

	// ActorID identifies who/what triggered the event (control-plane context).
	ActorID string

	// Source identifies which subsystem produced the event.
	// When non-empty, ToEnvelope uses this directly instead of inferring.
	Source Source

	// --- Step/harness context fields (zero-value = unset) ---

	// StepID identifies the step within an execution plan.
	StepID string

	// StepName is the human-readable name of the step.
	StepName string

	// StepType is the step type as a string (e.g., "tool", "skill", "llm").
	StepType string

	// Status is the step execution status (e.g., "started", "completed", "failed").
	Status string

	// ToolInput captures the arguments passed to a tool step.
	ToolInput map[string]any

	// ToolOutput captures the result returned by a tool step.
	ToolOutput any

	// Error holds an error message when the step failed.
	Error string

	// TraceID is the distributed trace identifier linking related events.
	TraceID string
}
