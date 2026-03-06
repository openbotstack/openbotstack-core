package model_test

import (
	"context"
	"testing"

	"github.com/openbotstack/openbotstack-core/model"
)

// MockProvider is a test implementation of ModelProvider.
type MockProvider struct {
	id           string
	capabilities []model.CapabilityType
	generateFunc func(ctx context.Context, req model.GenerateRequest) (*model.GenerateResponse, error)
}

func (m *MockProvider) ID() string {
	return m.id
}

func (m *MockProvider) Capabilities() []model.CapabilityType {
	return m.capabilities
}

func (m *MockProvider) Generate(ctx context.Context, req model.GenerateRequest) (*model.GenerateResponse, error) {
	if m.generateFunc != nil {
		return m.generateFunc(ctx, req)
	}
	return &model.GenerateResponse{Content: "mock response"}, nil
}

func (m *MockProvider) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	return nil, model.ErrCapabilityNotSupported
}

func TestCapabilityTypes(t *testing.T) {
	// Verify capability constants are defined
	caps := []model.CapabilityType{
		model.CapTextGeneration,
		model.CapToolCalling,
		model.CapJSONMode,
		model.CapEmbedding,
		model.CapVision,
	}

	for _, cap := range caps {
		if cap == "" {
			t.Errorf("Capability type should not be empty")
		}
	}
}

func TestModelProviderInterface(t *testing.T) {
	// Test that MockProvider satisfies ModelProvider interface
	var _ model.ModelProvider = &MockProvider{}

	provider := &MockProvider{
		id:           "test/mock",
		capabilities: []model.CapabilityType{model.CapTextGeneration},
	}

	if provider.ID() != "test/mock" {
		t.Errorf("Expected ID 'test/mock', got '%s'", provider.ID())
	}

	caps := provider.Capabilities()
	if len(caps) != 1 || caps[0] != model.CapTextGeneration {
		t.Errorf("Unexpected capabilities: %v", caps)
	}
}

func TestGenerateRequest(t *testing.T) {
	req := model.GenerateRequest{
		Messages: []model.Message{
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
	resp := model.GenerateResponse{
		Content:      "Hello back",
		FinishReason: "stop",
		Usage: model.TokenUsage{
			PromptTokens:     10,
			CompletionTokens: 5,
			TotalTokens:      15,
		},
	}

	if resp.Usage.TotalTokens != 15 {
		t.Errorf("Expected 15 total tokens, got %d", resp.Usage.TotalTokens)
	}
}
