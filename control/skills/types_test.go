package skills_test

import (
	"encoding/json"
	"testing"

	"github.com/openbotstack/openbotstack-core/control/skills"
)

// --- CapabilityType constants (6 constants) ---

func TestCapabilityType_Constants(t *testing.T) {
	tests := []struct {
		name  string
		value skills.CapabilityType
		want  string
	}{
		{"text_generation", skills.CapTextGeneration, "text_generation"},
		{"tool_calling", skills.CapToolCalling, "tool_calling"},
		{"json_mode", skills.CapJSONMode, "json_mode"},
		{"embedding", skills.CapEmbedding, "embedding"},
		{"vision", skills.CapVision, "vision"},
		{"streaming", skills.CapStreaming, "streaming"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.value) != tt.want {
				t.Errorf("expected %q, got %q", tt.want, string(tt.value))
			}
		})
	}
}

// --- JSONSchema round-trip serialization ---

func TestJSONSchema_MarshalUnmarshal(t *testing.T) {
	original := &skills.JSONSchema{
		Type: "object",
		Properties: map[string]*skills.JSONSchema{
			"name": {Type: "string"},
			"age":  {Type: "integer"},
		},
		Required: []string{"name"},
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded skills.JSONSchema
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if decoded.Type != "object" {
		t.Errorf("Type: expected 'object', got %q", decoded.Type)
	}
	if len(decoded.Properties) != 2 {
		t.Errorf("Properties: expected 2, got %d", len(decoded.Properties))
	}
	if decoded.Properties["name"].Type != "string" {
		t.Errorf("Properties[name].Type: expected 'string', got %q", decoded.Properties["name"].Type)
	}
	if len(decoded.Required) != 1 || decoded.Required[0] != "name" {
		t.Errorf("Required: expected ['name'], got %v", decoded.Required)
	}
}

// --- JSONSchema omitempty ---

func TestJSONSchema_OmitEmpty(t *testing.T) {
	s := &skills.JSONSchema{}
	data, err := json.Marshal(s)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	// With omitempty, all fields should be absent
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(m) != 0 {
		t.Errorf("expected empty JSON object, got %v", m)
	}
}

// --- GenerateRequest zero values do not panic ---

func TestGenerateRequest_ZeroValues(t *testing.T) {
	req := skills.GenerateRequest{}

	// Access all fields to ensure no panic
	_ = req.Messages
	_ = req.Tools
	_ = req.MaxTokens
	_ = req.Temperature
	_ = req.JSONSchema

	// Verify zero-value invariants
	if req.Messages != nil {
		t.Errorf("expected nil Messages, got %v", req.Messages)
	}
	if req.Tools != nil {
		t.Errorf("expected nil Tools, got %v", req.Tools)
	}
	if req.MaxTokens != 0 {
		t.Errorf("expected 0 MaxTokens, got %d", req.MaxTokens)
	}
	if req.Temperature != 0 {
		t.Errorf("expected 0 Temperature, got %f", req.Temperature)
	}
	if req.JSONSchema != nil {
		t.Errorf("expected nil JSONSchema, got %v", req.JSONSchema)
	}
}

// --- ToolDefinition field assignment and serialization ---

func TestToolDefinition_Fields(t *testing.T) {
	td := skills.ToolDefinition{
		Name:        "search",
		Description: "Search the web",
		Parameters: &skills.JSONSchema{
			Type: "object",
			Properties: map[string]*skills.JSONSchema{
				"query": {Type: "string"},
			},
			Required: []string{"query"},
		},
	}

	if td.Name != "search" {
		t.Errorf("Name: expected 'search', got %q", td.Name)
	}
	if td.Description != "Search the web" {
		t.Errorf("Description: expected 'Search the web', got %q", td.Description)
	}
	if td.Parameters == nil {
		t.Fatal("Parameters should not be nil")
	}
	if td.Parameters.Type != "object" {
		t.Errorf("Parameters.Type: expected 'object', got %q", td.Parameters.Type)
	}
}

func TestToolDefinition_Serialization(t *testing.T) {
	td := skills.ToolDefinition{
		Name:        "calculator",
		Description: "Performs arithmetic",
		Parameters: &skills.JSONSchema{
			Type: "object",
			Properties: map[string]*skills.JSONSchema{
				"expression": {Type: "string"},
			},
			Required: []string{"expression"},
		},
	}

	data, err := json.Marshal(td)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded skills.ToolDefinition
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if decoded.Name != "calculator" {
		t.Errorf("Name: expected 'calculator', got %q", decoded.Name)
	}
	if decoded.Description != "Performs arithmetic" {
		t.Errorf("Description: got %q", decoded.Description)
	}
	if decoded.Parameters == nil {
		t.Fatal("Parameters should not be nil after round-trip")
	}
}

// --- Message struct ---

func TestMessage_Fields(t *testing.T) {
	msg := skills.Message{
		Role:    "user",
		Content: "Hello",
		Name:    "test-user",
	}
	if msg.Role != "user" {
		t.Errorf("Role: expected 'user', got %q", msg.Role)
	}
	if msg.Content != "Hello" {
		t.Errorf("Content: expected 'Hello', got %q", msg.Content)
	}
	if msg.Name != "test-user" {
		t.Errorf("Name: expected 'test-user', got %q", msg.Name)
	}
}

// --- ToolCall struct ---

func TestToolCall_Fields(t *testing.T) {
	tc := skills.ToolCall{
		ID:        "call-1",
		Name:      "search",
		Arguments: `{"query": "test"}`,
	}
	if tc.ID != "call-1" {
		t.Errorf("ID: expected 'call-1', got %q", tc.ID)
	}
	if tc.Name != "search" {
		t.Errorf("Name: expected 'search', got %q", tc.Name)
	}
	if tc.Arguments != `{"query": "test"}` {
		t.Errorf("Arguments: got %q", tc.Arguments)
	}
}

// --- TokenUsage struct ---

func TestTokenUsage_Fields(t *testing.T) {
	usage := skills.TokenUsage{
		PromptTokens:     100,
		CompletionTokens: 50,
		TotalTokens:      150,
	}
	if usage.PromptTokens != 100 || usage.CompletionTokens != 50 || usage.TotalTokens != 150 {
		t.Errorf("TokenUsage fields mismatch: %+v", usage)
	}
}
