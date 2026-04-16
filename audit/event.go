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
// This struct is the canonical audit event schema shared between core and
// runtime. It is a superset of the runtime execution_logs.Event, adding
// ActorID for control-plane context while retaining all runtime fields.
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
	// This is an extension beyond the runtime Event; runtime consumers can ignore it.
	ActorID string
}
