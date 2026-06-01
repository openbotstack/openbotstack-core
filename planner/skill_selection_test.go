package planner

import (
	"strings"
	"testing"

	aitypes "github.com/openbotstack/openbotstack-core/ai/types"
)

// --- Phase 4: Skill Selection Tests ---
//
// Test planner's plan parsing logic with various query patterns.
// Uses parseResponse() directly — no LLM calls.

// SkillSelection 1: Exact match — math-add with numeric arguments
func TestSkillSelection_ExactMatch_MathAdd(t *testing.T) {
	p := NewLLMPlanner(nil, nil)

	llmResponse := `{
		"assistant_id": "default",
		"steps": [
			{"type": "skill", "name": "math-add", "arguments": {"a": 15, "b": 27}}
		]
	}`

	plan, err := p.parseResponse(llmResponse)
	if err != nil {
		t.Fatalf("parseResponse error: %v", err)
	}
	if len(plan.Steps) != 1 {
		t.Fatalf("steps = %d, want 1", len(plan.Steps))
	}
	if plan.Steps[0].Name != "math-add" {
		t.Errorf("step name = %q, want 'math-add'", plan.Steps[0].Name)
	}
	a, _ := plan.Steps[0].Arguments["a"].(float64)
	b, _ := plan.Steps[0].Arguments["b"].(float64)
	if a != 15 || b != 27 {
		t.Errorf("arguments = {a: %v, b: %v}, want {a: 15, b: 27}", a, b)
	}
}

// SkillSelection 2: Multi-step with template references
func TestSkillSelection_MultiStep_TemplateReferences(t *testing.T) {
	p := NewLLMPlanner(nil, nil)

	llmResponse := `{
		"assistant_id": "default",
		"steps": [
			{"type": "skill", "name": "summarize", "arguments": {"text": "long text...", "max_length": 100}},
			{"type": "skill", "name": "wordcount", "arguments": {"text": "{{summarize}}"}}
		]
	}`

	plan, err := p.parseResponse(llmResponse)
	if err != nil {
		t.Fatalf("parseResponse error: %v", err)
	}
	if len(plan.Steps) != 2 {
		t.Fatalf("steps = %d, want 2", len(plan.Steps))
	}

	if plan.Steps[0].Name != "summarize" {
		t.Errorf("step[0].name = %q, want 'summarize'", plan.Steps[0].Name)
	}

	// Second step has template reference
	if plan.Steps[1].Name != "wordcount" {
		t.Errorf("step[1].name = %q, want 'wordcount'", plan.Steps[1].Name)
	}
	textArg, _ := plan.Steps[1].Arguments["text"].(string)
	if textArg != "{{summarize}}" {
		t.Errorf("step[1].text = %q, want '{{summarize}}'", textArg)
	}
}

// SkillSelection 3: Field access syntax
func TestSkillSelection_FieldAccess(t *testing.T) {
	p := NewLLMPlanner(nil, nil)

	llmResponse := `{
		"assistant_id": "default",
		"steps": [
			{"type": "skill", "name": "math-add", "arguments": {"a": 20, "b": 22}},
			{"type": "skill", "name": "wordcount", "arguments": {"text": "{{math-add.sum}}"}}
		]
	}`

	plan, err := p.parseResponse(llmResponse)
	if err != nil {
		t.Fatalf("parseResponse error: %v", err)
	}

	textArg, _ := plan.Steps[1].Arguments["text"].(string)
	if textArg != "{{math-add.sum}}" {
		t.Errorf("text = %q, want '{{math-add.sum}}'", textArg)
	}
}

// SkillSelection 4: Empty plan — no matching skill
func TestSkillSelection_NoMatchingSkill(t *testing.T) {
	p := NewLLMPlanner(nil, nil)

	llmResponse := `{"assistant_id": "default", "steps": []}`

	plan, err := p.parseResponse(llmResponse)
	if err != nil {
		t.Fatalf("parseResponse error: %v", err)
	}
	if len(plan.Steps) != 0 {
		t.Errorf("steps = %d, want 0 (no matching skill)", len(plan.Steps))
	}
}

// SkillSelection 5: Ambiguous query — sentiment selected
func TestSkillSelection_AmbiguousQuery_SentimentVsSummarize(t *testing.T) {
	p := NewLLMPlanner(nil, nil)

	llmResponse := `{
		"assistant_id": "default",
		"steps": [
			{"type": "skill", "name": "sentiment", "arguments": {"text": "quarterly report summary"}}
		]
	}`

	plan, err := p.parseResponse(llmResponse)
	if err != nil {
		t.Fatalf("parseResponse error: %v", err)
	}
	if plan.Steps[0].Name != "sentiment" {
		t.Errorf("selected skill = %q, want 'sentiment'", plan.Steps[0].Name)
	}
}

// SkillSelection 6: Invalid JSON response
func TestSkillSelection_InvalidJSON(t *testing.T) {
	p := NewLLMPlanner(nil, nil)

	_, err := p.parseResponse("this is not JSON at all")
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

// SkillSelection 7: Markdown-wrapped JSON
func TestSkillSelection_MarkdownWrappedJSON(t *testing.T) {
	p := NewLLMPlanner(nil, nil)

	llmResponse := "```json\n{\"assistant_id\":\"default\",\"steps\":[{\"type\":\"skill\",\"name\":\"wordcount\",\"arguments\":{\"text\":\"hello\"}}]}\n```"

	plan, err := p.parseResponse(llmResponse)
	if err != nil {
		t.Fatalf("parseResponse error: %v", err)
	}
	if plan.Steps[0].Name != "wordcount" {
		t.Errorf("step name = %q, want 'wordcount'", plan.Steps[0].Name)
	}
}

// SkillSelection 8: BuildPrompt contains correct ToolSpecs
func TestSkillSelection_BuildPromptContainsToolSpecs(t *testing.T) {
	p := NewLLMPlanner(nil, nil)

	mockSkills := []aitypes.SkillDescriptor{
		{ID: "math-add", Name: "Math Add", Description: "Add two numbers"},
		{ID: "wordcount", Name: "Word Count", Description: "Count words in text"},
		{ID: "sentiment", Name: "Sentiment", Description: "Analyze text sentiment"},
	}

	pCtx := &PlannerContext{
		UserRequest: "Add 5 and 3",
		Skills:      mockSkills,
	}

	prompt := p.buildPrompt(pCtx)

	// Verify prompt contains all skill IDs
	for _, skill := range mockSkills {
		if !strings.Contains(prompt, skill.ID) {
			t.Errorf("prompt missing skill %q", skill.ID)
		}
	}

	// Verify prompt contains user request
	if !strings.Contains(prompt, "Add 5 and 3") {
		t.Error("prompt missing user request")
	}

	// Verify template syntax guidance
	if !strings.Contains(prompt, "{{step_name") {
		t.Error("prompt missing template syntax guidance")
	}
}
