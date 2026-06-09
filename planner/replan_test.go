package planner

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	aitypes "github.com/openbotstack/openbotstack-core/ai/types"
	"github.com/openbotstack/openbotstack-core/execution"
)

// ---------------------------------------------------------------------------
// Replan tests (Phase 2 — Controlled Replanning)
// ---------------------------------------------------------------------------

// TestReplan_SetsParentID verifies that the new plan's ParentID equals the original plan's ID.
func TestReplan_SetsParentID(t *testing.T) {
	planJSON := `{"assistant_id":"","steps":[{"type":"skill","name":"retry_step","arguments":{"data":"test"}}]}`

	router := &mockRouter{
		provider: &mockProvider{
			response: &aitypes.GenerateResponse{Content: planJSON},
		},
	}
	planner := NewLLMPlanner(router, nil)

	origPlan := &execution.ExecutionPlan{
		AssistantID: "asst-1",
		Steps: []execution.ExecutionStep{
			{Type: execution.StepTypeSkill, Name: "step1"},
			{Type: execution.StepTypeSkill, Name: "step2"},
		},
	}
	if err := origPlan.Validate(); err != nil {
		t.Fatalf("failed to validate original plan: %v", err)
	}

	rCtx := &ReplanContext{
		OriginalPlan: origPlan,
		FailedStep: execution.ExecutionStep{
			Name: "step2", Type: execution.StepTypeSkill, StepID: "step2-id",
		},
		Trigger:        ReplanTriggerToolFailure,
		PreviousResults: map[string]any{"step1": "result1"},
		PlannerContext: &PlannerContext{
			AssistantID: "asst-1",
			UserRequest: "do the thing",
			Skills: []aitypes.SkillDescriptor{
				{ID: "retry_step", Name: "Retry Step", Description: "A retry skill"},
			},
		},
		ErrorMessage: "tool execution failed",
	}

	newPlan, err := planner.Replan(context.Background(), rCtx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if newPlan.ParentID != origPlan.ID {
		t.Errorf("expected ParentID=%q, got %q", origPlan.ID, newPlan.ParentID)
	}
}

// TestReplan_IncludesFailureContext verifies the replan prompt contains failure details.
func TestReplan_IncludesFailureContext(t *testing.T) {
	// We use a capturing provider to intercept the prompt sent to the LLM.
	var capturedPrompt string

	planJSON := `{"assistant_id":"","steps":[{"type":"skill","name":"retry_step","arguments":{}}]}`

	router := &mockRouter{
		provider: &capturingMockProvider{
			response: &aitypes.GenerateResponse{Content: planJSON},
			captureFn: func(req aitypes.GenerateRequest) {
				// Capture the user message (the replan prompt)
				for _, msg := range req.Messages {
					if msg.Role == "user" {
						for _, cb := range msg.Contents {
							if cb.Type == "text" {
								capturedPrompt = cb.Text
							}
						}
					}
				}
			},
		},
	}
	planner := NewLLMPlanner(router, nil)

	origPlan := &execution.ExecutionPlan{
		AssistantID: "asst-1",
		Steps: []execution.ExecutionStep{
			{Type: execution.StepTypeSkill, Name: "fetch_data"},
			{Type: execution.StepTypeTool, Name: "process_data"},
		},
	}
	if err := origPlan.Validate(); err != nil {
		t.Fatalf("validate original plan: %v", err)
	}

	rCtx := &ReplanContext{
		OriginalPlan: origPlan,
		FailedStep: execution.ExecutionStep{
			Name: "process_data", Type: execution.StepTypeTool, StepID: "proc-id",
		},
		Trigger:         ReplanTriggerToolFailure,
		PreviousResults: map[string]any{"fetch_data": "some data"},
		PlannerContext: &PlannerContext{
			AssistantID: "asst-1",
			UserRequest: "process the records",
			Skills: []aitypes.SkillDescriptor{
				{ID: "retry_step", Name: "Retry", Description: "Retry step"},
			},
		},
		ErrorMessage: "database connection timeout after 30s",
	}

	_, err := planner.Replan(context.Background(), rCtx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if capturedPrompt == "" {
		t.Fatal("expected non-empty replan prompt, got empty string")
	}

	// Verify the prompt includes the error message
	if !strings.Contains(capturedPrompt, "database connection timeout after 30s") {
		t.Errorf("replan prompt should contain the error message; got prompt:\n%s", capturedPrompt)
	}

	// Verify the prompt includes the failed step name
	if !strings.Contains(capturedPrompt, "process_data") {
		t.Errorf("replan prompt should contain the failed step name 'process_data'; got prompt:\n%s", capturedPrompt)
	}

	// Verify the prompt mentions replanning
	if !strings.Contains(capturedPrompt, "revised") && !strings.Contains(capturedPrompt, "replan") {
		t.Errorf("replan prompt should mention replanning/revised; got prompt:\n%s", capturedPrompt)
	}
}

// TestReplan_ReturnsValidPlan verifies the returned plan passes validation (frozen, has StepIDs).
func TestReplan_ReturnsValidPlan(t *testing.T) {
	planJSON := `{"assistant_id":"","steps":[{"type":"skill","name":"new_step","arguments":{"key":"value"}}]}`

	router := &mockRouter{
		provider: &mockProvider{
			response: &aitypes.GenerateResponse{Content: planJSON},
		},
	}
	planner := NewLLMPlanner(router, nil)

	origPlan := &execution.ExecutionPlan{
		AssistantID: "asst-1",
		Steps: []execution.ExecutionStep{
			{Type: execution.StepTypeSkill, Name: "old_step"},
		},
	}
	if err := origPlan.Validate(); err != nil {
		t.Fatalf("validate original plan: %v", err)
	}

	rCtx := &ReplanContext{
		OriginalPlan: origPlan,
		FailedStep: execution.ExecutionStep{
			Name: "old_step", Type: execution.StepTypeSkill,
		},
		Trigger:         ReplanTriggerInvalidData,
		PreviousResults: map[string]any{},
		PlannerContext: &PlannerContext{
			AssistantID: "asst-1",
			UserRequest: "do something",
			Skills: []aitypes.SkillDescriptor{
				{ID: "new_step", Name: "New Step", Description: "A new step"},
			},
		},
		ErrorMessage: "invalid output format",
	}

	plan, err := planner.Replan(context.Background(), rCtx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Plan should be frozen
	if !plan.IsFrozen() {
		t.Error("expected replanned plan to be frozen after validation")
	}

	// Plan should have auto-generated StepIDs
	for i, step := range plan.Steps {
		if step.StepID == "" {
			t.Errorf("step %d (%q) should have auto-generated StepID", i, step.Name)
		}
	}

	// Plan should have an auto-generated ID
	if plan.ID == "" {
		t.Error("replanned plan should have auto-generated ID")
	}

	// Plan should have ParentID set
	if plan.ParentID != origPlan.ID {
		t.Errorf("expected ParentID=%q, got %q", origPlan.ID, plan.ParentID)
	}

	// Plan should have AssistantID set from context
	if plan.AssistantID != "asst-1" {
		t.Errorf("expected AssistantID='asst-1', got %q", plan.AssistantID)
	}
}

// TestReplan_NoSkills_ReturnsError verifies that empty skills returns ErrNoSkillsAvailable.
func TestReplan_NoSkills_ReturnsError(t *testing.T) {
	router := &mockRouter{
		provider: &mockProvider{
			response: &aitypes.GenerateResponse{Content: "{}"},
		},
	}
	planner := NewLLMPlanner(router, nil)

	origPlan := &execution.ExecutionPlan{
		AssistantID: "asst-1",
		Steps: []execution.ExecutionStep{
			{Type: execution.StepTypeSkill, Name: "step1"},
		},
	}
	if err := origPlan.Validate(); err != nil {
		t.Fatalf("validate original plan: %v", err)
	}

	rCtx := &ReplanContext{
		OriginalPlan: origPlan,
		FailedStep: execution.ExecutionStep{
			Name: "step1", Type: execution.StepTypeSkill,
		},
		Trigger:         ReplanTriggerToolFailure,
		PreviousResults: map[string]any{},
		PlannerContext: &PlannerContext{
			AssistantID: "asst-1",
			UserRequest: "test",
			Skills:      []aitypes.SkillDescriptor{}, // empty skills
		},
		ErrorMessage: "failed",
	}

	_, err := planner.Replan(context.Background(), rCtx)
	if err != ErrNoSkillsAvailable {
		t.Fatalf("expected ErrNoSkillsAvailable, got %v", err)
	}
}

// TestReplan_NilOriginalPlan_ReturnsError verifies nil original plan returns an error.
func TestReplan_NilOriginalPlan_ReturnsError(t *testing.T) {
	router := &mockRouter{
		provider: &mockProvider{
			response: &aitypes.GenerateResponse{Content: "{}"},
		},
	}
	planner := NewLLMPlanner(router, nil)

	rCtx := &ReplanContext{
		OriginalPlan: nil,
		FailedStep: execution.ExecutionStep{
			Name: "step1", Type: execution.StepTypeSkill,
		},
		Trigger:         ReplanTriggerToolFailure,
		PreviousResults: map[string]any{},
		PlannerContext: &PlannerContext{
			AssistantID: "asst-1",
			UserRequest: "test",
			Skills: []aitypes.SkillDescriptor{
				{ID: "s1", Name: "S1", Description: "skill"},
			},
		},
		ErrorMessage: "failed",
	}

	_, err := planner.Replan(context.Background(), rCtx)
	if err == nil {
		t.Fatal("expected error for nil OriginalPlan, got nil")
	}
	if !strings.Contains(err.Error(), "original plan") {
		t.Errorf("expected error to mention 'original plan', got: %v", err)
	}
}

// TestReplan_NilPlannerContext_ReturnsError verifies nil planner context returns an error.
func TestReplan_NilPlannerContext_ReturnsError(t *testing.T) {
	router := &mockRouter{
		provider: &mockProvider{
			response: &aitypes.GenerateResponse{Content: "{}"},
		},
	}
	planner := NewLLMPlanner(router, nil)

	origPlan := &execution.ExecutionPlan{
		AssistantID: "asst-1",
		Steps: []execution.ExecutionStep{
			{Type: execution.StepTypeSkill, Name: "step1"},
		},
	}
	if err := origPlan.Validate(); err != nil {
		t.Fatalf("validate original plan: %v", err)
	}

	rCtx := &ReplanContext{
		OriginalPlan:    origPlan,
		FailedStep:      execution.ExecutionStep{Name: "step1", Type: execution.StepTypeSkill},
		Trigger:         ReplanTriggerToolFailure,
		PreviousResults: map[string]any{},
		PlannerContext:  nil, // nil planner context
		ErrorMessage:    "failed",
	}

	_, err := planner.Replan(context.Background(), rCtx)
	if err == nil {
		t.Fatal("expected error for nil PlannerContext, got nil")
	}
	if !strings.Contains(err.Error(), "planner context") {
		t.Errorf("expected error to mention 'planner context', got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// capturingMockProvider captures the request for prompt inspection
// ---------------------------------------------------------------------------

type capturingMockProvider struct {
	response *aitypes.GenerateResponse
	err      error
	captureFn func(req aitypes.GenerateRequest)
}

func (m *capturingMockProvider) ID() string                            { return "capture-mock" }
func (m *capturingMockProvider) Capabilities() []aitypes.CapabilityType { return nil }
func (m *capturingMockProvider) Generate(_ context.Context, req aitypes.GenerateRequest) (*aitypes.GenerateResponse, error) {
	if m.captureFn != nil {
		m.captureFn(req)
	}
	return m.response, m.err
}
func (m *capturingMockProvider) Embed(_ context.Context, _ []string) ([][]float32, error) {
	return nil, nil
}

// ---------------------------------------------------------------------------
// Helper: build a valid replan context for reuse across tests
// ---------------------------------------------------------------------------

func newTestReplanContext(skills []aitypes.SkillDescriptor) *ReplanContext {
	origPlan := &execution.ExecutionPlan{
		AssistantID: "asst-1",
		Steps: []execution.ExecutionStep{
			{Type: execution.StepTypeSkill, Name: "step1"},
			{Type: execution.StepTypeTool, Name: "step2"},
		},
	}
	_ = origPlan.Validate()

	return &ReplanContext{
		OriginalPlan: origPlan,
		FailedStep: execution.ExecutionStep{
			Name: "step2", Type: execution.StepTypeTool, StepID: "step2-id",
		},
		Trigger:         ReplanTriggerToolFailure,
		PreviousResults: map[string]any{"step1": "result1"},
		PlannerContext: &PlannerContext{
			AssistantID: "asst-1",
			UserRequest: "test request",
			Skills:      skills,
		},
		ErrorMessage: "test error",
	}
}

// TestReplan_LLMFailure verifies LLM errors are propagated with ErrPlanningFailed wrapping.
func TestReplan_LLMFailure(t *testing.T) {
	router := &mockRouter{
		provider: &mockProvider{
			err: fmt.Errorf("LLM unavailable"),
		},
	}
	planner := NewLLMPlanner(router, nil)

	rCtx := newTestReplanContext([]aitypes.SkillDescriptor{
		{ID: "s1", Name: "S1", Description: "A skill"},
	})

	_, err := planner.Replan(context.Background(), rCtx)
	if err == nil {
		t.Fatal("expected error for LLM failure")
	}
	if !strings.Contains(err.Error(), "LLM unavailable") {
		t.Errorf("expected LLM error message, got: %v", err)
	}
}

// TestReplan_RoutingFailure verifies routing errors are propagated.
func TestReplan_RoutingFailure(t *testing.T) {
	router := &mockRouter{
		err: fmt.Errorf("no provider available"),
	}
	planner := NewLLMPlanner(router, nil)

	rCtx := newTestReplanContext([]aitypes.SkillDescriptor{
		{ID: "s1", Name: "S1", Description: "A skill"},
	})

	_, err := planner.Replan(context.Background(), rCtx)
	if err == nil {
		t.Fatal("expected error for routing failure")
	}
	if !strings.Contains(err.Error(), "routing failed") {
		t.Errorf("expected 'routing failed' in error, got: %v", err)
	}
}

// TestReplan_InvalidLLMResponse verifies parse errors are wrapped correctly.
func TestReplan_InvalidLLMResponse(t *testing.T) {
	router := &mockRouter{
		provider: &mockProvider{
			response: &aitypes.GenerateResponse{Content: "not valid JSON"},
		},
	}
	planner := NewLLMPlanner(router, nil)

	rCtx := newTestReplanContext([]aitypes.SkillDescriptor{
		{ID: "s1", Name: "S1", Description: "A skill"},
	})

	_, err := planner.Replan(context.Background(), rCtx)
	if err == nil {
		t.Fatal("expected error for invalid LLM response")
	}
	if !strings.Contains(err.Error(), "failed to parse") {
		t.Errorf("expected parse error, got: %v", err)
	}
}

// TestReplan_ValidationFailure verifies that LLM-returned plans still go through validation.
func TestReplan_ValidationFailure(t *testing.T) {
	// Return valid JSON but with an empty step name which triggers validation error.
	planJSON := `{"assistant_id":"asst-1","steps":[{"type":"skill","name":"","arguments":{}}]}`
	router := &mockRouter{
		provider: &mockProvider{
			response: &aitypes.GenerateResponse{Content: planJSON},
		},
	}
	planner := NewLLMPlanner(router, nil)

	rCtx := newTestReplanContext([]aitypes.SkillDescriptor{
		{ID: "s1", Name: "S1", Description: "A skill"},
	})

	_, err := planner.Replan(context.Background(), rCtx)
	if err == nil {
		t.Fatal("expected validation error for empty step name")
	}
	if !strings.Contains(err.Error(), "validation failed") {
		t.Errorf("expected 'validation failed', got: %v", err)
	}
}

// TestReplan_PromptContainsOriginalSteps verifies the prompt lists the original plan steps.
func TestReplan_PromptContainsOriginalSteps(t *testing.T) {
	var capturedPrompt string
	planJSON := `{"assistant_id":"","steps":[{"type":"skill","name":"retry","arguments":{}}]}`

	router := &mockRouter{
		provider: &capturingMockProvider{
			response: &aitypes.GenerateResponse{Content: planJSON},
			captureFn: func(req aitypes.GenerateRequest) {
				for _, msg := range req.Messages {
					if msg.Role == "user" {
						for _, cb := range msg.Contents {
							if cb.Type == "text" {
								capturedPrompt = cb.Text
							}
						}
					}
				}
			},
		},
	}
	planner := NewLLMPlanner(router, nil)

	origPlan := &execution.ExecutionPlan{
		AssistantID: "asst-1",
		Steps: []execution.ExecutionStep{
			{Type: execution.StepTypeSkill, Name: "fetch_data"},
			{Type: execution.StepTypeTool, Name: "process_data"},
			{Type: execution.StepTypeLLM, Name: "summarize"},
		},
	}
	_ = origPlan.Validate()

	rCtx := &ReplanContext{
		OriginalPlan: origPlan,
		FailedStep: execution.ExecutionStep{
			Name: "process_data", Type: execution.StepTypeTool,
		},
		Trigger:         ReplanTriggerToolFailure,
		PreviousResults: map[string]any{"fetch_data": "raw data"},
		PlannerContext: &PlannerContext{
			AssistantID: "asst-1",
			UserRequest: "process the data",
			Skills: []aitypes.SkillDescriptor{
				{ID: "retry", Name: "Retry", Description: "Retry"},
			},
		},
		ErrorMessage: "timeout",
	}

	_, err := planner.Replan(context.Background(), rCtx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify prompt contains all original step names
	for _, step := range origPlan.Steps {
		if !strings.Contains(capturedPrompt, step.Name) {
			t.Errorf("replan prompt should contain original step name %q", step.Name)
		}
	}

	// Verify prompt contains previous results
	if !strings.Contains(capturedPrompt, "raw data") {
		t.Error("replan prompt should contain previous result 'raw data'")
	}

	// Verify prompt says not to repeat completed work
	if !strings.Contains(capturedPrompt, "Do NOT repeat") {
		t.Error("replan prompt should instruct not to repeat completed work")
	}
}

// TestReplan_SetsAssistantIDFromContext verifies AssistantID is set from PlannerContext
// when the LLM response doesn't include one.
func TestReplan_SetsAssistantIDFromContext(t *testing.T) {
	// LLM returns empty assistant_id
	planJSON := `{"assistant_id":"","steps":[{"type":"skill","name":"new_step","arguments":{}}]}`

	router := &mockRouter{
		provider: &mockProvider{
			response: &aitypes.GenerateResponse{Content: planJSON},
		},
	}
	planner := NewLLMPlanner(router, nil)

	origPlan := &execution.ExecutionPlan{
		AssistantID: "ctx-assistant",
		Steps:       []execution.ExecutionStep{{Type: execution.StepTypeSkill, Name: "step1"}},
	}
	_ = origPlan.Validate()

	rCtx := &ReplanContext{
		OriginalPlan: origPlan,
		FailedStep:   execution.ExecutionStep{Name: "step1", Type: execution.StepTypeSkill},
		Trigger:      ReplanTriggerToolFailure,
		PlannerContext: &PlannerContext{
			AssistantID: "ctx-assistant",
			UserRequest: "test",
			Skills: []aitypes.SkillDescriptor{
				{ID: "new_step", Name: "New", Description: "new step"},
			},
		},
		ErrorMessage: "failed",
	}

	plan, err := planner.Replan(context.Background(), rCtx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if plan.AssistantID != "ctx-assistant" {
		t.Errorf("expected AssistantID from context 'ctx-assistant', got %q", plan.AssistantID)
	}
}

// TestReplan_EmptySteps_Allowed verifies that if the LLM returns an empty plan
// (the cooperative stop signal), it is returned as-is without validation.
func TestReplan_EmptySteps_Allowed(t *testing.T) {
	planJSON := `{"assistant_id":"asst-1","steps":[]}`

	router := &mockRouter{
		provider: &mockProvider{
			response: &aitypes.GenerateResponse{Content: planJSON},
		},
	}
	planner := NewLLMPlanner(router, nil)

	origPlan := &execution.ExecutionPlan{
		AssistantID: "asst-1",
		Steps:       []execution.ExecutionStep{{Type: execution.StepTypeSkill, Name: "step1"}},
	}
	_ = origPlan.Validate()

	rCtx := &ReplanContext{
		OriginalPlan: origPlan,
		FailedStep:   execution.ExecutionStep{Name: "step1", Type: execution.StepTypeSkill},
		Trigger:      ReplanTriggerToolFailure,
		PlannerContext: &PlannerContext{
			AssistantID: "asst-1",
			UserRequest: "test",
			Skills: []aitypes.SkillDescriptor{
				{ID: "s1", Name: "S1", Description: "skill"},
			},
		},
		ErrorMessage: "failed",
	}

	plan, err := planner.Replan(context.Background(), rCtx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(plan.Steps) != 0 {
		t.Errorf("expected 0 steps for empty plan, got %d", len(plan.Steps))
	}
}

// TestReplan_MultipleStepsWithResults verifies correct handling of multiple previous results.
func TestReplan_MultipleStepsWithResults(t *testing.T) {
	var capturedPrompt string
	planJSON := `{"assistant_id":"","steps":[{"type":"skill","name":"final","arguments":{}}]}`

	router := &mockRouter{
		provider: &capturingMockProvider{
			response: &aitypes.GenerateResponse{Content: planJSON},
			captureFn: func(req aitypes.GenerateRequest) {
				for _, msg := range req.Messages {
					if msg.Role == "user" {
						for _, cb := range msg.Contents {
							if cb.Type == "text" {
								capturedPrompt = cb.Text
							}
						}
					}
				}
			},
		},
	}
	planner := NewLLMPlanner(router, nil)

	origPlan := &execution.ExecutionPlan{
		AssistantID: "asst-1",
		Steps: []execution.ExecutionStep{
			{Type: execution.StepTypeTool, Name: "fetch"},
			{Type: execution.StepTypeSkill, Name: "parse"},
			{Type: execution.StepTypeTool, Name: "transform"},
		},
	}
	_ = origPlan.Validate()

	rCtx := &ReplanContext{
		OriginalPlan: origPlan,
		FailedStep: execution.ExecutionStep{
			Name: "transform", Type: execution.StepTypeTool,
		},
		Trigger: ReplanTriggerToolFailure,
		PreviousResults: map[string]any{
			"fetch": "raw json",
			"parse": map[string]any{"key": "value"},
		},
		PlannerContext: &PlannerContext{
			AssistantID: "asst-1",
			UserRequest: "process data",
			Skills: []aitypes.SkillDescriptor{
				{ID: "final", Name: "Final", Description: "Final step"},
			},
		},
		ErrorMessage: "transformation error",
	}

	_, err := planner.Replan(context.Background(), rCtx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Both previous results should appear in the prompt
	if !strings.Contains(capturedPrompt, "raw json") {
		t.Error("prompt should contain 'fetch' result")
	}
	if !strings.Contains(capturedPrompt, "key") {
		t.Error("prompt should contain 'parse' result")
	}
}

// TestBuildReplanPrompt_Format verifies buildReplanPrompt output structure.
func TestBuildReplanPrompt_Format(t *testing.T) {
	planner := NewLLMPlanner(nil, nil)

	origPlan := &execution.ExecutionPlan{
		AssistantID: "asst-1",
		Steps: []execution.ExecutionStep{
			{Type: execution.StepTypeSkill, Name: "step1"},
		},
	}
	_ = origPlan.Validate()

	rCtx := &ReplanContext{
		OriginalPlan: origPlan,
		FailedStep: execution.ExecutionStep{
			Name: "step1", Type: execution.StepTypeSkill,
		},
		Trigger:         ReplanTriggerInvalidData,
		PreviousResults: map[string]any{},
		PlannerContext: &PlannerContext{
			AssistantID: "asst-1",
			UserRequest: "do stuff",
			Skills: []aitypes.SkillDescriptor{
				{ID: "s1", Name: "S1", Description: "A skill"},
			},
		},
		ErrorMessage: "bad data format",
	}

	prompt := planner.buildReplanPrompt(rCtx)

	// Must contain key sections
	required := []string{
		"previous execution plan failed",
		"step1",
		"bad data format",
		"Available skills/tools",
		"Respond with a JSON object",
	}
	for _, substr := range required {
		if !strings.Contains(prompt, substr) {
			t.Errorf("buildReplanPrompt missing %q", substr)
		}
	}
}

// TestReplan_WithSoulContext verifies soul/personality is included in the replan prompt.
func TestReplan_WithSoulContext(t *testing.T) {
	var capturedPrompt string
	planJSON := fmt.Sprintf(`{"assistant_id":"","steps":[{"type":"skill","name":"s1","arguments":{}}]}`)

	router := &mockRouter{
		provider: &capturingMockProvider{
			response: &aitypes.GenerateResponse{Content: planJSON},
			captureFn: func(req aitypes.GenerateRequest) {
				for _, msg := range req.Messages {
					if msg.Role == "system" {
						for _, cb := range msg.Contents {
							if cb.Type == "text" {
								capturedPrompt = cb.Text
							}
						}
					}
				}
			},
		},
	}
	planner := NewLLMPlanner(router, nil)

	origPlan := &execution.ExecutionPlan{
		AssistantID: "asst-1",
		Steps:       []execution.ExecutionStep{{Type: execution.StepTypeSkill, Name: "step1"}},
	}
	_ = origPlan.Validate()

	rCtx := &ReplanContext{
		OriginalPlan: origPlan,
		FailedStep:   execution.ExecutionStep{Name: "step1", Type: execution.StepTypeSkill},
		Trigger:      ReplanTriggerToolFailure,
		PlannerContext: &PlannerContext{
			AssistantID: "asst-1",
			UserRequest: "test",
			Soul: AssistantSoul{
				SystemPrompt: "You are a helpful medical assistant",
				Personality:  "Precise and cautious",
			},
			Skills: []aitypes.SkillDescriptor{
				{ID: "s1", Name: "S1", Description: "skill"},
			},
		},
		ErrorMessage: "failed",
	}

	_, err := planner.Replan(context.Background(), rCtx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// System prompt should contain the soul system prompt
	if !strings.Contains(capturedPrompt, "You are a helpful medical assistant") {
		t.Error("replan system prompt should contain soul system prompt")
	}
}

// JSON round-trip helper for plan construction in tests
func mustMarshalSteps(steps []execution.ExecutionStep) string {
	type planJSON struct {
		AssistantID string                   `json:"assistant_id"`
		Steps       []map[string]interface{} `json:"steps"`
	}
	var jsonSteps []map[string]interface{}
	for _, s := range steps {
		jsonSteps = append(jsonSteps, map[string]interface{}{
			"type":      string(s.Type),
			"name":      s.Name,
			"arguments": s.Arguments,
		})
	}
	b, err := json.Marshal(planJSON{AssistantID: "", Steps: jsonSteps})
	if err != nil {
		panic(err)
	}
	return string(b)
}
