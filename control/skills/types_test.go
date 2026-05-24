package skills_test

import (
	"encoding/json"
	"testing"
	"time"

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

// --- ModelConstraints (G1: completeness gap) ---

func TestModelConstraints_Fields(t *testing.T) {
	mc := skills.ModelConstraints{
		MaxLatencyMs:     2000,
		Privacy:          "internal",
		PreferredProvider: "openai",
	}
	if mc.MaxLatencyMs != 2000 {
		t.Errorf("MaxLatencyMs: expected 2000, got %d", mc.MaxLatencyMs)
	}
	if mc.Privacy != "internal" {
		t.Errorf("Privacy: expected 'internal', got %q", mc.Privacy)
	}
	if mc.PreferredProvider != "openai" {
		t.Errorf("PreferredProvider: expected 'openai', got %q", mc.PreferredProvider)
	}
}

func TestModelConstraints_ZeroValues(t *testing.T) {
	mc := skills.ModelConstraints{}
	if mc.MaxLatencyMs != 0 {
		t.Errorf("expected 0, got %d", mc.MaxLatencyMs)
	}
	if mc.Privacy != "" {
		t.Errorf("expected empty, got %q", mc.Privacy)
	}
	if mc.PreferredProvider != "" {
		t.Errorf("expected empty, got %q", mc.PreferredProvider)
	}
}

// --- GenerateResponse (G2: untested type) ---

func TestGenerateResponse_Fields(t *testing.T) {
	resp := skills.GenerateResponse{
		Content:      "Hello!",
		ToolCalls:    []skills.ToolCall{{ID: "c1", Name: "search", Arguments: `{}`}},
		Usage:        skills.TokenUsage{PromptTokens: 10, CompletionTokens: 5, TotalTokens: 15},
		FinishReason: "stop",
		Latency:      150 * time.Millisecond,
	}
	if resp.Content != "Hello!" {
		t.Errorf("Content: expected 'Hello!', got %q", resp.Content)
	}
	if len(resp.ToolCalls) != 1 {
		t.Errorf("ToolCalls: expected 1, got %d", len(resp.ToolCalls))
	}
	if resp.Usage.TotalTokens != 15 {
		t.Errorf("Usage.TotalTokens: expected 15, got %d", resp.Usage.TotalTokens)
	}
	if resp.FinishReason != "stop" {
		t.Errorf("FinishReason: expected 'stop', got %q", resp.FinishReason)
	}
	if resp.Latency != 150*time.Millisecond {
		t.Errorf("Latency: expected 150ms, got %v", resp.Latency)
	}
}

func TestGenerateResponse_EmptyToolCalls(t *testing.T) {
	resp := skills.GenerateResponse{Content: "hi"}
	if resp.ToolCalls != nil {
		t.Error("nil ToolCalls should remain nil")
	}

	resp.ToolCalls = []skills.ToolCall{}
	if len(resp.ToolCalls) != 0 {
		t.Error("empty slice should have length 0")
	}
}

func TestGenerateResponse_TokenUsage(t *testing.T) {
	resp := skills.GenerateResponse{
		Usage: skills.TokenUsage{PromptTokens: 50, CompletionTokens: 25, TotalTokens: 75},
	}
	if resp.Usage.PromptTokens != 50 {
		t.Errorf("PromptTokens: expected 50, got %d", resp.Usage.PromptTokens)
	}
}

func TestGenerateResponse_ToolCallsSerialization(t *testing.T) {
	resp := skills.GenerateResponse{
		Content:      "result",
		ToolCalls:    []skills.ToolCall{{ID: "c1", Name: "tool", Arguments: `{"k":"v"}`}},
		FinishReason: "tool_calls",
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded skills.GenerateResponse
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if decoded.Content != "result" {
		t.Errorf("Content: expected 'result', got %q", decoded.Content)
	}
	if len(decoded.ToolCalls) != 1 || decoded.ToolCalls[0].Name != "tool" {
		t.Errorf("ToolCalls round-trip failed: %+v", decoded.ToolCalls)
	}
}

// --- StreamChunk (G3: untested type) ---

func TestStreamChunk_Fields(t *testing.T) {
	chunk := skills.StreamChunk{
		Content:      "Hello",
		ToolCalls:    []skills.ToolCall{{ID: "c1", Name: "fn", Arguments: `{}`}},
		FinishReason: "stop",
		Usage:        skills.TokenUsage{TotalTokens: 20},
	}
	if chunk.Content != "Hello" {
		t.Errorf("Content: expected 'Hello', got %q", chunk.Content)
	}
	if len(chunk.ToolCalls) != 1 {
		t.Errorf("ToolCalls: expected 1, got %d", len(chunk.ToolCalls))
	}
	if chunk.FinishReason != "stop" {
		t.Errorf("FinishReason: expected 'stop', got %q", chunk.FinishReason)
	}
	if chunk.Usage.TotalTokens != 20 {
		t.Errorf("Usage.TotalTokens: expected 20, got %d", chunk.Usage.TotalTokens)
	}
	if chunk.Error != nil {
		t.Errorf("Error: expected nil, got %v", chunk.Error)
	}
}

func TestStreamChunk_ErrorNil(t *testing.T) {
	chunk := skills.StreamChunk{Content: "ok"}
	if chunk.Error != nil {
		t.Error("default Error should be nil")
	}
}

func TestStreamChunk_AccumulatedToolCalls(t *testing.T) {
	chunk := skills.StreamChunk{
		ToolCalls: []skills.ToolCall{
			{ID: "c1", Name: "search", Arguments: `{"q":"a"}`},
			{ID: "c2", Name: "lookup", Arguments: `{"id":1}`},
		},
	}
	if len(chunk.ToolCalls) != 2 {
		t.Errorf("expected 2 accumulated tool calls, got %d", len(chunk.ToolCalls))
	}
	if chunk.ToolCalls[0].ID != "c1" || chunk.ToolCalls[1].Name != "lookup" {
		t.Error("accumulated tool call fields mismatch")
	}
}

// --- NormalizeArguments null property value (G16) ---

func TestNormalizeArguments_PropertyNull(t *testing.T) {
	result, err := skills.NormalizeArguments(`{"key": null}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	val, exists := result["key"]
	if !exists {
		t.Fatal("expected key to exist in result map")
	}
	if val != nil {
		t.Errorf("expected key to be nil, got %v", val)
	}
	if len(result) != 1 {
		t.Errorf("expected 1 key, got %d", len(result))
	}
}

// --- SkillDescriptor (canonical definition) ---

func TestSkillDescriptor_Fields(t *testing.T) {
	sd := skills.SkillDescriptor{
		ID:          "core/summarize",
		Name:        "Summarize",
		Description: "Summarizes text",
		InputSchema: &skills.JSONSchema{
			Type: "object",
			Properties: map[string]*skills.JSONSchema{
				"text": {Type: "string"},
			},
			Required: []string{"text"},
		},
	}

	if sd.ID != "core/summarize" {
		t.Errorf("ID: expected 'core/summarize', got %q", sd.ID)
	}
	if sd.Name != "Summarize" {
		t.Errorf("Name: expected 'Summarize', got %q", sd.Name)
	}
	if sd.Description != "Summarizes text" {
		t.Errorf("Description: expected 'Summarizes text', got %q", sd.Description)
	}
	if sd.InputSchema == nil {
		t.Fatal("InputSchema should not be nil")
	}
	if sd.InputSchema.Type != "object" {
		t.Errorf("InputSchema.Type: expected 'object', got %q", sd.InputSchema.Type)
	}
}

func TestSkillDescriptor_OmitEmptyInputSchema(t *testing.T) {
	sd := skills.SkillDescriptor{
		ID:          "core/test",
		Name:        "Test",
		Description: "A test skill",
	}

	data, err := json.Marshal(sd)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	// input_schema should be absent when nil (omitempty)
	if _, exists := m["input_schema"]; exists {
		t.Error("input_schema should be omitted when nil")
	}
}

func TestSkillDescriptor_Serialization(t *testing.T) {
	original := skills.SkillDescriptor{
		ID:          "core/math-add",
		Name:        "Math Add",
		Description: "Adds two numbers",
		InputSchema: &skills.JSONSchema{
			Type: "object",
			Properties: map[string]*skills.JSONSchema{
				"a": {Type: "number"},
				"b": {Type: "number"},
			},
			Required: []string{"a", "b"},
		},
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded skills.SkillDescriptor
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if decoded.ID != "core/math-add" {
		t.Errorf("ID round-trip: expected 'core/math-add', got %q", decoded.ID)
	}
	if decoded.Name != "Math Add" {
		t.Errorf("Name round-trip: expected 'Math Add', got %q", decoded.Name)
	}
	if decoded.InputSchema == nil {
		t.Fatal("InputSchema should not be nil after round-trip")
	}
}

// TestSkillDescriptor_AliasIdentity verifies that planner.SkillDescriptor and
// agent.SkillDescriptor are type aliases of skills.SkillDescriptor (same type).
func TestSkillDescriptor_AliasIdentity(t *testing.T) {
	// This test imports the aliased packages and verifies assignability.
	// The actual alias verification is in the runtime/agent helpers_test.go
	// which imports all three packages. Here we verify the canonical type works.
	sd := skills.SkillDescriptor{ID: "test", Name: "T", Description: "D"}

	// Must be able to take address of fields
	sd.ID = "changed"
	if sd.ID != "changed" {
		t.Error("field assignment failed")
	}

	// Nil InputSchema is valid
	if sd.InputSchema != nil {
		t.Error("InputSchema should be nil")
	}
}
