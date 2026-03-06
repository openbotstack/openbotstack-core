package model_test

import (
	"context"
	"testing"

	"github.com/openbotstack/openbotstack-core/model"
)

func TestClaudeProviderID(t *testing.T) {
	provider := model.NewClaudeProvider("test-api-key", "claude-3-opus-20240229")
	if provider.ID() != "anthropic/claude-3-opus-20240229" {
		t.Errorf("Expected ID 'anthropic/claude-3-opus-20240229', got '%s'", provider.ID())
	}
}

func TestClaudeProviderCapabilities(t *testing.T) {
	provider := model.NewClaudeProvider("test-api-key", "claude-3-opus-20240229")
	caps := provider.Capabilities()

	expected := []model.CapabilityType{
		model.CapTextGeneration,
		model.CapToolCalling,
		model.CapVision,
	}

	if len(caps) != len(expected) {
		t.Fatalf("Expected %d capabilities, got %d", len(expected), len(caps))
	}

	for i, cap := range expected {
		if caps[i] != cap {
			t.Errorf("Expected capability %s at index %d, got %s", cap, i, caps[i])
		}
	}
}

func TestOpenAIProviderID(t *testing.T) {
	provider := model.NewOpenAIProvider("test-api-key", "gpt-4o")
	if provider.ID() != "openai/gpt-4o" {
		t.Errorf("Expected ID 'openai/gpt-4o', got '%s'", provider.ID())
	}
}

func TestOpenAIProviderCapabilities(t *testing.T) {
	provider := model.NewOpenAIProvider("test-api-key", "gpt-4o")
	caps := provider.Capabilities()

	expected := []model.CapabilityType{
		model.CapTextGeneration,
		model.CapToolCalling,
		model.CapJSONMode,
		model.CapVision,
	}

	if len(caps) != len(expected) {
		t.Fatalf("Expected %d capabilities, got %d", len(expected), len(caps))
	}
}

func TestModelScopeProviderID(t *testing.T) {
	provider := model.NewModelScopeProvider("test-api-key", "qwen-max")
	if provider.ID() != "modelscope/qwen-max" {
		t.Errorf("Expected ID 'modelscope/qwen-max', got '%s'", provider.ID())
	}
}

func TestSiliconFlowProviderID(t *testing.T) {
	provider := model.NewSiliconFlowProvider("test-api-key", "deepseek-v3")
	if provider.ID() != "siliconflow/deepseek-v3" {
		t.Errorf("Expected ID 'siliconflow/deepseek-v3', got '%s'", provider.ID())
	}
}

func TestDefaultRouterRegistration(t *testing.T) {
	router := model.NewDefaultRouter()

	claude := model.NewClaudeProvider("key", "claude-3-opus-20240229")
	err := router.Register(claude)
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	providers := router.List()
	if len(providers) != 1 {
		t.Errorf("Expected 1 provider, got %d", len(providers))
	}
}

func TestDefaultRouterDuplicateRegistration(t *testing.T) {
	router := model.NewDefaultRouter()

	claude := model.NewClaudeProvider("key", "claude-3-opus-20240229")
	_ = router.Register(claude)
	err := router.Register(claude)

	if err != model.ErrProviderAlreadyExists {
		t.Errorf("Expected ErrProviderAlreadyExists, got %v", err)
	}
}

func TestDefaultRouterRoute(t *testing.T) {
	router := model.NewDefaultRouter()
	claude := model.NewClaudeProvider("key", "claude-3-opus-20240229")
	openai := model.NewOpenAIProvider("key", "gpt-4o")
	_ = router.Register(claude)
	_ = router.Register(openai)

	// Route for text_generation + tool_calling
	provider, err := router.Route(
		[]model.CapabilityType{model.CapTextGeneration, model.CapToolCalling},
		model.ModelConstraints{},
	)
	if err != nil {
		t.Fatalf("Route failed: %v", err)
	}
	if provider == nil {
		t.Fatal("Route returned nil provider")
	}
}

func TestDefaultRouterRouteNoMatch(t *testing.T) {
	router := model.NewDefaultRouter()
	claude := model.NewClaudeProvider("key", "claude-3-opus-20240229")
	_ = router.Register(claude)

	// Route for embedding (Claude doesn't support)
	_, err := router.Route(
		[]model.CapabilityType{model.CapEmbedding},
		model.ModelConstraints{},
	)

	if err != model.ErrNoMatchingProvider {
		t.Errorf("Expected ErrNoMatchingProvider, got %v", err)
	}
}

func TestDefaultRouterRoutePreferredProvider(t *testing.T) {
	router := model.NewDefaultRouter()
	claude := model.NewClaudeProvider("key", "claude-3-opus-20240229")
	openai := model.NewOpenAIProvider("key", "gpt-4o")
	_ = router.Register(claude)
	_ = router.Register(openai)

	// Route with preferred provider
	provider, err := router.Route(
		[]model.CapabilityType{model.CapTextGeneration},
		model.ModelConstraints{PreferredProvider: "anthropic/claude-3-opus-20240229"},
	)
	if err != nil {
		t.Fatalf("Route failed: %v", err)
	}
	if provider.ID() != "anthropic/claude-3-opus-20240229" {
		t.Errorf("Expected preferred provider, got %s", provider.ID())
	}
}

func TestProviderGenerateStub(t *testing.T) {
	provider := model.NewClaudeProvider("test-key", "claude-3-opus-20240229")
	ctx := context.Background()

	req := model.GenerateRequest{
		Messages: []model.Message{
			{Role: "user", Content: "Hello"},
		},
		MaxTokens: 100,
	}

	// Stub implementation should return placeholder
	resp, err := provider.Generate(ctx, req)
	if err != nil {
		// Expected for stub without real API key
		t.Logf("Generate returned error (expected for stub): %v", err)
	} else if resp != nil {
		t.Logf("Generate returned response: %s", resp.Content)
	}
}
