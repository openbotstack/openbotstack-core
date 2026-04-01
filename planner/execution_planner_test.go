package planner

import (
	"context"
	"testing"
	"time"

	"github.com/openbotstack/openbotstack-core/execution"
)

func TestValidator_Valid(t *testing.T) {
	v := NewValidator(nil) // default limits

	plan := &execution.ExecutionPlan{
		AssistantID: "assistant-1",
		Steps: []execution.ExecutionStep{
			{Type: execution.StepTypeSkill, Name: "patient_summary", Arguments: map[string]any{"id": "123"}},
			{Type: execution.StepTypeTool, Name: "ehr_query", Arguments: map[string]any{"patient_id": "123"}},
		},
	}

	if err := v.Validate(plan); err != nil {
		t.Fatalf("expected valid plan, got err: %v", err)
	}
}

func TestValidator_LimitsEnforced(t *testing.T) {
	v := NewValidator(&ExecutionLimits{
		MaxSteps:         1,
		MaxToolCalls:     0,
		MaxExecutionTime: time.Second,
	})

	t.Run("too many steps", func(t *testing.T) {
		plan := &execution.ExecutionPlan{
			AssistantID: "asst",
			Steps: []execution.ExecutionStep{
				{Type: execution.StepTypeSkill, Name: "s1"},
				{Type: execution.StepTypeSkill, Name: "s2"},
			},
		}
		if err := v.Validate(plan); err == nil {
			t.Fatal("expected ErrTooManySteps")
		}
	})

	t.Run("too many tools", func(t *testing.T) {
		plan := &execution.ExecutionPlan{
			AssistantID: "asst",
			Steps: []execution.ExecutionStep{
				{Type: execution.StepTypeTool, Name: "t1"},
			},
		}
		if err := v.Validate(plan); err == nil {
			t.Fatal("expected ErrTooManyToolCalls")
		}
	})

	t.Run("no assistant id", func(t *testing.T) {
		plan := &execution.ExecutionPlan{
			AssistantID: "",
			Steps:       []execution.ExecutionStep{},
		}
		if err := v.Validate(plan); err == nil {
			t.Fatal("expected ErrEmptyAssistantID")
		}
	})

	t.Run("empty step name", func(t *testing.T) {
		plan := &execution.ExecutionPlan{
			AssistantID: "asst",
			Steps: []execution.ExecutionStep{
				{Type: execution.StepTypeSkill, Name: ""},
			},
		}
		if err := v.Validate(plan); err == nil {
			t.Fatal("expected ErrEmptyStepName")
		}
	})

	t.Run("invalid step type", func(t *testing.T) {
		plan := &execution.ExecutionPlan{
			AssistantID: "asst",
			Steps: []execution.ExecutionStep{
				{Type: execution.StepType("unknown"), Name: "s1"},
			},
		}
		if err := v.Validate(plan); err == nil {
			t.Fatal("expected ErrInvalidStepType")
		}
	})
}

// MockProvider and MockRouter for testing execution planner
func TestLLMPlanner(t *testing.T) {
	// A basic test to make sure structure is correct
	// Note: We don't test LLM generation in unit tests usually so we will
	// just verify the validator and prompt building
	
	planner := NewLLMPlanner(nil, nil)
	prompt := planner.buildPrompt(&PlannerContext{
		AssistantID: "a1",
		UserRequest: "hello",
		Skills: []SkillDescriptor{
			{ID: "skill1", Name: "Skill 1", Description: "A skill"},
		},
	})
	
	if prompt == "" {
		t.Fatal("expected prompt, got empty string")
	}
	
	ctx := context.Background()
	_, err := planner.Plan(ctx, &PlannerContext{})
	
	// Should fail since provider/router is nil or no skills are available
	if err != ErrNoSkillsAvailable {
		t.Fatalf("expected ErrNoSkillsAvailable, got %v", err)
	}
}
