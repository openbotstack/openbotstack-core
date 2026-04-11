package providers

import (
	"strings"
	"testing"
	"time"

	"github.com/openbotstack/openbotstack-core/control/skills"
)

func TestProviderConfigValidateValid(t *testing.T) {
	cfg := ProviderConfig{
		BaseURL: "http://localhost:8000/v1",
		Model:   "test-model",
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if cfg.Timeout != 120*time.Second {
		t.Errorf("Expected default timeout 120s, got %v", cfg.Timeout)
	}
	if cfg.Capabilities == nil {
		t.Error("Expected default capabilities to be set")
	}
}

func TestProviderConfigValidateEmptyBaseURL(t *testing.T) {
	cfg := ProviderConfig{Model: "test-model"}
	if err := cfg.Validate(); err == nil {
		t.Error("Expected error for empty BaseURL")
	}
}

func TestProviderConfigValidateEmptyModel(t *testing.T) {
	cfg := ProviderConfig{BaseURL: "http://localhost:8000/v1"}
	if err := cfg.Validate(); err == nil {
		t.Error("Expected error for empty Model")
	}
}

func TestProviderConfigValidateInvalidURLScheme(t *testing.T) {
	cfg := ProviderConfig{
		BaseURL: "ftp://localhost:8000/v1",
		Model:   "test-model",
	}
	if err := cfg.Validate(); err == nil {
		t.Error("Expected error for invalid URL scheme")
	}
}

func TestProviderConfigValidateTrailingSlash(t *testing.T) {
	cfg := ProviderConfig{
		BaseURL: "http://localhost:8000/v1/",
		Model:   "test-model",
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if strings.HasSuffix(cfg.BaseURL, "/") {
		t.Error("Expected trailing slash to be trimmed")
	}
}

func TestProviderConfigValidateNegativeMaxRetries(t *testing.T) {
	cfg := ProviderConfig{
		BaseURL:    "http://localhost:8000/v1",
		Model:      "test-model",
		MaxRetries: -5,
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if cfg.MaxRetries != 0 {
		t.Errorf("Expected MaxRetries clamped to 0, got %d", cfg.MaxRetries)
	}
}

func TestProviderConfigValidateNilHeaders(t *testing.T) {
	cfg := ProviderConfig{
		BaseURL: "http://localhost:8000/v1",
		Model:   "test-model",
		Headers: nil,
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("Expected nil Headers to be valid, got: %v", err)
	}
}

func TestProviderConfigValidateEmptyAPIKey(t *testing.T) {
	cfg := ProviderConfig{
		BaseURL: "http://localhost:8000/v1",
		Model:   "test-model",
		APIKey:  "",
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("Expected empty APIKey to be valid for self-hosted, got: %v", err)
	}
}

func TestProviderConfigValidateDefaultCapabilities(t *testing.T) {
	cfg := ProviderConfig{
		BaseURL: "http://localhost:8000/v1",
		Model:   "test-model",
	}
	_ = cfg.Validate()
	expected := []skills.CapabilityType{
		skills.CapTextGeneration,
		skills.CapToolCalling,
		skills.CapStreaming,
	}
	if len(cfg.Capabilities) != len(expected) {
		t.Fatalf("Expected %d capabilities, got %d", len(expected), len(cfg.Capabilities))
	}
	for i, cap := range expected {
		if cfg.Capabilities[i] != cap {
			t.Errorf("Expected capability %s at index %d, got %s", cap, i, cfg.Capabilities[i])
		}
	}
}

func TestProviderConfigValidateCustomCapabilities(t *testing.T) {
	customCaps := []skills.CapabilityType{skills.CapTextGeneration}
	cfg := ProviderConfig{
		BaseURL:      "http://localhost:8000/v1",
		Model:        "test-model",
		Capabilities: customCaps,
	}
	_ = cfg.Validate()
	if len(cfg.Capabilities) != 1 || cfg.Capabilities[0] != skills.CapTextGeneration {
		t.Errorf("Expected custom capabilities to be preserved, got %v", cfg.Capabilities)
	}
}

func TestProviderConfigValidateCustomTimeout(t *testing.T) {
	cfg := ProviderConfig{
		BaseURL: "http://localhost:8000/v1",
		Model:   "test-model",
		Timeout: 30 * time.Second,
	}
	_ = cfg.Validate()
	if cfg.Timeout != 30*time.Second {
		t.Errorf("Expected custom timeout preserved as 30s, got %v", cfg.Timeout)
	}
}

func TestNewProviderFromConfig(t *testing.T) {
	cfg := ProviderConfig{
		BaseURL: "http://localhost:8000/v1",
		APIKey:  "test-key",
		Model:   "test-model",
	}
	provider, err := NewProviderFromConfig(cfg)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if provider.ID() != "openai/test-model" {
		t.Errorf("Expected ID 'openai/test-model', got '%s'", provider.ID())
	}
}

func TestNewProviderFromConfigInvalid(t *testing.T) {
	cfg := ProviderConfig{Model: "test-model"} // Missing BaseURL
	_, err := NewProviderFromConfig(cfg)
	if err == nil {
		t.Error("Expected error for invalid config")
	}
}

func TestNewProviderFromConfigInterfaceCompliance(t *testing.T) {
	cfg := ProviderConfig{
		BaseURL: "http://localhost:8000/v1",
		Model:   "test-model",
	}
	provider, err := NewProviderFromConfig(cfg)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	// Verify it satisfies ModelProvider
	var _ ModelProvider = provider
	// Verify it satisfies StreamingModelProvider
	var _ StreamingModelProvider = provider.(*openAIProvider)
}

func TestNewProviderFromConfigCapabilities(t *testing.T) {
	cfg := ProviderConfig{
		BaseURL: "http://localhost:8000/v1",
		Model:   "test-model",
	}
	provider, _ := NewProviderFromConfig(cfg)
	caps := provider.Capabilities()
	hasStreaming := false
	for _, c := range caps {
		if c == skills.CapStreaming {
			hasStreaming = true
		}
	}
	if !hasStreaming {
		t.Error("Expected CapStreaming in default capabilities")
	}
}

func TestStreamingTypeAssertion(t *testing.T) {
	// Verify that old providers do NOT satisfy StreamingModelProvider
	oldProvider := NewOpenAIProvider("", "key", "gpt-4o")
	var oldIface ModelProvider = oldProvider
	if _, ok := oldIface.(StreamingModelProvider); ok {
		t.Error("Old OpenAIProvider should NOT satisfy StreamingModelProvider")
	}

	// Verify new provider DOES satisfy StreamingModelProvider
	cfg := ProviderConfig{
		BaseURL: "http://localhost:8000/v1",
		Model:   "test-model",
	}
	newProvider, _ := NewProviderFromConfig(cfg)
	if _, ok := newProvider.(StreamingModelProvider); !ok {
		t.Error("NewProviderFromConfig result should satisfy StreamingModelProvider")
	}
}
