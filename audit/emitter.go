package audit

import "context"

// AuditEmitter publishes audit events.
// Full definition deferred to future implementation.
type AuditEmitter interface {
	// Emit publishes an audit event.
	Emit(ctx context.Context, event AuditEvent) error
}
