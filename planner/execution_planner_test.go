package planner

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/openbotstack/openbotstack-core/ai/providers"
	"github.com/openbotstack/openbotstack-core/assistant"
	"github.com/openbotstack/openbotstack-core/control/skills"
	registry "github.com/openbotstack/openbotstack-core/registry/skills"
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

// Basic structural test preserved
func TestLLMPlanner(t *testing.T) {
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
	if limits.MaxToolCalls != 5 {
		t.Errorf("expected MaxToolCalls=5, got %d", limits.MaxToolCalls)
	}
	if limits.MaxExecutionTime != 10*time.Second {
		t.Errorf("expected MaxExecutionTime=10s, got %v", limits.MaxExecutionTime)
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

func TestValidator_EmptyStepsValid(t *testing.T) {
	v := NewValidator(nil)
	plan := &execution.ExecutionPlan{
		AssistantID: "asst",
		Steps:       []execution.ExecutionStep{},
	}
	if err := v.Validate(plan); err != nil {
		t.Fatalf("empty steps should be valid (no steps to check), got: %v", err)
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
		Soul: assistant.AssistantSoul{
			Personality:  "Friendly and precise",
			Instructions: "Always be concise",
		},
		Skills: []SkillDescriptor{
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
		MemoryContext: []assistant.SearchResult{
			{Content: []byte("user likes Go"), Score: 0.9},
			{Content: []byte("user prefers dark mode"), Score: 0.8},
		},
		Skills: []SkillDescriptor{
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
		Skills:      []SkillDescriptor{},
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
		Skills: []SkillDescriptor{
			{
				ID:          "tax-calc",
				Name:        "Tax Calculator",
				Description: "Calculates tax",
				InputSchema: &skills.JSONSchema{
					Type: "object",
					Properties: map[string]*skills.JSONSchema{
						"income": {Type: "number"},
					},
					Required: []string{"income"},
				},
			},
		},
	})

	if !strings.Contains(prompt, "Input schema:") {
		t.Error("expected prompt to contain Input schema label")
	}
	// Verify the schema is serialized with actual JSON content, not "{}"
	if strings.Contains(prompt, "Input schema: {}") {
		t.Error("expected non-empty schema JSON, got {}")
	}
	if !strings.Contains(prompt, `"type":"object"`) {
		t.Error("expected schema to contain type:object")
	}
}

func TestBuildPrompt_WithNilInputSchema(t *testing.T) {
	p := NewLLMPlanner(nil, nil)
	prompt := p.buildPrompt(&PlannerContext{
		AssistantID: "a1",
		UserRequest: "hello",
		Skills: []SkillDescriptor{
			{
				ID:          "hello",
				Name:        "Hello",
				Description: "Says hello",
				InputSchema: nil,
			},
		},
	})

	if !strings.Contains(prompt, "Input schema: {}") {
		t.Error("expected prompt to contain empty schema placeholder when InputSchema is nil")
	}
}

func TestBuildPrompt_ContainsUserRequest(t *testing.T) {
	p := NewLLMPlanner(nil, nil)
	prompt := p.buildPrompt(&PlannerContext{
		AssistantID: "a1",
		UserRequest: "analyze the dataset",
		Skills: []SkillDescriptor{
			{ID: "s1", Name: "S1", Description: "A skill"},
		},
	})

	if !strings.Contains(prompt, "User request: analyze the dataset") {
		t.Error("expected prompt to contain user request")
	}
}

func TestBuildPrompt_ContainsJSONFormatInstructions(t *testing.T) {
	p := NewLLMPlanner(nil, nil)
	prompt := p.buildPrompt(&PlannerContext{
		AssistantID: "a1",
		UserRequest: "test",
		Skills: []SkillDescriptor{
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
		Soul:        assistant.AssistantSoul{}, // all empty
		Skills: []SkillDescriptor{
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

	// parseResponse only strips ```json / ``` prefixes/suffixes, it does not
	// strip arbitrary text before a code block. Text before the block will
	// cause a JSON parse error, which is the expected behavior.
	input := "Here is the plan:\n```json\n{\"assistant_id\":\"a1\",\"steps\":[{\"type\":\"skill\",\"name\":\"s1\",\"arguments\":{}}]}\n```"
	_, err := p.parseResponse(input)
	if err == nil {
		t.Fatal("expected error for text before code block (parseResponse does not strip it)")
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
	capabilities []skills.CapabilityType
	response     *skills.GenerateResponse
	err          error
}

func (m *mockProvider) ID() string                            { return m.id }
func (m *mockProvider) Capabilities() []skills.CapabilityType { return m.capabilities }
func (m *mockProvider) Generate(_ context.Context, _ skills.GenerateRequest) (*skills.GenerateResponse, error) {
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

func (m *mockRouter) Route(_ []skills.CapabilityType, _ skills.ModelConstraints) (providers.ModelProvider, error) {
	return m.provider, m.err
}

func (m *mockRouter) Register(_ providers.ModelProvider) error { return nil }
func (m *mockRouter) List() []string                           { return nil }

func TestPlan_NoSkills(t *testing.T) {
	p := NewLLMPlanner(nil, nil)
	_, err := p.Plan(context.Background(), &PlannerContext{
		AssistantID: "a1",
		Skills:      []SkillDescriptor{},
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
		Skills: []SkillDescriptor{
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
		Skills: []SkillDescriptor{
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
			response: &skills.GenerateResponse{
				Content: "this is not valid JSON",
			},
		},
	}
	p := NewLLMPlanner(router, nil)
	_, err := p.Plan(context.Background(), &PlannerContext{
		AssistantID: "a1",
		Skills: []SkillDescriptor{
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
			response: &skills.GenerateResponse{
				Content: planJSON,
			},
		},
	}
	p := NewLLMPlanner(router, nil)
	_, err := p.Plan(context.Background(), &PlannerContext{
		AssistantID: "a1",
		Skills: []SkillDescriptor{
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
			response: &skills.GenerateResponse{
				Content: planJSON,
			},
		},
	}
	p := NewLLMPlanner(router, nil)
	plan, err := p.Plan(context.Background(), &PlannerContext{
		AssistantID: "a1",
		Skills: []SkillDescriptor{
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
			response: &skills.GenerateResponse{
				Content: planJSON,
			},
		},
	}
	p := NewLLMPlanner(router, nil)
	plan, err := p.Plan(context.Background(), &PlannerContext{
		AssistantID: "context-assistant",
		Skills: []SkillDescriptor{
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
			response: &skills.GenerateResponse{
				Content: planJSON,
			},
		},
	}
	p := NewLLMPlanner(router, nil)
	plan, err := p.Plan(context.Background(), &PlannerContext{
		AssistantID: "a1",
		Skills: []SkillDescriptor{
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
			response: &skills.GenerateResponse{
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
		Skills: []SkillDescriptor{
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

// ---------------------------------------------------------------------------
// ContextBuilder tests
// ---------------------------------------------------------------------------

func TestNewContextBuilder(t *testing.T) {
	cb := NewContextBuilder(nil)
	if cb == nil {
		t.Fatal("expected non-nil ContextBuilder")
	}
}

func TestContextBuilder_NilRuntime(t *testing.T) {
	cb := NewContextBuilder(nil)
	_, err := cb.Build(context.Background(), nil, "test request")
	if err == nil {
		t.Fatal("expected error for nil runtime")
	}
	if !strings.Contains(err.Error(), "runtime is nil") {
		t.Errorf("expected 'runtime is nil' error, got: %v", err)
	}
}

func TestContextBuilder_WithRuntime(t *testing.T) {
	mockReg := &mockSkillRegistry{
		skills: map[string]*mockSkillDef{
			"skill-a": {id: "skill-a", name: "Skill A", desc: "Does A", schema: &skills.JSONSchema{Type: "object"}},
			"skill-b": {id: "skill-b", name: "Skill B", desc: "Does B", schema: nil},
		},
	}
	mem := assistant.NewSessionMemory()

	cb := NewContextBuilder(mockReg)
	pCtx, err := cb.Build(context.Background(), &assistant.AssistantRuntime{
		AssistantID: "asst-1",
		Soul:        assistant.DefaultSoul(),
		Skills:      []string{"skill-a", "skill-b", "skill-c"}, // skill-c doesn't exist
		Memory:      mem,
	}, "hello")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pCtx.AssistantID != "asst-1" {
		t.Errorf("expected assistant ID asst-1, got %s", pCtx.AssistantID)
	}
	if len(pCtx.Skills) != 2 {
		t.Fatalf("expected 2 skills (skill-c should be skipped), got %d", len(pCtx.Skills))
	}
	if pCtx.Skills[0].ID != "skill-a" {
		t.Errorf("expected first skill to be skill-a, got %s", pCtx.Skills[0].ID)
	}
	if pCtx.Skills[0].InputSchema == nil || pCtx.Skills[0].InputSchema.Type != "object" {
		t.Error("expected skill-a to have input schema")
	}
	if pCtx.Skills[1].InputSchema != nil {
		t.Error("expected skill-b to have nil input schema")
	}
	if pCtx.UserRequest != "hello" {
		t.Errorf("expected user request 'hello', got %s", pCtx.UserRequest)
	}
}

func TestContextBuilder_MemorySearchError(t *testing.T) {
	// Use a memory that returns an error on Search
	mockReg := &mockSkillRegistry{
		skills: map[string]*mockSkillDef{
			"skill-a": {id: "skill-a", name: "Skill A", desc: "Does A", schema: nil},
		},
	}

	cb := NewContextBuilder(mockReg)
	pCtx, err := cb.Build(context.Background(), &assistant.AssistantRuntime{
		AssistantID: "asst-1",
		Soul:        assistant.DefaultSoul(),
		Skills:      []string{"skill-a"},
		Memory:      &errorMemory{},
	}, "hello")

	if err != nil {
		t.Fatalf("unexpected error (memory errors should be tolerated): %v", err)
	}
	// Memory search error should be tolerated, resulting in nil memory context
	if len(pCtx.MemoryContext) != 0 {
		t.Errorf("expected empty memory context on search error, got %d items", len(pCtx.MemoryContext))
	}
}

func TestContextBuilder_EmptySkillList(t *testing.T) {
	mockReg := &mockSkillRegistry{
		skills: map[string]*mockSkillDef{},
	}
	mem := assistant.NewSessionMemory()

	cb := NewContextBuilder(mockReg)
	pCtx, err := cb.Build(context.Background(), &assistant.AssistantRuntime{
		AssistantID: "asst-1",
		Soul:        assistant.DefaultSoul(),
		Skills:      []string{},
		Memory:      mem,
	}, "hello")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(pCtx.Skills) != 0 {
		t.Errorf("expected 0 skills, got %d", len(pCtx.Skills))
	}
}

// errorMemory is an AssistantMemory that always returns an error on Search
type errorMemory struct{}

func (m *errorMemory) Get(_ context.Context, _ string) ([]byte, error) {
	return nil, fmt.Errorf("not found")
}
func (m *errorMemory) Set(_ context.Context, _ string, _ []byte) error {
	return nil
}
func (m *errorMemory) Search(_ context.Context, _ string, _ int) ([]assistant.SearchResult, error) {
	return nil, fmt.Errorf("search error")
}

// mockSkillRegistry implements registry.SkillRegistry
type mockSkillRegistry struct {
	skills map[string]*mockSkillDef
}

func (m *mockSkillRegistry) Register(_ registry.Skill) error { return nil }
func (m *mockSkillRegistry) Get(id string) (registry.Skill, error) {
	s, ok := m.skills[id]
	if !ok {
		return nil, fmt.Errorf("not found: %s", id)
	}
	return s, nil
}
func (m *mockSkillRegistry) List() []string { return nil }
func (m *mockSkillRegistry) ListByPermission(_ []string) []registry.Skill {
	return nil
}
func (m *mockSkillRegistry) Validate() error { return nil }

// mockSkillDef implements registry.Skill
type mockSkillDef struct {
	id     string
	name   string
	desc   string
	schema *skills.JSONSchema
}

func (m *mockSkillDef) ID() string                      { return m.id }
func (m *mockSkillDef) Name() string                    { return m.name }
func (m *mockSkillDef) Description() string             { return m.desc }
func (m *mockSkillDef) InputSchema() *skills.JSONSchema { return m.schema }
func (m *mockSkillDef) OutputSchema() *skills.JSONSchema {
	return &skills.JSONSchema{Type: "object"}
}
func (m *mockSkillDef) RequiredPermissions() []string { return nil }
func (m *mockSkillDef) Timeout() time.Duration        { return 5 * time.Second }
func (m *mockSkillDef) Validate() error               { return nil }
