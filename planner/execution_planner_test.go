package planner

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/openbotstack/openbotstack-core/ai/providers"

	aitypes "github.com/openbotstack/openbotstack-core/ai/types"
	"github.com/openbotstack/openbotstack-core/execution"
)

// ---------------------------------------------------------------------------
// Existing tests (preserved)
// ---------------------------------------------------------------------------

func TestValidator_Valid(t *testing.T) {
	v := NewValidator(nil) // default limits

	plan := &execution.ExecutionPlan{
		AssistantID: "assistant-1",
		Steps: []execution.ExecutionStep{
			{Type: execution.StepTypeSkill, Name: "data_summary", Arguments: map[string]any{"id": "123"}},
			{Type: execution.StepTypeTool, Name: "db_query", Arguments: map[string]any{"record_id": "123"}},
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

// Basic structural test preserved
func TestLLMPlanner(t *testing.T) {
	planner := NewLLMPlanner(nil, nil)
	prompt := planner.buildPrompt(&PlannerContext{
		AssistantID: "a1",
		UserRequest: "hello",
		Skills: []aitypes.SkillDescriptor{
			{ID: "skill1", Name: "Skill 1", Description: "A skill"},
		},
	})

	if prompt == "" {
		t.Fatal("expected prompt, got empty string")
	}

	ctx := context.Background()
	_, err := planner.Plan(ctx, &PlannerContext{})

	if err != ErrNoSkillsAvailable {
		t.Fatalf("expected ErrNoSkillsAvailable, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// Validator tests — additional coverage
// ---------------------------------------------------------------------------

func TestValidator_NilPlan(t *testing.T) {
	v := NewValidator(nil)
	if err := v.Validate(nil); err != ErrNilPlan {
		t.Fatalf("expected ErrNilPlan, got %v", err)
	}
}

func TestValidator_DefaultLimits(t *testing.T) {
	limits := DefaultLimits()
	if limits.MaxSteps != 10 {
		t.Errorf("expected MaxSteps=10, got %d", limits.MaxSteps)
	}
	if limits.MaxToolCalls != 15 {
		t.Errorf("expected MaxToolCalls=15, got %d", limits.MaxToolCalls)
	}
	if limits.MaxExecutionTime != 300*time.Second {
		t.Errorf("expected MaxExecutionTime=300s, got %v", limits.MaxExecutionTime)
	}
}

func TestValidator_ValidPlanWithinCustomLimits(t *testing.T) {
	v := NewValidator(&ExecutionLimits{
		MaxSteps:         5,
		MaxToolCalls:     3,
		MaxExecutionTime: 30 * time.Second,
	})

	plan := &execution.ExecutionPlan{
		AssistantID: "asst-1",
		Steps: []execution.ExecutionStep{
			{Type: execution.StepTypeSkill, Name: "s1"},
			{Type: execution.StepTypeTool, Name: "t1"},
			{Type: execution.StepTypeSkill, Name: "s2"},
		},
	}

	if err := v.Validate(plan); err != nil {
		t.Fatalf("expected valid, got %v", err)
	}
}

func TestValidator_MaxStepsExceeded(t *testing.T) {
	v := NewValidator(&ExecutionLimits{
		MaxSteps:         2,
		MaxToolCalls:     10,
		MaxExecutionTime: time.Second,
	})

	plan := &execution.ExecutionPlan{
		AssistantID: "asst",
		Steps: []execution.ExecutionStep{
			{Type: execution.StepTypeSkill, Name: "s1"},
			{Type: execution.StepTypeSkill, Name: "s2"},
			{Type: execution.StepTypeSkill, Name: "s3"},
		},
	}

	err := v.Validate(plan)
	if err == nil {
		t.Fatal("expected error for too many steps")
	}
	if !strings.Contains(err.Error(), "max 2") {
		t.Errorf("expected error message to contain 'max 2', got: %v", err)
	}
}

func TestValidator_MaxToolCallsExceeded(t *testing.T) {
	v := NewValidator(&ExecutionLimits{
		MaxSteps:         10,
		MaxToolCalls:     1,
		MaxExecutionTime: time.Second,
	})

	plan := &execution.ExecutionPlan{
		AssistantID: "asst",
		Steps: []execution.ExecutionStep{
			{Type: execution.StepTypeTool, Name: "t1"},
			{Type: execution.StepTypeTool, Name: "t2"},
		},
	}

	err := v.Validate(plan)
	if err == nil {
		t.Fatal("expected error for too many tool calls")
	}
	if !strings.Contains(err.Error(), "max 1") {
		t.Errorf("expected error message to contain 'max 1', got: %v", err)
	}
}

func TestValidator_EmptyStepsInvalid(t *testing.T) {
	v := NewValidator(nil)
	plan := &execution.ExecutionPlan{
		AssistantID: "asst",
		Steps:       []execution.ExecutionStep{},
	}
	err := v.Validate(plan)
	if err == nil {
		t.Fatal("expected error for empty steps, got nil")
	}
	if err != ErrEmptySteps {
		t.Errorf("expected ErrEmptySteps, got: %v", err)
	}
}

func TestValidator_StepErrorsContainIndex(t *testing.T) {
	v := NewValidator(nil)

	t.Run("empty name at index", func(t *testing.T) {
		plan := &execution.ExecutionPlan{
			AssistantID: "asst",
			Steps: []execution.ExecutionStep{
				{Type: execution.StepTypeSkill, Name: "valid"},
				{Type: execution.StepTypeSkill, Name: ""},
			},
		}
		err := v.Validate(plan)
		if err == nil {
			t.Fatal("expected error")
		}
		if !strings.Contains(err.Error(), "index 1") {
			t.Errorf("expected error to reference index 1, got: %v", err)
		}
	})

	t.Run("invalid type at index", func(t *testing.T) {
		plan := &execution.ExecutionPlan{
			AssistantID: "asst",
			Steps: []execution.ExecutionStep{
				{Type: execution.StepType("bad"), Name: "s1"},
			},
		}
		err := v.Validate(plan)
		if err == nil {
			t.Fatal("expected error")
		}
		if !strings.Contains(err.Error(), "index 0") {
			t.Errorf("expected error to reference index 0, got: %v", err)
		}
	})
}

// ---------------------------------------------------------------------------
// buildPrompt tests
// ---------------------------------------------------------------------------

func TestBuildPrompt_WithSoul(t *testing.T) {
	p := NewLLMPlanner(nil, nil)
	prompt := p.buildPrompt(&PlannerContext{
		AssistantID: "a1",
		UserRequest: "summarize the report",
		Soul: AssistantSoul{
			Personality:  "Friendly and precise",
			Instructions: "Always be concise",
		},
		Skills: []aitypes.SkillDescriptor{
			{ID: "summarize", Name: "Summarize", Description: "Summarizes text"},
		},
	})

	if !strings.Contains(prompt, "Personality: Friendly and precise") {
		t.Error("expected prompt to contain personality")
	}
	if !strings.Contains(prompt, "Specific Instructions:\nAlways be concise") {
		t.Error("expected prompt to contain instructions")
	}
}

func TestBuildPrompt_WithMemoryContext(t *testing.T) {
	p := NewLLMPlanner(nil, nil)
	prompt := p.buildPrompt(&PlannerContext{
		AssistantID: "a1",
		UserRequest: "what did we discuss?",
		MemoryContext: []SearchResult{
			{Content: []byte("user likes Go"), Score: 0.9},
			{Content: []byte("user prefers dark mode"), Score: 0.8},
		},
		Skills: []aitypes.SkillDescriptor{
			{ID: "s1", Name: "S1", Description: "skill 1"},
		},
	})

	if !strings.Contains(prompt, "Relevant Memory Context:") {
		t.Error("expected prompt to contain memory context header")
	}
	if !strings.Contains(prompt, "- user likes Go") {
		t.Error("expected prompt to contain first memory")
	}
	if !strings.Contains(prompt, "- user prefers dark mode") {
		t.Error("expected prompt to contain second memory")
	}
}

func TestBuildPrompt_EmptySkillsList(t *testing.T) {
	p := NewLLMPlanner(nil, nil)
	prompt := p.buildPrompt(&PlannerContext{
		AssistantID: "a1",
		UserRequest: "do something",
		Skills:      []aitypes.SkillDescriptor{},
	})

	if !strings.Contains(prompt, "Available skills/tools:") {
		t.Error("expected prompt to contain skills header even when empty")
	}
}

func TestBuildPrompt_WithInputSchema(t *testing.T) {
	p := NewLLMPlanner(nil, nil)
	prompt := p.buildPrompt(&PlannerContext{
		AssistantID: "a1",
		UserRequest: "calculate tax",
		Skills: []aitypes.SkillDescriptor{
			{
				ID:          "tax-calc",
				Name:        "Tax Calculator",
				Description: "Calculates tax",
				InputSchema: &aitypes.JSONSchema{
					Type: "object",
					Properties: map[string]*aitypes.JSONSchema{
						"income": {Type: "number"},
					},
					Required: []string{"income"},
				},
			},
		},
	})

	// Planner exposes parameter names/types so the LLM generates correct arguments
	if !strings.Contains(prompt, "Params:") {
		t.Error("planner should contain Params section from InputSchema")
	}
	if !strings.Contains(prompt, "income:") {
		t.Error("planner should expose schema parameter 'income'")
	}
	if !strings.Contains(prompt, "Tax Calculator") {
		t.Error("expected prompt to contain skill name")
	}
	if !strings.Contains(prompt, "Calculates tax") {
		t.Error("expected prompt to contain skill description")
	}
}

func TestBuildPrompt_WithNilInputSchema(t *testing.T) {
	p := NewLLMPlanner(nil, nil)
	prompt := p.buildPrompt(&PlannerContext{
		AssistantID: "a1",
		UserRequest: "hello",
		Skills: []aitypes.SkillDescriptor{
			{
				ID:          "hello",
				Name:        "Hello",
				Description: "Says hello",
				InputSchema: nil,
			},
		},
	})

	if !strings.Contains(prompt, "- hello (Hello): Says hello") {
		t.Error("expected prompt to contain skill listing even with nil InputSchema")
	}
}

func TestBuildPrompt_ContainsUserRequest(t *testing.T) {
	p := NewLLMPlanner(nil, nil)
	prompt := p.buildPrompt(&PlannerContext{
		AssistantID: "a1",
		UserRequest: "analyze the dataset",
		Skills: []aitypes.SkillDescriptor{
			{ID: "s1", Name: "S1", Description: "A skill"},
		},
	})

	if !strings.Contains(prompt, "<user_request>\nanalyze the dataset\n</user_request>") {
		t.Error("expected prompt to contain user request in XML boundary tags")
	}
}

func TestBuildPrompt_ContainsJSONFormatInstructions(t *testing.T) {
	p := NewLLMPlanner(nil, nil)
	prompt := p.buildPrompt(&PlannerContext{
		AssistantID: "a1",
		UserRequest: "test",
		Skills: []aitypes.SkillDescriptor{
			{ID: "s1", Name: "S1", Description: "A skill"},
		},
	})

	if !strings.Contains(prompt, "Respond with a JSON object") {
		t.Error("expected prompt to contain JSON format instructions")
	}
	if !strings.Contains(prompt, "/no_think") {
		t.Error("expected prompt to contain /no_think")
	}
}

func TestBuildPrompt_EmptySoulFields(t *testing.T) {
	p := NewLLMPlanner(nil, nil)
	prompt := p.buildPrompt(&PlannerContext{
		AssistantID: "a1",
		UserRequest: "test",
		Soul:        AssistantSoul{}, // all empty
		Skills: []aitypes.SkillDescriptor{
			{ID: "s1", Name: "S1", Description: "A skill"},
		},
	})

	if strings.Contains(prompt, "Personality:") {
		t.Error("expected no personality section when empty")
	}
	if strings.Contains(prompt, "Specific Instructions:") {
		t.Error("expected no instructions section when empty")
	}
}

// ---------------------------------------------------------------------------
// parseResponse tests
// ---------------------------------------------------------------------------

func TestParseResponse_ValidJSON(t *testing.T) {
	p := NewLLMPlanner(nil, nil)

	input := `{"assistant_id":"a1","steps":[{"type":"skill","name":"summarize","arguments":{"text":"hello"}}]}`
	plan, err := p.parseResponse(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if plan.AssistantID != "a1" {
		t.Errorf("expected assistant_id=a1, got %s", plan.AssistantID)
	}
	if len(plan.Steps) != 1 {
		t.Fatalf("expected 1 step, got %d", len(plan.Steps))
	}
	if plan.Steps[0].Name != "summarize" {
		t.Errorf("expected step name=summarize, got %s", plan.Steps[0].Name)
	}
}

func TestParseResponse_JSONInMarkdownCodeBlock(t *testing.T) {
	p := NewLLMPlanner(nil, nil)

	input := "```json\n{\"assistant_id\":\"a1\",\"steps\":[{\"type\":\"skill\",\"name\":\"s1\",\"arguments\":{}}]}\n```"
	plan, err := p.parseResponse(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if plan.AssistantID != "a1" {
		t.Errorf("expected assistant_id=a1, got %s", plan.AssistantID)
	}
}

func TestParseResponse_JSONInGenericCodeBlock(t *testing.T) {
	p := NewLLMPlanner(nil, nil)

	input := "```\n{\"assistant_id\":\"a1\",\"steps\":[{\"type\":\"tool\",\"name\":\"t1\",\"arguments\":{}}]}\n```"
	plan, err := p.parseResponse(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if plan.AssistantID != "a1" {
		t.Errorf("expected assistant_id=a1, got %s", plan.AssistantID)
	}
	if plan.Steps[0].Type != execution.StepTypeTool {
		t.Errorf("expected step type=tool, got %s", plan.Steps[0].Type)
	}
}

func TestParseResponse_EmptyResponse(t *testing.T) {
	p := NewLLMPlanner(nil, nil)

	_, err := p.parseResponse("")
	if err == nil {
		t.Fatal("expected error for empty response")
	}
	if !strings.Contains(err.Error(), "invalid json") {
		t.Errorf("expected 'invalid json' in error, got: %v", err)
	}
}

func TestParseResponse_InvalidJSON(t *testing.T) {
	p := NewLLMPlanner(nil, nil)

	_, err := p.parseResponse("this is not json at all")
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
	if !strings.Contains(err.Error(), "invalid json") {
		t.Errorf("expected 'invalid json' in error, got: %v", err)
	}
}

func TestParseResponse_EmptyStepsArray(t *testing.T) {
	p := NewLLMPlanner(nil, nil)

	input := `{"assistant_id":"a1","steps":[]}`
	plan, err := p.parseResponse(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(plan.Steps) != 0 {
		t.Errorf("expected 0 steps, got %d", len(plan.Steps))
	}
}

func TestParseResponse_WithAssistantID(t *testing.T) {
	p := NewLLMPlanner(nil, nil)

	input := `{"assistant_id":"my-assistant","steps":[{"type":"skill","name":"s1","arguments":{}}]}`
	plan, err := p.parseResponse(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if plan.AssistantID != "my-assistant" {
		t.Errorf("expected assistant_id=my-assistant, got %s", plan.AssistantID)
	}
}

func TestParseResponse_WhitespaceOnly(t *testing.T) {
	p := NewLLMPlanner(nil, nil)

	_, err := p.parseResponse("   \n\t  ")
	if err == nil {
		t.Fatal("expected error for whitespace-only response")
	}
}

func TestParseResponse_JSONWithLeadingTrailingSpaces(t *testing.T) {
	p := NewLLMPlanner(nil, nil)

	input := `   {"assistant_id":"a1","steps":[{"type":"skill","name":"s1","arguments":{}}]}   `
	plan, err := p.parseResponse(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if plan.AssistantID != "a1" {
		t.Errorf("expected assistant_id=a1, got %s", plan.AssistantID)
	}
}

func TestParseResponse_CodeBlockWithExtraContent(t *testing.T) {
	p := NewLLMPlanner(nil, nil)

	// parseResponse now extracts JSON from within text (supports thinking models)
	input := "Here is the plan:\n```json\n{\"assistant_id\":\"a1\",\"steps\":[{\"type\":\"skill\",\"name\":\"s1\",\"arguments\":{}}]}\n```"
	plan, err := p.parseResponse(input)
	if err != nil {
		t.Fatalf("should extract JSON from text with code block: %v", err)
	}
	if plan.AssistantID != "a1" {
		t.Errorf("assistant_id = %q, want %q", plan.AssistantID, "a1")
	}
}

func TestParseResponse_PlainJSONWithReasoningField(t *testing.T) {
	p := NewLLMPlanner(nil, nil)

	input := `{"assistant_id":"a1","steps":[{"type":"skill","name":"s1","arguments":{}}],"reasoning":"User needs a summary"}`
	plan, err := p.parseResponse(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if plan.Reasoning != "User needs a summary" {
		t.Errorf("expected reasoning to be preserved, got: %s", plan.Reasoning)
	}
}

// ---------------------------------------------------------------------------
// Plan method tests (using mock router and provider)
// ---------------------------------------------------------------------------

// mockProvider implements providers.ModelProvider
type mockProvider struct {
	id           string
	capabilities []aitypes.CapabilityType
	response     *aitypes.GenerateResponse
	err          error
}

func (m *mockProvider) ID() string                            { return m.id }
func (m *mockProvider) Capabilities() []aitypes.CapabilityType { return m.capabilities }
func (m *mockProvider) Generate(_ context.Context, _ aitypes.GenerateRequest) (*aitypes.GenerateResponse, error) {
	return m.response, m.err
}
func (m *mockProvider) Embed(_ context.Context, _ []string) ([][]float32, error) {
	return nil, nil
}

// mockRouter implements providers.ModelRouter
type mockRouter struct {
	provider providers.ModelProvider
	err      error
}

func (m *mockRouter) Route(_ []aitypes.CapabilityType, _ aitypes.ModelConstraints) (providers.ModelProvider, error) {
	return m.provider, m.err
}

func (m *mockRouter) Register(_ providers.ModelProvider) error { return nil }
func (m *mockRouter) List() []string                           { return nil }

func TestPlan_NoSkills(t *testing.T) {
	p := NewLLMPlanner(nil, nil)
	_, err := p.Plan(context.Background(), &PlannerContext{
		AssistantID: "a1",
		Skills:      []aitypes.SkillDescriptor{},
	})
	if err != ErrNoSkillsAvailable {
		t.Fatalf("expected ErrNoSkillsAvailable, got %v", err)
	}
}

func TestPlan_RoutingFailure(t *testing.T) {
	router := &mockRouter{
		err: fmt.Errorf("no provider available"),
	}
	p := NewLLMPlanner(router, nil)
	_, err := p.Plan(context.Background(), &PlannerContext{
		AssistantID: "a1",
		Skills: []aitypes.SkillDescriptor{
			{ID: "s1", Name: "S1", Description: "A skill"},
		},
	})
	if err == nil {
		t.Fatal("expected error for routing failure")
	}
	if !strings.Contains(err.Error(), "routing failed") {
		t.Errorf("expected 'routing failed' in error, got: %v", err)
	}
}

func TestPlan_LLMFailure(t *testing.T) {
	router := &mockRouter{
		provider: &mockProvider{
			err: fmt.Errorf("LLM service unavailable"),
		},
	}
	p := NewLLMPlanner(router, nil)
	_, err := p.Plan(context.Background(), &PlannerContext{
		AssistantID: "a1",
		Skills: []aitypes.SkillDescriptor{
			{ID: "s1", Name: "S1", Description: "A skill"},
		},
	})
	if err == nil {
		t.Fatal("expected error for LLM failure")
	}
	if !strings.Contains(err.Error(), "LLM service unavailable") {
		t.Errorf("expected LLM error message, got: %v", err)
	}
}

func TestPlan_InvalidLLMResponse(t *testing.T) {
	router := &mockRouter{
		provider: &mockProvider{
			response: &aitypes.GenerateResponse{
				Content: "this is not valid JSON",
			},
		},
	}
	p := NewLLMPlanner(router, nil)
	_, err := p.Plan(context.Background(), &PlannerContext{
		AssistantID: "a1",
		Skills: []aitypes.SkillDescriptor{
			{ID: "s1", Name: "S1", Description: "A skill"},
		},
	})
	if err == nil {
		t.Fatal("expected error for invalid LLM response")
	}
	if !strings.Contains(err.Error(), "failed to parse LLM response") {
		t.Errorf("expected parse error message, got: %v", err)
	}
}

func TestPlan_ValidationFailure(t *testing.T) {
	// Return valid JSON but with an empty step name which triggers validation error.
	// Note: empty assistant_id in response gets replaced by pCtx.AssistantID before
	// validation, so we use an empty step name instead to trigger validation.
	planJSON := `{"assistant_id":"a1","steps":[{"type":"skill","name":"","arguments":{}}]}`
	router := &mockRouter{
		provider: &mockProvider{
			response: &aitypes.GenerateResponse{
				Content: planJSON,
			},
		},
	}
	p := NewLLMPlanner(router, nil)
	_, err := p.Plan(context.Background(), &PlannerContext{
		AssistantID: "a1",
		Skills: []aitypes.SkillDescriptor{
			{ID: "s1", Name: "S1", Description: "A skill"},
		},
	})
	if err == nil {
		t.Fatal("expected validation error (empty step name)")
	}
	if !strings.Contains(err.Error(), "validation failed") {
		t.Errorf("expected 'validation failed', got: %v", err)
	}
}

func TestPlan_Success(t *testing.T) {
	planJSON := `{"assistant_id":"","steps":[{"type":"skill","name":"summarize","arguments":{"text":"hello"}}]}`
	router := &mockRouter{
		provider: &mockProvider{
			response: &aitypes.GenerateResponse{
				Content: planJSON,
			},
		},
	}
	p := NewLLMPlanner(router, nil)
	plan, err := p.Plan(context.Background(), &PlannerContext{
		AssistantID: "a1",
		Skills: []aitypes.SkillDescriptor{
			{ID: "summarize", Name: "Summarize", Description: "Summarizes text"},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if plan.AssistantID != "a1" {
		t.Errorf("expected assistant_id to be set from context (a1), got %s", plan.AssistantID)
	}
	if len(plan.Steps) != 1 {
		t.Fatalf("expected 1 step, got %d", len(plan.Steps))
	}
}

func TestPlan_AssistantIDPreservedFromResponse(t *testing.T) {
	planJSON := `{"assistant_id":"response-assistant","steps":[{"type":"skill","name":"s1","arguments":{}}]}`
	router := &mockRouter{
		provider: &mockProvider{
			response: &aitypes.GenerateResponse{
				Content: planJSON,
			},
		},
	}
	p := NewLLMPlanner(router, nil)
	plan, err := p.Plan(context.Background(), &PlannerContext{
		AssistantID: "context-assistant",
		Skills: []aitypes.SkillDescriptor{
			{ID: "s1", Name: "S1", Description: "A skill"},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// When response has assistant_id, it should be preserved (not overwritten by context)
	if plan.AssistantID != "response-assistant" {
		t.Errorf("expected response-assistant, got %s", plan.AssistantID)
	}
}

func TestPlan_SuccessWithMarkdownResponse(t *testing.T) {
	planJSON := "```json\n{\"assistant_id\":\"a1\",\"steps\":[{\"type\":\"skill\",\"name\":\"s1\",\"arguments\":{}}]}\n```"
	router := &mockRouter{
		provider: &mockProvider{
			response: &aitypes.GenerateResponse{
				Content: planJSON,
			},
		},
	}
	p := NewLLMPlanner(router, nil)
	plan, err := p.Plan(context.Background(), &PlannerContext{
		AssistantID: "a1",
		Skills: []aitypes.SkillDescriptor{
			{ID: "s1", Name: "S1", Description: "A skill"},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if plan.AssistantID != "a1" {
		t.Errorf("expected assistant_id=a1, got %s", plan.AssistantID)
	}
}

func TestPlan_ValidationFailsTooManySteps(t *testing.T) {
	// Build a plan with 3 steps but configure limits to allow only 2
	steps := make([]map[string]any, 3)
	for i := range steps {
		steps[i] = map[string]any{
			"type":      "skill",
			"name":      fmt.Sprintf("s%d", i),
			"arguments": map[string]any{},
		}
	}
	stepsJSON, _ := json.Marshal(steps)
	planJSON := fmt.Sprintf(`{"assistant_id":"a1","steps":%s}`, string(stepsJSON))

	router := &mockRouter{
		provider: &mockProvider{
			response: &aitypes.GenerateResponse{
				Content: planJSON,
			},
		},
	}
	limits := &ExecutionLimits{
		MaxSteps:         2,
		MaxToolCalls:     10,
		MaxExecutionTime: 10 * time.Second,
	}
	p := NewLLMPlanner(router, limits)
	_, err := p.Plan(context.Background(), &PlannerContext{
		AssistantID: "a1",
		Skills: []aitypes.SkillDescriptor{
			{ID: "s1", Name: "S1", Description: "A skill"},
		},
	})
	if err == nil {
		t.Fatal("expected validation error for too many steps")
	}
	if !strings.Contains(err.Error(), "validation failed") {
		t.Errorf("expected 'validation failed', got: %v", err)
	}
}

// capturingProvider captures the GenerateRequest for inspection.
type capturingProvider struct {
	response *aitypes.GenerateResponse
	captured *aitypes.GenerateRequest
}

func (c *capturingProvider) ID() string                            { return "capture" }
func (c *capturingProvider) Capabilities() []aitypes.CapabilityType {
	return []aitypes.CapabilityType{aitypes.CapTextGeneration}
}
func (c *capturingProvider) Generate(_ context.Context, req aitypes.GenerateRequest) (*aitypes.GenerateResponse, error) {
	c.captured = &req
	return c.response, nil
}
func (c *capturingProvider) Embed(_ context.Context, _ []string) ([][]float32, error) {
	return nil, nil
}

func textContent(t *testing.T, m aitypes.Message) string {
	t.Helper()
	for _, c := range m.Contents {
		if c.Type == "text" {
			return c.Text
		}
	}
	return ""
}

func TestPlan_ConversationHistoryInjected(t *testing.T) {
	history := []aitypes.Message{
		{Role: "user", Contents: []aitypes.ContentBlock{aitypes.NewTextBlock("previous question")}},
		{Role: "assistant", Contents: []aitypes.ContentBlock{aitypes.NewTextBlock("previous answer")}},
	}

	planJSON := `{"assistant_id":"a1","steps":[{"type":"llm","name":"respond","arguments":{"prompt":"summary"}}]}`
	cp := &capturingProvider{
		response: &aitypes.GenerateResponse{Content: planJSON},
	}

	router := &mockRouter{provider: cp}
	p := NewLLMPlanner(router, nil)

	_, err := p.Plan(context.Background(), &PlannerContext{
		AssistantID:         "a1",
		Skills:              []aitypes.SkillDescriptor{{ID: "s1", Name: "S1", Description: "A skill"}},
		UserRequest:         "what did we discuss?",
		Soul:                AssistantSoul{SystemPrompt: "you are helpful"},
		ConversationHistory: history,
	})
	if err != nil {
		t.Fatalf("Plan failed: %v", err)
	}

	msgs := cp.captured.Messages
	if len(msgs) != 4 {
		t.Fatalf("expected 4 messages (system + 2 history + user), got %d", len(msgs))
	}
	if msgs[0].Role != "system" {
		t.Errorf("msg[0]: expected system, got %s", msgs[0].Role)
	}
	if msgs[1].Role != "user" || textContent(t, msgs[1]) != "previous question" {
		t.Errorf("msg[1]: expected history user, got role=%s content=%s", msgs[1].Role, textContent(t, msgs[1]))
	}
	if msgs[2].Role != "assistant" || textContent(t, msgs[2]) != "previous answer" {
		t.Errorf("msg[2]: expected history assistant, got role=%s content=%s", msgs[2].Role, textContent(t, msgs[2]))
	}
	if msgs[3].Role != "user" {
		t.Errorf("msg[3]: expected user, got %s", msgs[3].Role)
	}
}

func TestPlan_ConversationHistoryFiltersSystemRole(t *testing.T) {
	history := []aitypes.Message{
		{Role: "system", Contents: []aitypes.ContentBlock{aitypes.NewTextBlock("old system prompt")}},
		{Role: "user", Contents: []aitypes.ContentBlock{aitypes.NewTextBlock("hello")}},
	}

	planJSON := `{"assistant_id":"a1","steps":[{"type":"llm","name":"respond","arguments":{"prompt":"hi"}}]}`
	cp := &capturingProvider{
		response: &aitypes.GenerateResponse{Content: planJSON},
	}

	router := &mockRouter{provider: cp}
	p := NewLLMPlanner(router, nil)

	_, err := p.Plan(context.Background(), &PlannerContext{
		AssistantID:         "a1",
		Skills:              []aitypes.SkillDescriptor{{ID: "s1", Name: "S1", Description: "A skill"}},
		Soul:                AssistantSoul{SystemPrompt: "current system prompt"},
		ConversationHistory: history,
	})
	if err != nil {
		t.Fatalf("Plan failed: %v", err)
	}

	msgs := cp.captured.Messages
	if len(msgs) != 3 {
		t.Fatalf("expected 3 messages (system + filtered history user + user), got %d", len(msgs))
	}
	if msgs[0].Role != "system" {
		t.Errorf("msg[0]: expected system, got %s", msgs[0].Role)
	}
	if msgs[1].Role != "user" || textContent(t, msgs[1]) != "hello" {
		t.Errorf("msg[1]: expected history user 'hello', got role=%s content=%s", msgs[1].Role, textContent(t, msgs[1]))
	}
	if msgs[2].Role != "user" {
		t.Errorf("msg[2]: expected current user, got %s", msgs[2].Role)
	}
}

func TestPlan_NilHistory_SameBehavior(t *testing.T) {
	planJSON := `{"assistant_id":"a1","steps":[{"type":"llm","name":"respond","arguments":{"prompt":"ok"}}]}`
	cp := &capturingProvider{
		response: &aitypes.GenerateResponse{Content: planJSON},
	}

	router := &mockRouter{provider: cp}
	p := NewLLMPlanner(router, nil)

	_, err := p.Plan(context.Background(), &PlannerContext{
		AssistantID: "a1",
		Skills:      []aitypes.SkillDescriptor{{ID: "s1", Name: "S1", Description: "A skill"}},
		Soul:        AssistantSoul{SystemPrompt: "prompt"},
	})
	if err != nil {
		t.Fatalf("Plan failed: %v", err)
	}

	msgs := cp.captured.Messages
	if len(msgs) != 2 {
		t.Fatalf("expected 2 messages (no history), got %d", len(msgs))
	}
}

func TestReplan_ConversationHistoryInjected(t *testing.T) {
	history := []aitypes.Message{
		{Role: "user", Contents: []aitypes.ContentBlock{aitypes.NewTextBlock("what is 2+2?")}},
		{Role: "assistant", Contents: []aitypes.ContentBlock{aitypes.NewTextBlock("4")}},
	}

	planJSON := `{"assistant_id":"a1","steps":[{"type":"skill","name":"retry","arguments":{}}]}`
	cp := &capturingProvider{
		response: &aitypes.GenerateResponse{Content: planJSON},
	}

	router := &mockRouter{provider: cp}
	p := NewLLMPlanner(router, nil)

	origPlan := &execution.ExecutionPlan{
		AssistantID: "a1",
		Steps:       []execution.ExecutionStep{{Type: execution.StepTypeSkill, Name: "step1"}},
	}
	if err := origPlan.Validate(); err != nil {
		t.Fatalf("validate: %v", err)
	}

	rCtx := &ReplanContext{
		OriginalPlan: origPlan,
		FailedStep:   execution.ExecutionStep{Name: "step1", StepID: "s1"},
		Trigger:      ReplanTriggerToolFailure,
		PlannerContext: &PlannerContext{
			AssistantID:         "a1",
			Skills:              []aitypes.SkillDescriptor{{ID: "s1", Name: "S1", Description: "A skill"}},
			Soul:                AssistantSoul{SystemPrompt: "you are helpful"},
			ConversationHistory: history,
		},
	}

	_, err := p.Replan(context.Background(), rCtx)
	if err != nil {
		t.Fatalf("Replan failed: %v", err)
	}

	msgs := cp.captured.Messages
	// Expected: [system, history_user, history_assistant, user(replan prompt)]
	if len(msgs) != 4 {
		t.Fatalf("expected 4 messages (system + 2 history + user), got %d", len(msgs))
	}
	if msgs[0].Role != "system" {
		t.Errorf("msg[0]: expected system, got %s", msgs[0].Role)
	}
	if msgs[1].Role != "user" || textContent(t, msgs[1]) != "what is 2+2?" {
		t.Errorf("msg[1]: expected history user, got role=%s content=%s", msgs[1].Role, textContent(t, msgs[1]))
	}
	if msgs[2].Role != "assistant" || textContent(t, msgs[2]) != "4" {
		t.Errorf("msg[2]: expected history assistant, got role=%s content=%s", msgs[2].Role, textContent(t, msgs[2]))
	}
	if msgs[3].Role != "user" {
		t.Errorf("msg[3]: expected replan user prompt, got %s", msgs[3].Role)
	}
}

func TestReplan_ConversationHistoryFiltersSystemRole(t *testing.T) {
	history := []aitypes.Message{
		{Role: "system", Contents: []aitypes.ContentBlock{aitypes.NewTextBlock("old system prompt")}},
		{Role: "user", Contents: []aitypes.ContentBlock{aitypes.NewTextBlock("hello")}},
	}

	planJSON := `{"assistant_id":"a1","steps":[{"type":"skill","name":"retry","arguments":{}}]}`
	cp := &capturingProvider{
		response: &aitypes.GenerateResponse{Content: planJSON},
	}

	router := &mockRouter{provider: cp}
	p := NewLLMPlanner(router, nil)

	origPlan := &execution.ExecutionPlan{
		AssistantID: "a1",
		Steps:       []execution.ExecutionStep{{Type: execution.StepTypeSkill, Name: "step1"}},
	}
	if err := origPlan.Validate(); err != nil {
		t.Fatalf("validate: %v", err)
	}

	rCtx := &ReplanContext{
		OriginalPlan: origPlan,
		FailedStep:   execution.ExecutionStep{Name: "step1", StepID: "s1"},
		Trigger:      ReplanTriggerToolFailure,
		PlannerContext: &PlannerContext{
			AssistantID:         "a1",
			Skills:              []aitypes.SkillDescriptor{{ID: "s1", Name: "S1", Description: "A skill"}},
			Soul:                AssistantSoul{SystemPrompt: "current prompt"},
			ConversationHistory: history,
		},
	}

	_, err := p.Replan(context.Background(), rCtx)
	if err != nil {
		t.Fatalf("Replan failed: %v", err)
	}

	msgs := cp.captured.Messages
	// system from history should be filtered: [system, user(history), user(replan)]
	if len(msgs) != 3 {
		t.Fatalf("expected 3 messages (system + filtered history user + replan user), got %d", len(msgs))
	}
	if msgs[0].Role != "system" {
		t.Errorf("msg[0]: expected system, got %s", msgs[0].Role)
	}
	if msgs[1].Role != "user" {
		t.Errorf("msg[1]: expected history user, got %s", msgs[1].Role)
	}
}

func TestReplan_NilHistory_SameBehavior(t *testing.T) {
	planJSON := `{"assistant_id":"a1","steps":[{"type":"skill","name":"retry","arguments":{}}]}`
	cp := &capturingProvider{
		response: &aitypes.GenerateResponse{Content: planJSON},
	}

	router := &mockRouter{provider: cp}
	p := NewLLMPlanner(router, nil)

	origPlan := &execution.ExecutionPlan{
		AssistantID: "a1",
		Steps:       []execution.ExecutionStep{{Type: execution.StepTypeSkill, Name: "step1"}},
	}
	if err := origPlan.Validate(); err != nil {
		t.Fatalf("validate: %v", err)
	}

	rCtx := &ReplanContext{
		OriginalPlan: origPlan,
		FailedStep:   execution.ExecutionStep{Name: "step1", StepID: "s1"},
		Trigger:      ReplanTriggerToolFailure,
		PlannerContext: &PlannerContext{
			AssistantID: "a1",
			Skills:      []aitypes.SkillDescriptor{{ID: "s1", Name: "S1", Description: "A skill"}},
			Soul:        AssistantSoul{SystemPrompt: "prompt"},
		},
	}

	_, err := p.Replan(context.Background(), rCtx)
	if err != nil {
		t.Fatalf("Replan failed: %v", err)
	}

	msgs := cp.captured.Messages
	if len(msgs) != 2 {
		t.Fatalf("expected 2 messages (no history), got %d", len(msgs))
	}
}
