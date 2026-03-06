// Package runtime defines the execution interfaces for openbotstack-core.
//
// These interfaces are defined in core but implemented in openbotstack-runtime.
// This ensures the control plane can declare execution requirements without
// depending on runtime implementation details.
package runtime

import (
	"context"
	"time"

	"github.com/openbotstack/openbotstack-core/skill"
)

// ExecutionRequest is the input to skill execution.
type ExecutionRequest struct {
	// SkillID identifies the skill to execute.
	SkillID string

	// Input is the JSON input to the skill.
	Input []byte

	// Timeout overrides the skill's default timeout.
	Timeout time.Duration

	// TenantID for resource isolation.
	TenantID string

	// UserID for audit logging.
	UserID string

	// RequestID for tracing.
	RequestID string
}

// ExecutionResult is the output from skill execution.
type ExecutionResult struct {
	// Output is the JSON output from the skill.
	Output []byte

	// Error is the error message if execution failed.
	Error string

	// Status indicates execution outcome.
	Status ExecutionStatus

	// Duration is the actual execution time.
	Duration time.Duration

	// ResourceUsage tracks consumed resources.
	ResourceUsage ResourceUsage
}

// ExecutionStatus indicates the outcome of execution.
type ExecutionStatus string

const (
	StatusSuccess  ExecutionStatus = "success"
	StatusFailed   ExecutionStatus = "failed"
	StatusTimeout  ExecutionStatus = "timeout"
	StatusCanceled ExecutionStatus = "canceled"
	StatusRejected ExecutionStatus = "rejected" // policy denied
)

// ResourceUsage tracks execution resource consumption.
type ResourceUsage struct {
	CPUTimeMs  int64
	MemoryMB   int64
	TokensUsed int64
}

// SkillExecutor executes skills in a sandboxed environment.
//
// This interface is defined in core but implemented in runtime.
type SkillExecutor interface {
	// Execute runs a skill with the given input.
	Execute(ctx context.Context, req ExecutionRequest) (*ExecutionResult, error)

	// CanExecute checks if the skill can be executed.
	CanExecute(ctx context.Context, skillID string) (bool, error)

	// LoadSkill prepares a skill for execution.
	LoadSkill(ctx context.Context, pkg skill.Skill) error
}

// CircuitBreaker prevents cascading failures in execution.
type CircuitBreaker interface {
	// Allow checks if the circuit allows execution.
	Allow(ctx context.Context, skillID string) (bool, error)

	// RecordSuccess records a successful execution.
	RecordSuccess(ctx context.Context, skillID string)

	// RecordFailure records a failed execution.
	RecordFailure(ctx context.Context, skillID string)

	// State returns the current circuit state.
	State(ctx context.Context, skillID string) (CircuitState, error)
}

// CircuitState indicates the circuit breaker state.
type CircuitState string

const (
	CircuitClosed   CircuitState = "closed"    // normal operation
	CircuitOpen     CircuitState = "open"      // blocking requests
	CircuitHalfOpen CircuitState = "half_open" // testing recovery
)
