package providers

import (
	"context"
	"testing"

	"github.com/openbotstack/openbotstack-core/ai"
	"github.com/openbotstack/openbotstack-core/control/skills"
)

// MockProvider is a test implementation of ModelProvider.
type MockProvider struct {
	id           string
	capabilities []skills.CapabilityType
	generateFunc func(ctx context.Context, req skills.GenerateRequest) (*skills.GenerateResponse, error)
}

func (m *MockProvider) ID() string {
	return m.id
}

func (m *MockProvider) Capabilities() []skills.CapabilityType {
	return m.capabilities
}

func (m *MockProvider) Generate(ctx context.Context, req skills.GenerateRequest) (*skills.GenerateResponse, error) {
	if m.generateFunc != nil {
		return m.generateFunc(ctx, req)
	}
	return &skills.GenerateResponse{Content: "mock response"}, nil
}

func (m *MockProvider) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	return nil, ai.ErrCapabilityNotSupported
}

func TestCapabilityTypes(t *testing.T) {
	// Verify capability constants are defined
	caps := []skills.CapabilityType{
		skills.CapTextGeneration,
		skills.CapToolCalling,
		skills.CapJSONMode,
		skills.CapEmbedding,
		skills.CapVision,
		skills.CapStreaming,
	}

	for _, cap := range caps {
		if cap == "" {
			t.Errorf("Capability type should not be empty")
		}
	}
}

func TestModelProviderInterface(t *testing.T) {
	// Test that MockProvider satisfies ModelProvider interface
	var _ ModelProvider = &MockProvider{}

	provider := &MockProvider{
		id:           "test/mock",
		capabilities: []skills.CapabilityType{skills.CapTextGeneration},
	}

	if provider.ID() != "test/mock" {
		t.Errorf("Expected ID 'test/mock', got '%s'", provider.ID())
	}

	caps := provider.Capabilities()
	if len(caps) != 1 || caps[0] != skills.CapTextGeneration {
		t.Errorf("Unexpected capabilities: %v", caps)
	}
}

func TestGenerateRequest(t *testing.T) {
	req := skills.GenerateRequest{
		Messages: []skills.Message{
			{Role: "user", Content: "Hello"},
		},
		MaxTokens:   100,
		Temperature: 0.7,
	}

	if len(req.Messages) != 1 {
		t.Errorf("Expected 1 message, got %d", len(req.Messages))
	}
}

func TestGenerateResponse(t *testing.T) {
	resp := skills.GenerateResponse{
		Content:      "Hello back",
		FinishReason: "stop",
		Usage: skills.TokenUsage{
			PromptTokens:     10,
			CompletionTokens: 5,
			TotalTokens:      15,
		},
	}

	if resp.Usage.TotalTokens != 15 {
		t.Errorf("Expected 15 total tokens, got %d", resp.Usage.TotalTokens)
	}
}

func TestStreamChunk(t *testing.T) {
	chunk := skills.StreamChunk{
		Content:      "Hello",
		ToolCalls:    []skills.ToolCall{{ID: "call_1", Name: "test_tool", Arguments: `{"key":"value"}`}},
		FinishReason: "stop",
		Usage:        skills.TokenUsage{PromptTokens: 10, CompletionTokens: 5, TotalTokens: 15},
	}
	if chunk.Content != "Hello" {
		t.Errorf("Expected Content 'Hello', got '%s'", chunk.Content)
	}
	if len(chunk.ToolCalls) != 1 {
		t.Fatalf("Expected 1 ToolCall, got %d", len(chunk.ToolCalls))
	}
	if chunk.ToolCalls[0].Name != "test_tool" {
		t.Errorf("Expected ToolCall Name 'test_tool', got '%s'", chunk.ToolCalls[0].Name)
	}
	if chunk.FinishReason != "stop" {
		t.Errorf("Expected FinishReason 'stop', got '%s'", chunk.FinishReason)
	}
	if chunk.Usage.TotalTokens != 15 {
		t.Errorf("Expected TotalTokens 15, got %d", chunk.Usage.TotalTokens)
	}
}

func TestStreamChunkError(t *testing.T) {
	chunk := skills.StreamChunk{
		Error: context.Canceled,
	}
	if chunk.Error == nil {
		t.Error("Expected non-nil Error")
	}
	if chunk.Error.Error() != "context canceled" {
		t.Errorf("Expected 'context canceled', got '%s'", chunk.Error.Error())
	}
}

func TestCapStreaming(t *testing.T) {
	if skills.CapStreaming != "streaming" {
		t.Errorf("Expected 'streaming', got '%s'", skills.CapStreaming)
	}
}

// MockStreamingProvider is a test implementation of StreamingModelProvider.
type MockStreamingProvider struct {
	MockProvider
}

func (m *MockStreamingProvider) GenerateStream(ctx context.Context, req skills.GenerateRequest) (<-chan skills.StreamChunk, error) {
	ch := make(chan skills.StreamChunk, 1)
	ch <- skills.StreamChunk{Content: "mock stream", FinishReason: "stop"}
	close(ch)
	return ch, nil
}

func TestStreamingModelProviderInterface(t *testing.T) {
	// Verify MockStreamingProvider satisfies StreamingModelProvider
	var _ StreamingModelProvider = &MockStreamingProvider{}

	// Verify it also satisfies ModelProvider (embedded)
	var _ ModelProvider = &MockStreamingProvider{}
}

func TestNewErrorTypes(t *testing.T) {
	if ai.ErrProviderUnavailable == nil {
		t.Error("ErrProviderUnavailable should not be nil")
	}
	if ai.ErrBadRequest == nil {
		t.Error("ErrBadRequest should not be nil")
	}
	if ai.ErrAuthenticationFailed == nil {
		t.Error("ErrAuthenticationFailed should not be nil")
	}
}
