package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/openbotstack/openbotstack-core/ai"
	"github.com/openbotstack/openbotstack-core/control/skills"
)

// Default base URLs for known providers.
const (
	defaultOpenAIBaseURL      = "https://api.openai.com/v1"
	defaultClaudeBaseURL      = "https://api.anthropic.com/v1"
	defaultModelScopeBaseURL  = "https://api-inference.modelscope.cn/v1"
	defaultSiliconFlowBaseURL = "https://api.siliconflow.cn/v1"

	claudeAPIVersion = "2023-06-01"

	httpTimeoutSeconds = 120
)

// ----- OpenAI-compatible request/response types -----

type chatRequest struct {
	Model       string        `json:"model"`
	Messages    []chatMessage `json:"messages"`
	MaxTokens   int           `json:"max_tokens,omitempty"`
	Temperature *float64      `json:"temperature,omitempty"`
	Tools       []chatTool    `json:"tools,omitempty"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
	Name    string `json:"name,omitempty"`
}

type chatTool struct {
	Type     string       `json:"type"`
	Function chatFunction `json:"function"`
}

type chatFunction struct {
	Name        string           `json:"name"`
	Description string           `json:"description,omitempty"`
	Parameters  *skills.JSONSchema `json:"parameters,omitempty"`
}

type chatResponse struct {
	Choices []chatChoice `json:"choices"`
	Usage   chatUsage    `json:"usage"`
	Error   *chatError   `json:"error,omitempty"`
}

type chatChoice struct {
	Message      chatResponseMessage `json:"message"`
	FinishReason string              `json:"finish_reason"`
}

type chatResponseMessage struct {
	Role      string              `json:"role"`
	Content   string              `json:"content"`
	ToolCalls []chatToolCall      `json:"tool_calls,omitempty"`
}

type chatToolCall struct {
	ID       string           `json:"id"`
	Type     string           `json:"type"`
	Function chatFunctionCall `json:"function"`
}

type chatFunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type chatUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type chatError struct {
	Message string `json:"message"`
	Type    string `json:"type"`
}

// openAICompatibleGenerate sends a chat completion request to any
// OpenAI-compatible endpoint and returns a GenerateResponse.
func openAICompatibleGenerate(
	ctx context.Context,
	client *http.Client,
	baseURL, apiKey, model string,
	headers map[string]string,
	req skills.GenerateRequest,
) (*skills.GenerateResponse, error) {
	// Build messages
	messages := make([]chatMessage, 0, len(req.Messages))
	for _, m := range req.Messages {
		messages = append(messages, chatMessage{
			Role:    m.Role,
			Content: m.Content,
			Name:    m.Name,
		})
	}

	// Build tools
	var tools []chatTool
	for _, t := range req.Tools {
		tools = append(tools, chatTool{
			Type: "function",
			Function: chatFunction{
				Name:        t.Name,
				Description: t.Description,
				Parameters:  t.Parameters,
			},
		})
	}

	body := chatRequest{
		Model:    model,
		Messages: messages,
		Tools:    tools,
	}
	if req.MaxTokens > 0 {
		body.MaxTokens = req.MaxTokens
	}
	if req.Temperature > 0 {
		temp := req.Temperature
		body.Temperature = &temp
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	endpoint := strings.TrimRight(baseURL, "/") + "/chat/completions"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+apiKey)
	for k, v := range headers {
		httpReq.Header.Set(k, v)
	}

	start := time.Now()
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("api error (status %d): %s", resp.StatusCode, string(respBody))
	}

	var chatResp chatResponse
	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	if chatResp.Error != nil {
		return nil, fmt.Errorf("api error: %s (%s)", chatResp.Error.Message, chatResp.Error.Type)
	}

	latency := time.Since(start)

	result := &skills.GenerateResponse{
		Usage: skills.TokenUsage{
			PromptTokens:     chatResp.Usage.PromptTokens,
			CompletionTokens: chatResp.Usage.CompletionTokens,
			TotalTokens:      chatResp.Usage.TotalTokens,
		},
		Latency: latency,
	}

	if len(chatResp.Choices) > 0 {
		choice := chatResp.Choices[0]
		result.Content = choice.Message.Content
		result.FinishReason = choice.FinishReason

		for _, tc := range choice.Message.ToolCalls {
			result.ToolCalls = append(result.ToolCalls, skills.ToolCall{
				ID:        tc.ID,
				Name:      tc.Function.Name,
				Arguments: tc.Function.Arguments,
			})
		}
	}

	return result, nil
}

// ----- Provider implementations -----

// OpenAIProvider implements ModelProvider for OpenAI models.
type OpenAIProvider struct {
	baseURL   string
	apiKey    string
	modelName string
	client    *http.Client
}

// NewOpenAIProvider creates a new OpenAI provider.
// If baseURL is empty, defaults to the official OpenAI API.
func NewOpenAIProvider(baseURL, apiKey, modelName string) *OpenAIProvider {
	if baseURL == "" {
		baseURL = defaultOpenAIBaseURL
	}
	return &OpenAIProvider{
		baseURL:   baseURL,
		apiKey:    apiKey,
		modelName: modelName,
		client:    &http.Client{Timeout: httpTimeoutSeconds * time.Second},
	}
}

func (p *OpenAIProvider) ID() string {
	return "openai/" + p.modelName
}

func (p *OpenAIProvider) Capabilities() []skills.CapabilityType {
	return []skills.CapabilityType{
		skills.CapTextGeneration,
		skills.CapToolCalling,
		skills.CapJSONMode,
		skills.CapVision,
	}
}

func (p *OpenAIProvider) Generate(ctx context.Context, req skills.GenerateRequest) (*skills.GenerateResponse, error) {
	if p.apiKey == "" {
		return nil, fmt.Errorf("openai: API key not configured")
	}
	return openAICompatibleGenerate(ctx, p.client, p.baseURL, p.apiKey, p.modelName, nil, req)
}

func (p *OpenAIProvider) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	// TODO(phase-3): Implement OpenAI embedding endpoint (/v1/embeddings)
	return nil, ai.ErrCapabilityNotSupported
}

// ClaudeProvider implements ModelProvider for Anthropic Claude models.
// Claude uses the Messages API (/v1/messages) but this provider wraps it
// via OpenAI-compatible proxies (e.g., LiteLLM, AWS Bedrock).
// For direct Anthropic API access, set baseURL to a proxy that translates
// OpenAI-format requests to Claude Messages format.
type ClaudeProvider struct {
	baseURL   string
	apiKey    string
	modelName string
	client    *http.Client
}

// NewClaudeProvider creates a new Claude provider.
// If baseURL is empty, defaults to the Anthropic API.
func NewClaudeProvider(baseURL, apiKey, modelName string) *ClaudeProvider {
	if baseURL == "" {
		baseURL = defaultClaudeBaseURL
	}
	return &ClaudeProvider{
		baseURL:   baseURL,
		apiKey:    apiKey,
		modelName: modelName,
		client:    &http.Client{Timeout: httpTimeoutSeconds * time.Second},
	}
}

func (p *ClaudeProvider) ID() string {
	return "anthropic/" + p.modelName
}

func (p *ClaudeProvider) Capabilities() []skills.CapabilityType {
	return []skills.CapabilityType{
		skills.CapTextGeneration,
		skills.CapToolCalling,
		skills.CapVision,
	}
}

func (p *ClaudeProvider) Generate(ctx context.Context, req skills.GenerateRequest) (*skills.GenerateResponse, error) {
	if p.apiKey == "" {
		return nil, fmt.Errorf("claude: API key not configured")
	}
	headers := map[string]string{
		"x-api-key":         p.apiKey,
		"anthropic-version": claudeAPIVersion,
	}
	return openAICompatibleGenerate(ctx, p.client, p.baseURL, p.apiKey, p.modelName, headers, req)
}

func (p *ClaudeProvider) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	return nil, ai.ErrCapabilityNotSupported
}

// ModelScopeProvider implements ModelProvider for Alibaba ModelScope.
type ModelScopeProvider struct {
	baseURL   string
	apiKey    string
	modelName string
	client    *http.Client
}

// NewModelScopeProvider creates a new ModelScope provider.
func NewModelScopeProvider(baseURL, apiKey, modelName string) *ModelScopeProvider {
	if baseURL == "" {
		baseURL = defaultModelScopeBaseURL
	}
	return &ModelScopeProvider{
		baseURL:   baseURL,
		apiKey:    apiKey,
		modelName: modelName,
		client:    &http.Client{Timeout: httpTimeoutSeconds * time.Second},
	}
}

func (p *ModelScopeProvider) ID() string {
	return "modelscope/" + p.modelName
}

func (p *ModelScopeProvider) Capabilities() []skills.CapabilityType {
	return []skills.CapabilityType{
		skills.CapTextGeneration,
		skills.CapToolCalling,
	}
}

func (p *ModelScopeProvider) Generate(ctx context.Context, req skills.GenerateRequest) (*skills.GenerateResponse, error) {
	if p.apiKey == "" {
		return nil, fmt.Errorf("modelscope: API key not configured")
	}
	return openAICompatibleGenerate(ctx, p.client, p.baseURL, p.apiKey, p.modelName, nil, req)
}

func (p *ModelScopeProvider) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	return nil, ai.ErrCapabilityNotSupported
}

// SiliconFlowProvider implements ModelProvider for SiliconFlow gateway.
type SiliconFlowProvider struct {
	baseURL   string
	apiKey    string
	modelName string
	client    *http.Client
}

// NewSiliconFlowProvider creates a new SiliconFlow provider.
func NewSiliconFlowProvider(baseURL, apiKey, modelName string) *SiliconFlowProvider {
	if baseURL == "" {
		baseURL = defaultSiliconFlowBaseURL
	}
	return &SiliconFlowProvider{
		baseURL:   baseURL,
		apiKey:    apiKey,
		modelName: modelName,
		client:    &http.Client{Timeout: httpTimeoutSeconds * time.Second},
	}
}

func (p *SiliconFlowProvider) ID() string {
	return "siliconflow/" + p.modelName
}

func (p *SiliconFlowProvider) Capabilities() []skills.CapabilityType {
	return []skills.CapabilityType{
		skills.CapTextGeneration,
		skills.CapToolCalling,
	}
}

func (p *SiliconFlowProvider) Generate(ctx context.Context, req skills.GenerateRequest) (*skills.GenerateResponse, error) {
	if p.apiKey == "" {
		return nil, fmt.Errorf("siliconflow: API key not configured")
	}
	return openAICompatibleGenerate(ctx, p.client, p.baseURL, p.apiKey, p.modelName, nil, req)
}

func (p *SiliconFlowProvider) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	return nil, ai.ErrCapabilityNotSupported
}
