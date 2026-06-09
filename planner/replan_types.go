package planner

import (
	"context"

	"github.com/openbotstack/openbotstack-core/execution"
)

// ReplanTrigger indicates why replanning was requested.
type ReplanTrigger string

const (
	// ReplanTriggerToolFailure indicates a tool/skill step failed after retries.
	ReplanTriggerToolFailure ReplanTrigger = "tool_failure"

	// ReplanTriggerInvalidData indicates step output was missing or malformed.
	// Reserved for future use (not yet produced by ShouldReplan).
	ReplanTriggerInvalidData ReplanTrigger = "invalid_data"

	// ReplanTriggerExplicitSignal indicates the tool returned an explicit replan request.
	ReplanTriggerExplicitSignal ReplanTrigger = "explicit_signal"

	// ReplanTriggerPolicyAllowed indicates policy explicitly approved replanning.
	// Reserved for future use (not yet produced by ShouldReplan).
	ReplanTriggerPolicyAllowed ReplanTrigger = "policy_allowed"
)

// ReplanContext provides the planner with context about why replanning is needed.
type ReplanContext struct {
	// OriginalPlan is the plan being replaced (read-only, frozen).
	OriginalPlan *execution.ExecutionPlan

	// FailedStep is the step that triggered replanning.
	FailedStep execution.ExecutionStep

	// FailedStepResult is the outcome of the failed step (may be nil if step never completed).
	FailedStepResult *execution.StepResult

	// Trigger indicates why replanning was requested.
	Trigger ReplanTrigger

	// PreviousResults contains outputs from successfully completed steps.
	PreviousResults map[string]any

	// PlannerContext is the original planning context (skills, soul, memory).
	PlannerContext *PlannerContext

	// ErrorMessage describes the failure that triggered replanning.
	ErrorMessage string
}

// Replanner is an optional interface that planners may implement to support
// controlled replanning. The harness injects this via HarnessDeps.
type Replanner interface {
	// Replan generates a new execution plan given the failure context.
	// The returned plan must have ParentID set to OriginalPlan.ID.
	Replan(ctx context.Context, rCtx *ReplanContext) (*execution.ExecutionPlan, error)
}
