package execution

import (
	"context"
	"time"
)

// --- Hooks ---

// HookResult controls what happens after a hook runs.
type HookResult struct {
	Deny   bool   `json:"deny"`
	Reason string `json:"reason,omitempty"`
}

// HookContext provides context information to hooks.
type HookContext struct {
	Step       *ExecutionStep
	StepIndex  int
	Plan       *ExecutionPlan
	EC         *ExecutionContext
	ToolInput  map[string]any
	ToolOutput any
}

// Hook function signatures. All hooks run outside the LLM context
// and must be deterministic Go code.
type (
	PreStepExecuteHook  func(ctx context.Context, hctx *HookContext) (*HookResult, error)
	PostStepExecuteHook func(ctx context.Context, hctx *HookContext) error
	PreToolUseHook      func(ctx context.Context, hctx *HookContext) (*HookResult, error)
	PostToolUseHook     func(ctx context.Context, hctx *HookContext) error
	OnStopHook          func(ctx context.Context, hctx *HookContext)
)

// --- Failure handling ---

// RetryPolicy defines per-step retry behavior.
type RetryPolicy struct {
	MaxRetries     int           `json:"max_retries"`
	InitialBackoff time.Duration `json:"initial_backoff"`
	MaxBackoff     time.Duration `json:"max_backoff"`
	FailFast       bool          `json:"fail_fast"`
	FallbackTool   string        `json:"fallback_tool,omitempty"`
}

// DefaultRetryPolicy returns the standard retry configuration.
func DefaultRetryPolicy() RetryPolicy {
	return RetryPolicy{
		MaxRetries:     2,
		InitialBackoff: 500 * time.Millisecond,
		MaxBackoff:     5 * time.Second,
		FailFast:       true,
	}
}

// StepFailure records the outcome of a failed step execution.
type StepFailure struct {
	StepID    string
	StepName  string
	Attempt   int
	Error     error
	WillRetry bool
	Fallback  bool
}

// --- Permissions ---

// ApprovalMode controls how tool execution approvals work.
type ApprovalMode string

const (
	ApprovalModeAuto    ApprovalMode = "auto"
	ApprovalModeRequire ApprovalMode = "require"
	ApprovalModeDeny    ApprovalMode = "deny"
)

// PermissionConfig controls tool/skill execution permissions per-execution.
type PermissionConfig struct {
	AllowedTools map[string]bool `json:"allowed_tools,omitempty"`
	DeniedTools  map[string]bool `json:"denied_tools,omitempty"`
	ApprovalMode ApprovalMode    `json:"approval_mode,omitempty"`
}

// IsAllowed checks if a tool/skill is permitted by this config.
func (pc *PermissionConfig) IsAllowed(name string) (bool, string) {
	if pc == nil {
		return true, ""
	}
	if len(pc.DeniedTools) > 0 && pc.DeniedTools[name] {
		return false, "denied by permission config"
	}
	if len(pc.AllowedTools) > 0 && !pc.AllowedTools[name] {
		return false, "not in allowed list"
	}
	if pc.ApprovalMode == ApprovalModeDeny {
		return false, "approval mode is deny"
	}
	return true, ""
}
