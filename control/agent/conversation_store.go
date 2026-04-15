package agent

import "context"

// ConversationStore persists and retrieves conversation messages.
//
// Directory structure follows the 3+1 layered model:
//
//	System → Tenant → User → Session
//
// Path: {dataDir}/memory/{tenant_id}/users/{user_id}/sessions/{session_id}.md
//
// Implementations MUST enforce tenant isolation: messages stored by
// tenant A must never be visible to tenant B.
//
// Implementations MUST be safe for concurrent use.
type ConversationStore interface {
	// AppendMessage adds a message to a session's conversation.
	// The msg.UserID and msg.TenantID determine the storage path.
	AppendMessage(ctx context.Context, msg SessionMessage) error

	// GetHistory retrieves messages for a session in chronological order.
	// userID is required for the 3+1 layered directory structure.
	// If maxMessages > 0, returns at most that many recent messages.
	// Returns an empty slice (not nil) if no messages exist.
	GetHistory(ctx context.Context, tenantID, userID, sessionID string, maxMessages int) ([]Message, error)

	// GetSummary retrieves the current summary for a session.
	// Returns empty string if no summary exists.
	GetSummary(ctx context.Context, tenantID, userID, sessionID string) (string, error)

	// StoreSummary persists a summary for a session.
	// Replaces any existing summary.
	StoreSummary(ctx context.Context, tenantID, userID, sessionID, summary string) error

	// ClearSession removes all messages and summary for a session.
	ClearSession(ctx context.Context, tenantID, userID, sessionID string) error
}

// SessionMessage is a single message to be persisted.
type SessionMessage struct {
	TenantID  string // tenant isolation key
	UserID    string // user who sent the message
	SessionID string // conversation session identifier
	Role      string // "user", "assistant", "system"
	Content   string // message body
	Timestamp string // RFC3339Nano
}
