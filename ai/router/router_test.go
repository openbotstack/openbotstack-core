package router_test

import (
	"github.com/openbotstack/openbotstack-core/ai"
	"github.com/openbotstack/openbotstack-core/ai/providers"
	"github.com/openbotstack/openbotstack-core/ai/router"
	"github.com/openbotstack/openbotstack-core/control/skills"
	"testing"
)

func TestDefaultRouterRegistration(t *testing.T) {
	router := router.NewDefaultRouter()

	claude := providers.NewClaudeProvider("", "key", "claude-3-opus-20240229")
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
	router := router.NewDefaultRouter()

	claude := providers.NewClaudeProvider("", "key", "claude-3-opus-20240229")
	_ = router.Register(claude)
	err := router.Register(claude)

	if err != ai.ErrProviderAlreadyExists {
		t.Errorf("Expected ai.ErrProviderAlreadyExists, got %v", err)
	}
}

func TestDefaultRouterRoute(t *testing.T) {
	router := router.NewDefaultRouter()
	claude := providers.NewClaudeProvider("", "key", "claude-3-opus-20240229")
	openai := providers.NewOpenAIProvider("", "key", "gpt-4o")
	_ = router.Register(claude)
	_ = router.Register(openai)

	// Route for text_generation + tool_calling
	provider, err := router.Route(
		[]skills.CapabilityType{skills.CapTextGeneration, skills.CapToolCalling},
		skills.ModelConstraints{},
	)
	if err != nil {
		t.Fatalf("Route failed: %v", err)
	}
	if provider == nil {
		t.Fatal("Route returned nil provider")
	}
}

func TestDefaultRouterRouteNoMatch(t *testing.T) {
	router := router.NewDefaultRouter()
	claude := providers.NewClaudeProvider("", "key", "claude-3-opus-20240229")
	_ = router.Register(claude)

	// Route for embedding (Claude doesn't support)
	_, err := router.Route(
		[]skills.CapabilityType{skills.CapEmbedding},
		skills.ModelConstraints{},
	)

	if err != ai.ErrNoMatchingProvider {
		t.Errorf("Expected ai.ErrNoMatchingProvider, got %v", err)
	}
}

func TestDefaultRouterRoutePreferredProvider(t *testing.T) {
	router := router.NewDefaultRouter()
	claude := providers.NewClaudeProvider("", "key", "claude-3-opus-20240229")
	openai := providers.NewOpenAIProvider("", "key", "gpt-4o")
	_ = router.Register(claude)
	_ = router.Register(openai)

	// Route with preferred provider
	provider, err := router.Route(
		[]skills.CapabilityType{skills.CapTextGeneration},
		skills.ModelConstraints{PreferredProvider: "anthropic/claude-3-opus-20240229"},
	)
	if err != nil {
		t.Fatalf("Route failed: %v", err)
	}
	if provider.ID() != "anthropic/claude-3-opus-20240229" {
		t.Errorf("Expected preferred provider, got %s", provider.ID())
	}
}

