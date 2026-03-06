package model

import (
	"context"
	"fmt"
)

// ClaudeProvider implements ModelProvider for Anthropic Claude models.
type ClaudeProvider struct {
	apiKey    string
	modelName string
}

// NewClaudeProvider creates a new Claude provider.
func NewClaudeProvider(apiKey, modelName string) *ClaudeProvider {
	return &ClaudeProvider{
		apiKey:    apiKey,
		modelName: modelName,
	}
}

func (p *ClaudeProvider) ID() string {
	return "anthropic/" + p.modelName
}

func (p *ClaudeProvider) Capabilities() []CapabilityType {
	return []CapabilityType{
		CapTextGeneration,
		CapToolCalling,
		CapVision,
	}
}

func (p *ClaudeProvider) Generate(ctx context.Context, req GenerateRequest) (*GenerateResponse, error) {
	// TODO: Implement actual Claude API call
	// This is a stub for interface compliance
	if p.apiKey == "" {
		return nil, fmt.Errorf("claude: API key not configured")
	}
	return nil, fmt.Errorf("claude: Generate not implemented")
}

func (p *ClaudeProvider) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	return nil, ErrCapabilityNotSupported
}

// OpenAIProvider implements ModelProvider for OpenAI models.
type OpenAIProvider struct {
	apiKey    string
	modelName string
}

// NewOpenAIProvider creates a new OpenAI provider.
func NewOpenAIProvider(apiKey, modelName string) *OpenAIProvider {
	return &OpenAIProvider{
		apiKey:    apiKey,
		modelName: modelName,
	}
}

func (p *OpenAIProvider) ID() string {
	return "openai/" + p.modelName
}

func (p *OpenAIProvider) Capabilities() []CapabilityType {
	return []CapabilityType{
		CapTextGeneration,
		CapToolCalling,
		CapJSONMode,
		CapVision,
	}
}

func (p *OpenAIProvider) Generate(ctx context.Context, req GenerateRequest) (*GenerateResponse, error) {
	// TODO: Implement actual OpenAI API call
	if p.apiKey == "" {
		return nil, fmt.Errorf("openai: API key not configured")
	}
	return nil, fmt.Errorf("openai: Generate not implemented")
}

func (p *OpenAIProvider) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	// OpenAI supports embedding via text-embedding models
	// TODO: Implement
	return nil, ErrCapabilityNotSupported
}

// ModelScopeProvider implements ModelProvider for Alibaba ModelScope.
type ModelScopeProvider struct {
	apiKey    string
	modelName string
}

// NewModelScopeProvider creates a new ModelScope provider.
func NewModelScopeProvider(apiKey, modelName string) *ModelScopeProvider {
	return &ModelScopeProvider{
		apiKey:    apiKey,
		modelName: modelName,
	}
}

func (p *ModelScopeProvider) ID() string {
	return "modelscope/" + p.modelName
}

func (p *ModelScopeProvider) Capabilities() []CapabilityType {
	return []CapabilityType{
		CapTextGeneration,
		CapToolCalling,
	}
}

func (p *ModelScopeProvider) Generate(ctx context.Context, req GenerateRequest) (*GenerateResponse, error) {
	if p.apiKey == "" {
		return nil, fmt.Errorf("modelscope: API key not configured")
	}
	return nil, fmt.Errorf("modelscope: Generate not implemented")
}

func (p *ModelScopeProvider) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	return nil, ErrCapabilityNotSupported
}

// SiliconFlowProvider implements ModelProvider for SiliconFlow gateway.
type SiliconFlowProvider struct {
	apiKey    string
	modelName string
}

// NewSiliconFlowProvider creates a new SiliconFlow provider.
func NewSiliconFlowProvider(apiKey, modelName string) *SiliconFlowProvider {
	return &SiliconFlowProvider{
		apiKey:    apiKey,
		modelName: modelName,
	}
}

func (p *SiliconFlowProvider) ID() string {
	return "siliconflow/" + p.modelName
}

func (p *SiliconFlowProvider) Capabilities() []CapabilityType {
	return []CapabilityType{
		CapTextGeneration,
		CapToolCalling,
	}
}

func (p *SiliconFlowProvider) Generate(ctx context.Context, req GenerateRequest) (*GenerateResponse, error) {
	if p.apiKey == "" {
		return nil, fmt.Errorf("siliconflow: API key not configured")
	}
	return nil, fmt.Errorf("siliconflow: Generate not implemented")
}

func (p *SiliconFlowProvider) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	return nil, ErrCapabilityNotSupported
}
