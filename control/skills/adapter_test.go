package skills

import (
	"encoding/json"
	"testing"
	"time"
)

// --- Mock for SkillInfo ---

type mockSkillInfo struct {
	id          string
	desc        string
	inputSchema *JSONSchema
}

func (m *mockSkillInfo) ID() string               { return m.id }
func (m *mockSkillInfo) Description() string      { return m.desc }
func (m *mockSkillInfo) InputSchema() *JSONSchema { return m.inputSchema }

// Also implement full Skill for completeness
func (m *mockSkillInfo) Name() string                  { return m.id }
func (m *mockSkillInfo) OutputSchema() *JSONSchema     { return nil }
func (m *mockSkillInfo) RequiredPermissions() []string { return nil }
func (m *mockSkillInfo) Timeout() time.Duration        { return 30 * time.Second }
func (m *mockSkillInfo) Validate() error               { return nil }

// --- SkillsToOpenAITools tests ---

func TestSkillsToOpenAITools_BasicConversion(t *testing.T) {
	skills := []SkillInfo{
		&mockSkillInfo{
			id:   "core/search",
			desc: "Search the web",
			inputSchema: &JSONSchema{
				Type: "object",
				Properties: map[string]*JSONSchema{
					"query": {Type: "string", Description: "search query"},
				},
				Required: []string{"query"},
			},
		},
	}

	tools := SkillsToOpenAITools(skills)
	if len(tools) != 1 {
		t.Fatalf("expected 1 tool, got %d", len(tools))
	}

	tool := tools[0]
	if tool.Name != "core/search" {
		t.Errorf("expected name 'core/search', got %q", tool.Name)
	}
	if tool.Description != "Search the web" {
		t.Errorf("expected description 'Search the web', got %q", tool.Description)
	}
	if tool.Parameters == nil {
		t.Fatal("expected non-nil parameters")
	}
	if tool.Parameters.Type != "object" {
		t.Errorf("expected parameters type 'object', got %q", tool.Parameters.Type)
	}
}

func TestSkillsToOpenAITools_NilSchema(t *testing.T) {
	skills := []SkillInfo{
		&mockSkillInfo{id: "core/noop", desc: "Does nothing"},
	}

	tools := SkillsToOpenAITools(skills)
	if len(tools) != 1 {
		t.Fatalf("expected 1 tool, got %d", len(tools))
	}
	if tools[0].Parameters != nil {
		t.Error("nil input schema should produce nil parameters")
	}
}

func TestSkillsToOpenAITools_MultipleSkills(t *testing.T) {
	skills := []SkillInfo{
		&mockSkillInfo{id: "a", desc: "Skill A"},
		&mockSkillInfo{id: "b", desc: "Skill B"},
		&mockSkillInfo{id: "c", desc: "Skill C"},
	}

	tools := SkillsToOpenAITools(skills)
	if len(tools) != 3 {
		t.Fatalf("expected 3 tools, got %d", len(tools))
	}
}

func TestSkillsToOpenAITools_EmptySlice(t *testing.T) {
	tools := SkillsToOpenAITools(nil)
	if len(tools) != 0 {
		t.Errorf("expected 0 tools, got %d", len(tools))
	}
}

func TestSkillsToOpenAITools_SchemaPreserved(t *testing.T) {
	inputSchema := &JSONSchema{
		Type:       "object",
		Required:   []string{"text"},
		Enum:       []any{"a", "b"},
		MinLength:  intPtr(1),
		MaxLength:  intPtr(100),
		Minimum:    floatPtr(0),
		Maximum:    floatPtr(10),
		Pattern:    "^[a-z]+$",
		Items:      &JSONSchema{Type: "string"},
		AnyOf:      []*JSONSchema{{Type: "string"}, {Type: "number"}},
		AdditionalProperties: boolPtr(false),
		Properties: map[string]*JSONSchema{
			"text": {Type: "string"},
		},
	}
	skills := []SkillInfo{
		&mockSkillInfo{id: "complex", desc: "Complex schema", inputSchema: inputSchema},
	}

	tools := SkillsToOpenAITools(skills)
	tool := tools[0]

	original, _ := json.Marshal(inputSchema)
	got, _ := json.Marshal(tool.Parameters)
	if string(original) != string(got) {
		t.Errorf("schema not preserved:\nwant: %s\ngot:  %s", original, got)
	}
}

// --- SkillsToAnthropicTools tests ---

