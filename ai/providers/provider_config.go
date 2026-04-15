package providers

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/openbotstack/openbotstack-core/control/skills"
)

// ProviderConfig holds configuration for any OpenAI-compatible LLM endpoint.
type ProviderConfig struct {
	// BaseURL is the API endpoint base URL. Required.
	// e.g. "http://localhost:8000/v1"
	BaseURL string
	// APIKey is the authentication key. Optional for self-hosted endpoints.
	APIKey string
	// Model is the model identifier. Required.
	// e.g. "Qwen/Qwen2.5-72B-Instruct"
	Model string
	// Headers are custom HTTP headers sent with every request.
	Headers map[string]string
	// Timeout is the per-request HTTP timeout. Defaults to 120s.
	Timeout time.Duration
	// MaxRetries is the number of retries for 5xx/network errors. Defaults to 0.
	MaxRetries int
	// Capabilities declares what the model supports. Defaults to TextGeneration, ToolCalling, Streaming.
	Capabilities []skills.CapabilityType
}

// Validate checks the config and applies defaults. Returns error if required fields are missing.
func (c *ProviderConfig) Validate() error {
	if c.BaseURL == "" {
		return fmt.Errorf("provider config: BaseURL is required")
	}
	if !strings.HasPrefix(c.BaseURL, "http://") && !strings.HasPrefix(c.BaseURL, "https://") {
		return fmt.Errorf("provider config: BaseURL must start with http:// or https://")
	}
	if c.Model == "" {
		return fmt.Errorf("provider config: Model is required")
	}

	// Trim trailing slash
	c.BaseURL = strings.TrimRight(c.BaseURL, "/")

	// Apply defaults
	if c.Timeout == 0 {
		c.Timeout = 120 * time.Second
	}
	if c.MaxRetries < 0 {
		c.MaxRetries = 0
	}
	if c.Capabilities == nil {
		c.Capabilities = []skills.CapabilityType{
			skills.CapTextGeneration,
			skills.CapToolCalling,
			skills.CapStreaming,
		}
	}

	return nil
}

// openAIProvider is a unified provider for any OpenAI-compatible endpoint.
type openAIProvider struct {
	baseURL      string
	apiKey       string
	modelName    string
	headers      map[string]string
	client       *http.Client
	capabilities []skills.CapabilityType
	maxRetries   int
}

// NewProviderFromConfig creates a ModelProvider from ProviderConfig.
// The returned provider implements StreamingModelProvider.
func NewProviderFromConfig(cfg ProviderConfig) (ModelProvider, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return &openAIProvider{
		baseURL:      cfg.BaseURL,
		apiKey:       cfg.APIKey,
		modelName:    cfg.Model,
		headers:      cfg.Headers,
		client:       &http.Client{Timeout: cfg.Timeout},
		capabilities: cfg.Capabilities,
		maxRetries:   cfg.MaxRetries,
	}, nil
}

func (p *openAIProvider) ID() string {
	return "openai/" + p.modelName
}

func (p *openAIProvider) Capabilities() []skills.CapabilityType {
	return p.capabilities
}

func (p *openAIProvider) Generate(ctx context.Context, req skills.GenerateRequest) (*skills.GenerateResponse, error) {
	return openAICompatibleGenerate(ctx, p.client, p.baseURL, p.apiKey, p.modelName, p.headers, req, p.maxRetries)
}

func (p *openAIProvider) GenerateStream(ctx context.Context, req skills.GenerateRequest) (<-chan skills.StreamChunk, error) {
	return openAICompatibleStream(ctx, p.client, p.baseURL, p.apiKey, p.modelName, p.headers, req, p.maxRetries)
}

func (p *openAIProvider) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	model := "text-embedding-3-small"
	return openAICompatibleEmbed(ctx, p.client, p.baseURL, p.apiKey, model, p.headers, texts, 0)
}
