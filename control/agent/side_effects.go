package agent

import (
	"context"

	"github.com/openbotstack/openbotstack-core/audit"
)

// SideEffects groups persistence and audit operations into one interface.
// The Agent calls these unconditionally — a noop implementation is used in tests
// and when no backend is configured.
type SideEffects interface {
	AppendMessage(ctx context.Context, msg SessionMessage) error
	GetHistory(ctx context.Context, tenantID, userID, sessionID string, maxMessages int) ([]Message, error)
	GetSummary(ctx context.Context, tenantID, userID, sessionID string) (string, error)
	StoreSummary(ctx context.Context, tenantID, userID, sessionID, summary string) error
	ClearSession(ctx context.Context, tenantID, userID, sessionID string) error
	EmitAudit(ctx context.Context, event audit.AuditEvent)
}

// noopSideEffects is the default when no persistence or audit is configured.
type noopSideEffects struct{}

func (noopSideEffects) AppendMessage(_ context.Context, _ SessionMessage) error { return nil }
func (noopSideEffects) GetHistory(_ context.Context, _, _, _ string, _ int) ([]Message, error) {
	return nil, nil
}
func (noopSideEffects) GetSummary(_ context.Context, _, _, _ string) (string, error) {
	return "", nil
}
func (noopSideEffects) StoreSummary(_ context.Context, _, _, _, _ string) error { return nil }
func (noopSideEffects) ClearSession(_ context.Context, _, _, _ string) error     { return nil }
func (noopSideEffects) EmitAudit(_ context.Context, _ audit.AuditEvent)         {}

// composableSideEffects wraps a ConversationStore and AuditEmitter into SideEffects.
type composableSideEffects struct {
	store ConversationStore
	audit *audit.AuditEmitter
}

func (c *composableSideEffects) AppendMessage(ctx context.Context, msg SessionMessage) error {
	return c.store.AppendMessage(ctx, msg)
}

func (c *composableSideEffects) GetHistory(ctx context.Context, tenantID, userID, sessionID string, maxMessages int) ([]Message, error) {
	return c.store.GetHistory(ctx, tenantID, userID, sessionID, maxMessages)
}

func (c *composableSideEffects) GetSummary(ctx context.Context, tenantID, userID, sessionID string) (string, error) {
	return c.store.GetSummary(ctx, tenantID, userID, sessionID)
}

func (c *composableSideEffects) StoreSummary(ctx context.Context, tenantID, userID, sessionID, summary string) error {
	return c.store.StoreSummary(ctx, tenantID, userID, sessionID, summary)
}

func (c *composableSideEffects) ClearSession(ctx context.Context, tenantID, userID, sessionID string) error {
	return c.store.ClearSession(ctx, tenantID, userID, sessionID)
}

func (c *composableSideEffects) EmitAudit(ctx context.Context, event audit.AuditEvent) {
	c.audit.Emit(ctx, event)
}

// NewSideEffects creates a SideEffects from a ConversationStore and AuditEmitter.
// If both are nil, returns a noop implementation.
func NewSideEffects(store ConversationStore, emitter *audit.AuditEmitter) SideEffects {
	if store == nil && emitter == nil {
		return noopSideEffects{}
	}
	return &composableSideEffects{store: store, audit: emitter}
}
