package execution

import "context"

// HookResult controls what happens after a hook runs.
type HookResult struct {
	Deny   bool   `json:"deny"`
	Reason string `json:"reason,omitempty"`
}

// HookContext provides context information to hooks.
type HookContext struct {
	Step      *ExecutionStep
	StepIndex int
	Plan      *ExecutionPlan
	EC        *ExecutionContext
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
