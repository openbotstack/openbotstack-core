package providers

import (
	"encoding/json"
	"testing"

	skills "github.com/openbotstack/openbotstack-core/control/skills"
)

// --- mapToolChoiceToOpenAI tests ---

func TestMapToolChoiceToOpenAI_Nil(t *testing.T) {
	result := mapToolChoiceToOpenAI(nil)
	if result != nil {
		t.Errorf("nil should return nil, got %v", result)
	}
}

func TestMapToolChoiceToOpenAI_Auto(t *testing.T) {
	result := mapToolChoiceToOpenAI(skills.ToolChoiceAuto)
	s, ok := result.(string)
	if !ok {
		t.Fatalf("expected string, got %T", result)
	}
	if s != "auto" {
		t.Errorf("expected 'auto', got %q", s)
	}
}

func TestMapToolChoiceToOpenAI_Required(t *testing.T) {
	result := mapToolChoiceToOpenAI(skills.ToolChoiceRequired)
	s, ok := result.(string)
	if !ok {
		t.Fatalf("expected string, got %T", result)
	}
	if s != "required" {
		t.Errorf("expected 'required', got %q", s)
	}
}

func TestMapToolChoiceToOpenAI_None(t *testing.T) {
	result := mapToolChoiceToOpenAI(skills.ToolChoiceNone)
	s, ok := result.(string)
	if !ok {
		t.Fatalf("expected string, got %T", result)
	}
	if s != "none" {
		t.Errorf("expected 'none', got %q", s)
	}
}

func TestMapToolChoiceToOpenAI_Specific(t *testing.T) {
	result := mapToolChoiceToOpenAI(skills.ToolChoiceSpecific{Name: "search"})
	tc, ok := result.(openAIToolChoice)
	if !ok {
		t.Fatalf("expected openAIToolChoice, got %T", result)
	}
	if tc.Type != "function" {
		t.Errorf("expected type 'function', got %q", tc.Type)
	}
	if tc.Function.Name != "search" {
		t.Errorf("expected function name 'search', got %q", tc.Function.Name)
	}
}

func TestMapToolChoiceToOpenAI_Serialization_Specific(t *testing.T) {
	result := mapToolChoiceToOpenAI(skills.ToolChoiceSpecific{Name: "calc"})
	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}
	expected := `{"type":"function","function":{"name":"calc"}}`
	if string(data) != expected {
		t.Errorf("expected %s, got %s", expected, string(data))
	}
}

func TestMapToolChoiceToOpenAI_Serialization_Auto(t *testing.T) {
	result := mapToolChoiceToOpenAI(skills.ToolChoiceAuto)
	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}
	expected := `"auto"`
	if string(data) != expected {
		t.Errorf("expected %s, got %s", expected, string(data))
	}
}

func TestMapToolChoiceToOpenAI_UnknownMode(t *testing.T) {
	result := mapToolChoiceToOpenAI(skills.ToolChoiceMode("unknown"))
	if result != nil {
		t.Errorf("unknown mode should return nil, got %v", result)
	}
}

func TestMapToolChoiceToOpenAI_WrongType(t *testing.T) {
	result := mapToolChoiceToOpenAI("auto")
	if result != nil {
		t.Errorf("plain string should return nil, got %v", result)
	}
}

// --- mapToolChoiceToAnthropic tests ---

func TestMapToolChoiceToAnthropic_Nil(t *testing.T) {
	result := mapToolChoiceToAnthropic(nil)
	if result != nil {
		t.Errorf("nil should return nil, got %v", result)
	}
}

func TestMapToolChoiceToAnthropic_Auto(t *testing.T) {
	result := mapToolChoiceToAnthropic(skills.ToolChoiceAuto)
	s, ok := result.(string)
	if !ok {
		t.Fatalf("expected string, got %T", result)
	}
	if s != "auto" {
		t.Errorf("expected 'auto', got %q", s)
	}
}

func TestMapToolChoiceToAnthropic_Required(t *testing.T) {
	result := mapToolChoiceToAnthropic(skills.ToolChoiceRequired)
	tc, ok := result.(anthropicToolChoice)
	if !ok {
		t.Fatalf("expected anthropicToolChoice, got %T", result)
	}
	if tc.Type != "any" {
		t.Errorf("expected type 'any' for required, got %q", tc.Type)
	}
}

