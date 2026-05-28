package execution

import (
	"context"
	"sync"
	"time"
)

// StepResult represents the outcome of a single execution step (skill or tool).
type StepResult struct {
	StepName string
	Type     string // "skill", "tool", or "llm"
	Output   any
	Error    error
	Duration time.Duration
	StepID   string
	Retries  int
	Fallback bool
}

// ExecutionContext holds the request-scoped state for an execution plan.
// It tracks limits, identities, and the accumulated results of steps.
type ExecutionContext struct {
	// Standard context for cancellation/timeout
	context.Context

	// Request Identity
	RequestID   string
	AssistantID string
	SessionID   string
	TenantID    string
	UserID      string

	// Execution bounds
	StartedAt time.Time
	Deadline  time.Time

	// Loop tracking (used by execution harness)
	LoopMode         string // "harness" (default)
	CurrentTaskIndex int
	CurrentTurn      int

	// GrantedPermissions lists the permissions granted to this execution context.
	// Populated by the control plane before execution. The step executor uses
	// this to gate access to tools with required permissions (read_file, write_file, web_fetch).
	GrantedPermissions []string

	// Request-scoped progress callback for SSE streaming.
	// When set, loop kernels use this instead of the instance-level callback.
	// This prevents cross-tenant callback leakage under concurrent requests.
	ProgressFn func(eventType, content string, turn int, tool string)

	// State (guarded by mutex)
	mu      sync.RWMutex
	results []StepResult
}

// NewExecutionContext creates a new execution context to track a multi-step execution.
func NewExecutionContext(ctx context.Context, reqID, asstID, sessionID, tenantID, userID string) *ExecutionContext {
	// Inherit deadline if available, otherwise we just track StartedAt.
	// Actual timeout enforcement should rely on the inner context.
	deadline, ok := ctx.Deadline()
	if !ok {
		// Just a placeholder if no explicit timeout
		deadline = time.Time{}
	}

	return &ExecutionContext{
		Context:     ctx,
		RequestID:   reqID,
		AssistantID: asstID,
		SessionID:   sessionID,
		TenantID:    tenantID,
		UserID:      userID,
		StartedAt:   time.Now(),
		Deadline:    deadline,
		results:     make([]StepResult, 0),
	}
}

// AddResult appends a step result to the execution history in a thread-safe manner.
func (ec *ExecutionContext) AddResult(res StepResult) {
	ec.mu.Lock()
	defer ec.mu.Unlock()
	
	ec.results = append(ec.results, res)
}

// Results returns a copy of all accumulated step results.
func (ec *ExecutionContext) Results() []StepResult {
	ec.mu.RLock()
	defer ec.mu.RUnlock()
	
	res := make([]StepResult, len(ec.results))
	copy(res, ec.results)
	return res
}
