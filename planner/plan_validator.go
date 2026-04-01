package planner

import (
	"errors"
	"fmt"

	"github.com/openbotstack/openbotstack-core/execution"
)

var (
	// ErrNilPlan is returned when an execution plan is nil.
	ErrNilPlan = errors.New("planner: execution plan is nil")

	// ErrEmptyAssistantID is returned when plan has no assistant ID.
	ErrEmptyAssistantID = errors.New("planner: plan has empty assistant ID")

	// ErrTooManySteps is returned when the plan exceeds maximum allowed steps.
	ErrTooManySteps = errors.New("planner: plan exceeds maximum allowed steps")

	// ErrTooManyToolCalls is returned when the plan exceeds maximum allowed tool calls.
	ErrTooManyToolCalls = errors.New("planner: plan exceeds maximum allowed tool calls")

	// ErrInvalidStepType is returned for unrecognized step types.
	ErrInvalidStepType = errors.New("planner: invalid step type")

	// ErrEmptyStepName is returned when a step has no name.
	ErrEmptyStepName = errors.New("planner: step missing name")
)

// Validator enforces structural and limit constraints on an execution plan.
type Validator struct {
	limits ExecutionLimits
}

// NewValidator creates a new plan validator with the given limits.
// If limits is nil, default limits are used.
func NewValidator(limits *ExecutionLimits) *Validator {
	if limits == nil {
		def := DefaultLimits()
		limits = &def
	}
	return &Validator{limits: *limits}
}

// Validate checks if the execution plan is valid and within bounds.
func (v *Validator) Validate(plan *execution.ExecutionPlan) error {
	if plan == nil {
		return ErrNilPlan
	}

	if plan.AssistantID == "" {
		return ErrEmptyAssistantID
	}

	if len(plan.Steps) > v.limits.MaxSteps {
		return fmt.Errorf("%w: got %d, max %d", ErrTooManySteps, len(plan.Steps), v.limits.MaxSteps)
	}

	toolCount := 0
	for i, step := range plan.Steps {
		if step.Name == "" {
			return fmt.Errorf("%w at index %d", ErrEmptyStepName, i)
		}

		switch step.Type {
		case execution.StepTypeSkill:
			// valid
		case execution.StepTypeTool:
			toolCount++
		default:
			return fmt.Errorf("%w at index %d: %q", ErrInvalidStepType, i, step.Type)
		}
	}

	if toolCount > v.limits.MaxToolCalls {
		return fmt.Errorf("%w: got %d, max %d", ErrTooManyToolCalls, toolCount, v.limits.MaxToolCalls)
	}

	return nil
}
