// Package audit provides audit event schemas for OpenBotStack.
//
// All significant actions in the control plane must emit audit events
// for traceability, compliance, and debugging.
package audit

import "time"

// AuditEvent represents a single auditable action.
type AuditEvent struct {
	// ID is a unique identifier for this event.
	ID string

	// Type categorizes the event (e.g., "state_transition", "skill_invocation").
	Type string

	// Timestamp is when the event occurred.
	Timestamp time.Time

	// ActorID identifies who/what triggered the event.
	ActorID string

	// Payload contains event-specific data.
	Payload map[string]any

	// RequestID links the event to a specific request.
	RequestID string
}