func TestSkillsToAnthropicTools_BasicConversion(t *testing.T) {
	skills := []SkillInfo{
		&mockSkillInfo{
			id:   "core/search",
			desc: "Search the web",
			inputSchema: &JSONSchema{
				Type: "object",
				Properties: map[string]*JSONSchema{
					"query": {Type: "string"},
				},
				Required: []string{"query"},
			},
		},
	}

	tools := SkillsToAnthropicTools(skills)
	if len(tools) != 1 {
		t.Fatalf("expected 1 tool, got %d", len(tools))
	}

	tool := tools[0]
	if tool.Name != "core/search" {
		t.Errorf("expected name 'core/search', got %q", tool.Name)
	}
	if tool.Description != "Search the web" {
		t.Errorf("expected description 'Search the web', got %q", tool.Description)
	}
	if tool.Parameters == nil {
		t.Fatal("expected non-nil parameters")
	}
}

func TestSkillsToAnthropicTools_EmptySlice(t *testing.T) {
	tools := SkillsToAnthropicTools(nil)
	if len(tools) != 0 {
		t.Errorf("expected 0 tools, got %d", len(tools))
	}
}

func TestSkillsToAnthropicTools_NilSchema(t *testing.T) {
	skills := []SkillInfo{
		&mockSkillInfo{id: "noop", desc: "No schema"},
	}

	tools := SkillsToAnthropicTools(skills)
	if len(tools) != 1 {
		t.Fatalf("expected 1 tool, got %d", len(tools))
	}
	if tools[0].Parameters != nil {
		t.Error("nil input schema should produce nil parameters")
	}
}

// --- NormalizeArguments tests ---

func TestNormalizeArguments_JSONString(t *testing.T) {
	result, err := NormalizeArguments(`{"key":"value","num":42}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result["key"] != "value" {
		t.Errorf("expected key='value', got %v", result["key"])
	}
	if v, ok := result["num"].(float64); !ok || v != 42 {
		t.Errorf("expected num=42, got %v", result["num"])
	}
}

func TestNormalizeArguments_EmptyObject(t *testing.T) {
	result, err := NormalizeArguments(`{}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected empty map, got %d keys", len(result))
	}
}

func TestNormalizeArguments_InvalidJSON(t *testing.T) {
	_, err := NormalizeArguments(`not json`)
	if err == nil {
		t.Error("invalid JSON should return error")
	}
}

func TestNormalizeArguments_EmptyString(t *testing.T) {
	result, err := NormalizeArguments("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected empty map for empty string, got %d keys", len(result))
	}
}

func TestNormalizeArguments_NestedObject(t *testing.T) {
	result, err := NormalizeArguments(`{"outer":{"inner":"deep"}}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	outer, ok := result["outer"].(map[string]interface{})
	if !ok {
		t.Fatal("expected nested map")
	}
	if outer["inner"] != "deep" {
		t.Errorf("expected inner=deep, got %v", outer["inner"])
	}
}

func TestNormalizeArguments_ArrayValue(t *testing.T) {
	result, err := NormalizeArguments(`{"items":[1,2,3]}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	arr, ok := result["items"].([]interface{})
	if !ok {
		t.Fatal("expected array")
	}
	if len(arr) != 3 {
		t.Errorf("expected 3 items, got %d", len(arr))
	}
}

func TestNormalizeArguments_NullJSON(t *testing.T) {
	result, err := NormalizeArguments("null")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Error("null JSON should not return nil map")
	}
	if len(result) != 0 {
		t.Errorf("expected empty map, got %d keys", len(result))
	}
}

func TestNormalizeArguments_NonObjectBoolean(t *testing.T) {
	_, err := NormalizeArguments("true")
	if err == nil {
		t.Error("boolean value should fail (not an object)")
	}
}

func TestNormalizeArguments_NonObjectNumber(t *testing.T) {
	_, err := NormalizeArguments("42")
	if err == nil {
		t.Error("number value should fail (not an object)")
	}
}

func TestNormalizeArguments_NonObjectString(t *testing.T) {
	_, err := NormalizeArguments(`"hello"`)
	if err == nil {
		t.Error("string value should fail (not an object)")
	}
}

func TestNormalizeArguments_NonObjectArray(t *testing.T) {
	_, err := NormalizeArguments(`[1,2]`)
	if err == nil {
		t.Error("array value should fail (not an object)")
	}
}

// --- Pointer helpers (avoid collision with validation package) ---

func intPtr(v int) *int            { return &v }
func floatPtr(v float64) *float64 { return &v }
func boolPtr(v bool) *bool        { return &v }

// --- ParallelToolCalls tests ---

func TestGenerateRequest_ParallelToolCalls(t *testing.T) {
	req := GenerateRequest{}
	if req.ParallelToolCalls != nil {
		t.Error("default should be nil")
	}

	enabled := true
	req.ParallelToolCalls = &enabled
	if req.ParallelToolCalls == nil || !*req.ParallelToolCalls {
		t.Error("should be true")
	}

	disabled := false
	req.ParallelToolCalls = &disabled
	if req.ParallelToolCalls == nil || *req.ParallelToolCalls {
		t.Error("should be false")
	}
}
