package planner

import (
	"context"
	"testing"

	"github.com/openbotstack/openbotstack-core/execution"
)

func TestReplanTriggerValues(t *testing.T) {
	tests := []struct {
		trigger ReplanTrigger
		want    string
	}{
		{ReplanTriggerToolFailure, "tool_failure"},
		{ReplanTriggerInvalidData, "invalid_data"},
		{ReplanTriggerExplicitSignal, "explicit_signal"},
		{ReplanTriggerPolicyAllowed, "policy_allowed"},
	}
	for _, tt := range tests {
		if string(tt.trigger) != tt.want {
			t.Errorf("ReplanTrigger got %q, want %q", tt.trigger, tt.want)
		}
	}
}

func TestReplanContext_Fields(t *testing.T) {
	origPlan := &execution.ExecutionPlan{
		Steps: []execution.ExecutionStep{
			{Name: "step1", Type: execution.StepTypeTool},
		},
	}
	origPlan.ID = "plan-abc"
	_ = origPlan.Validate()

	failedStep := execution.ExecutionStep{Name: "step2", Type: execution.StepTypeTool, StepID: "step2-id"}
	pCtx := &PlannerContext{
		AssistantID: "assistant-1",
		UserRequest: "test request",
	}

	rCtx := &ReplanContext{
		OriginalPlan:   origPlan,
		FailedStep:     failedStep,
		Trigger:        ReplanTriggerToolFailure,
		PreviousResults: map[string]any{"step1": "result1"},
		PlannerContext: pCtx,
		ErrorMessage:   "tool execution failed",
	}

	if rCtx.OriginalPlan.ID != "plan-abc" {
		t.Errorf("OriginalPlan.ID = %q, want %q", rCtx.OriginalPlan.ID, "plan-abc")
	}
	if rCtx.FailedStep.Name != "step2" {
		t.Errorf("FailedStep.Name = %q, want %q", rCtx.FailedStep.Name, "step2")
	}
	if rCtx.Trigger != ReplanTriggerToolFailure {
		t.Errorf("Trigger = %q, want %q", rCtx.Trigger, ReplanTriggerToolFailure)
	}
	if rCtx.ErrorMessage != "tool execution failed" {
		t.Errorf("ErrorMessage = %q, want %q", rCtx.ErrorMessage, "tool execution failed")
	}
	if rCtx.PlannerContext.AssistantID != "assistant-1" {
		t.Errorf("PlannerContext.AssistantID = %q", rCtx.PlannerContext.AssistantID)
	}
}

func TestReplanContext_NilFields(t *testing.T) {
	rCtx := &ReplanContext{}
	if rCtx.OriginalPlan != nil {
		t.Error("OriginalPlan should be nil by default")
	}
	if rCtx.Trigger != "" {
		t.Errorf("Trigger should be empty by default, got %q", rCtx.Trigger)
	}
	if rCtx.ErrorMessage != "" {
		t.Error("ErrorMessage should be empty by default")
	}
}

// Compile-time check that LLMPlanner satisfies the Replanner interface.
func TestLLMPlanner_ImplementsReplanner(t *testing.T) {
	var _ Replanner = (*LLMPlanner)(nil)
}

// Compile-time check that Replanner interface has the correct method signature.
func TestReplannerInterfaceSignature(t *testing.T) {
	var r Replanner = &stubReplanner{}
	_, _ = r.Replan(context.Background(), &ReplanContext{})
}

type stubReplanner struct{}

func (s *stubReplanner) Replan(ctx context.Context, rCtx *ReplanContext) (*execution.ExecutionPlan, error) {
	return nil, nil
}
