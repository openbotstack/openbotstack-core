package audit

import "context"

// AuditEmitter publishes audit events.
// Implementations: runtime/logging/execution_logs.AuditLogger (SQLite-backed).
type AuditEmitter interface {
	// Emit publishes an audit event.
	Emit(ctx context.Context, event AuditEvent) error
}
