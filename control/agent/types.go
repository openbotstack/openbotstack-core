package agent

import (
	"errors"
	"fmt"
	"github.com/openbotstack/openbotstack-core/execution"
)

// Common errors for the agent package.
var (
	// ErrNilPlan is returned when an execution plan is nil.
	ErrNilPlan = errors.New("agent: execution plan is nil")

	// ErrPlanningFailed is returned when the planner fails to produce a plan.
	ErrPlanningFailed = errors.New("agent: planning failed")

	// ErrNoSkillsAvailable is returned when no skills are registered.
	ErrNoSkillsAvailable = errors.New("agent: no skills available for planning")
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

// Message represents a single chat message in conversation history.
type Message struct {
	Role        string `json:"role"`
	Content     string `json:"content"`
	ExecutionID string `json:"execution_id,omitempty"`
}

// ValidatePlanForAgent validates that a plan has at least one step.
func ValidatePlanForAgent(p *execution.ExecutionPlan) error {
	if p == nil {
		return fmt.Errorf("%w: plan is nil", ErrNilPlan)
	}
	if len(p.Steps) == 0 {
		return fmt.Errorf("%w: plan has no steps", ErrNilPlan)
	}
	return nil
}
