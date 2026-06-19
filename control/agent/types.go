package agent

import (
	"github.com/openbotstack/openbotstack-core/control/profile"
	"github.com/openbotstack/openbotstack-core/execution"
)

// MessageRequest represents input to the Agent.
type MessageRequest struct {
	TenantID  string `json:"tenant_id"`
	UserID    string `json:"user_id"`
	SessionID string `json:"session_id"`
	Message   string `json:"message"`

	// SessionProfile is an optional session-scope profile overlay (ADR-042). When set,
	// the agent merges Global + Tenant + this session overlay and uses the effective
	// Soul for planning. Only session-allowed fields (language/theme/markdown/...) have
	// effect; others are ignored by the Merge matrix.
	SessionProfile *profile.AssistantProfile `json:"session_profile,omitempty"`

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
