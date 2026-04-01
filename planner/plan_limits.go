package planner

import "time"

// ExecutionLimits defines the maximum limits for any execution plan to ensure
// bounded, deterministic execution.
type ExecutionLimits struct {
	MaxSteps         int
	MaxToolCalls     int
	MaxExecutionTime time.Duration
}

// DefaultLimits provides the standard system limits.
func DefaultLimits() ExecutionLimits {
	return ExecutionLimits{
		MaxSteps:         10,
		MaxToolCalls:     5,
		MaxExecutionTime: 10 * time.Second,
	}
}