func TestMapToolChoiceToAnthropic_None(t *testing.T) {
	result := mapToolChoiceToAnthropic(skills.ToolChoiceNone)
	s, ok := result.(string)
	if !ok {
		t.Fatalf("expected string, got %T", result)
	}
	if s != "none" {
		t.Errorf("expected 'none', got %q", s)
	}
}

func TestMapToolChoiceToAnthropic_Specific(t *testing.T) {
	result := mapToolChoiceToAnthropic(skills.ToolChoiceSpecific{Name: "search"})
	tc, ok := result.(anthropicToolChoice)
	if !ok {
		t.Fatalf("expected anthropicToolChoice, got %T", result)
	}
	if tc.Type != "tool" {
		t.Errorf("expected type 'tool', got %q", tc.Type)
	}
	if tc.Name != "search" {
		t.Errorf("expected name 'search', got %q", tc.Name)
	}
}

func TestMapToolChoiceToAnthropic_Serialization_Specific(t *testing.T) {
	result := mapToolChoiceToAnthropic(skills.ToolChoiceSpecific{Name: "calc"})
	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}
	expected := `{"type":"tool","name":"calc"}`
	if string(data) != expected {
		t.Errorf("expected %s, got %s", expected, string(data))
	}
}

func TestMapToolChoiceToAnthropic_UnknownMode(t *testing.T) {
	result := mapToolChoiceToAnthropic(skills.ToolChoiceMode("unknown"))
	if result != nil {
		t.Errorf("unknown mode should return nil, got %v", result)
	}
}

func TestMapToolChoiceToAnthropic_WrongType(t *testing.T) {
	result := mapToolChoiceToAnthropic(42)
	if result != nil {
		t.Errorf("int should return nil, got %v", result)
	}
}

// --- ToolChoiceMode constant tests ---

func TestToolChoiceMode_Values(t *testing.T) {
	if skills.ToolChoiceAuto != "auto" {
		t.Errorf("expected 'auto', got %q", skills.ToolChoiceAuto)
	}
	if skills.ToolChoiceRequired != "required" {
		t.Errorf("expected 'required', got %q", skills.ToolChoiceRequired)
	}
	if skills.ToolChoiceNone != "none" {
		t.Errorf("expected 'none', got %q", skills.ToolChoiceNone)
	}
}

// --- ToolChoiceSpecific tests ---

func TestToolChoiceSpecific_Field(t *testing.T) {
	tc := skills.ToolChoiceSpecific{Name: "my_tool"}
	if tc.Name != "my_tool" {
		t.Errorf("expected 'my_tool', got %q", tc.Name)
	}
}

// --- GenerateRequest ToolChoice field test ---

func TestGenerateRequest_ToolChoice(t *testing.T) {
	req := skills.GenerateRequest{
		ToolChoice: skills.ToolChoiceAuto,
	}
	if req.ToolChoice != skills.ToolChoiceAuto {
		t.Errorf("expected ToolChoiceAuto, got %v", req.ToolChoice)
	}

	req.ToolChoice = skills.ToolChoiceSpecific{Name: "search"}
	if tc, ok := req.ToolChoice.(skills.ToolChoiceSpecific); !ok || tc.Name != "search" {
		t.Errorf("expected ToolChoiceSpecific{name:search}, got %v", req.ToolChoice)
	}
}

// --- G17: unsupported ToolChoice types ---

func TestMapToolChoiceToOpenAI_UnsupportedTypes(t *testing.T) {
	tests := []struct {
		name string
		val  any
	}{
		{"int", 42},
		{"slice", []string{"a", "b"}},
		{"struct", struct{ X int }{X: 1}},
		{"float", 3.14},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mapToolChoiceToOpenAI(tt.val)
			if result != nil {
				t.Errorf("unsupported type %T should return nil, got %v", tt.val, result)
			}
		})
	}
}

func TestMapToolChoiceToAnthropic_UnsupportedTypes(t *testing.T) {
	tests := []struct {
		name string
		val  any
	}{
		{"int", 42},
		{"slice", []string{"a", "b"}},
		{"struct", struct{ X int }{X: 1}},
		{"float", 3.14},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mapToolChoiceToAnthropic(tt.val)
			if result != nil {
				t.Errorf("unsupported type %T should return nil, got %v", tt.val, result)
			}
		})
	}
}
