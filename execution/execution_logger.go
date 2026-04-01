package execution

import (
	"context"
	"time"
)

// ExecutionLogRecord represents a single log entry for an execution step.
type ExecutionLogRecord struct {
	RequestID   string    `json:"request_id"`
	AssistantID string    `json:"assistant_id"`
	StepName    string    `json:"step_name"`
	StepType    string    `json:"step_type"`
	Status      string    `json:"status"`
	Output      any       `json:"output,omitempty"`
	Error       string    `json:"error,omitempty"`
	Duration    time.Duration `json:"duration_ms"`
	Timestamp   time.Time `json:"timestamp"`
}

// ExecutionLogger defines the interface for recording execution events.
type ExecutionLogger interface {
	// LogStep records the result of a single step.
	LogStep(ctx context.Context, record ExecutionLogRecord) error
	
	// LogPlanStart records the beginning of a multi-step plan.
	LogPlanStart(ctx context.Context, requestID, assistantID string, plan ExecutionPlan) error
	
	// LogPlanEnd records the completion of a multi-step plan.
	LogPlanEnd(ctx context.Context, requestID, assistantID string, err error) error
}
