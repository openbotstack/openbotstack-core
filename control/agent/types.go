package agent

import (
	"github.com/openbotstack/openbotstack-core/execution"
)

// MessageRequest represents input to the Agent.
type MessageRequest struct {
	TenantID  string `json:"tenant_id"`
	UserID    string `json:"user_id"`
	SessionID string `json:"session_id"`
	Message   string `json:"message"`

	// ProgressCallback is an optional per-request callback for execution progress events.
	// When set, it takes priority over any agent-level shared callback, eliminating
	// cross-tenant callback leakage in concurrent request scenarios.
	ProgressCallback func(eventType, content string, turn int, tool string)
}

// MessageResponse represents output from the Agent.
type MessageResponse struct {
	SessionID   string                   `json:"session_id"`
	Message     string                   `json:"message"`
	SkillUsed   string                   `json:"skill_used,omitempty"`
	ExecutionID string                   `json:"execution_id,omitempty"`
	Plan        *execution.ExecutionPlan `json:"plan,omitempty"`
}
