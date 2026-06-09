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
	"github.com/openbotstack/openbotstack-core/ai/types"
)

// Default base URLs for known providers.
const (
	defaultOpenAIBaseURL      = "https://api.openai.com/v1"
	defaultClaudeBaseURL      = "https://api.anthropic.com/v1"
	defaultModelScopeBaseURL  = "https://api-inference.modelscope.cn/v1"
	defaultSiliconFlowBaseURL = "https://api.siliconflow.cn/v1"

	claudeAPIVersion = "2023-06-01"

	httpTimeoutSeconds = 300
)

// ----- OpenAI-compatible request/response types -----

type chatRequest struct {
	Model              string                 `json:"model"`
	Messages           []chatMessage          `json:"messages"`
	MaxTokens          int                    `json:"max_tokens,omitempty"`
	Temperature        *float64               `json:"temperature,omitempty"`
	Tools              []chatTool             `json:"tools,omitempty"`
	ToolChoice         any                    `json:"tool_choice,omitempty"`
	ParallelToolCalls  *bool                  `json:"parallel_tool_calls,omitempty"`
	ChatTemplateKwargs map[string]interface{} `json:"chat_template_kwargs,omitempty"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content any    `json:"content"`
	Name    string `json:"name,omitempty"`
}

// openAIContentBlock represents a single content part in a multimodal message.
type openAIContentBlock struct {
	Type     string          `json:"type"`
	Text     string          `json:"text,omitempty"`
	ImageURL *openAIImageURL `json:"image_url,omitempty"`
}

type openAIImageURL struct {
	URL string `json:"url"`
}

// contentsToOpenAI converts ContentBlocks to OpenAI content format.
// Single text block → plain string (backward compatible).
// Multimodal → array of content blocks.
func contentsToOpenAI(contents []types.ContentBlock) any {
	if len(contents) == 0 {
		return nil
	}
	if len(contents) == 1 && contents[0].Type == "text" {
		return contents[0].Text
	}
	blocks := make([]openAIContentBlock, 0, len(contents))
	for _, c := range contents {
		switch c.Type {
		case "text":
			blocks = append(blocks, openAIContentBlock{Type: "text", Text: c.Text})
		case "image":
			blocks = append(blocks, openAIContentBlock{
				Type:     "image_url",
				ImageURL: &openAIImageURL{URL: c.ImageRef()},
			})
		}
	}
	return blocks
}

// buildOpenAIMessages converts GenerateRequest messages to OpenAI chat messages.
func buildOpenAIMessages(req types.GenerateRequest) []chatMessage {
	messages := make([]chatMessage, 0, len(req.Messages))
	for _, m := range req.Messages {
		messages = append(messages, chatMessage{
			Role:    m.Role,
			Content: contentsToOpenAI(m.Contents),
			Name:    m.Name,
		})
	}
	return messages
}

type chatTool struct {
	Type     string       `json:"type"`
	Function chatFunction `json:"function"`
}

type chatFunction struct {
	Name        string           `json:"name"`
	Description string           `json:"description,omitempty"`
	Parameters  *types.JSONSchema `json:"parameters,omitempty"`
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
	Role             string         `json:"role"`
	Content          string         `json:"content"`
	ReasoningContent string         `json:"reasoning_content,omitempty"`
	ToolCalls        []chatToolCall `json:"tool_calls,omitempty"`
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

// ----- OpenAI-compatible Embeddings request/response types -----

type embedRequest struct {
	Model      string   `json:"model"`
	Input      []string `json:"input"`
	Dimensions int      `json:"dimensions,omitempty"`
}

type embedResponse struct {
	Object string        `json:"object"`
	Data   []embedData   `json:"data"`
	Model  string        `json:"model"`
	Usage  embedUsage    `json:"usage"`
	Error  *chatError    `json:"error,omitempty"`
}

type embedData struct {
	Object    string    `json:"object"`
	Embedding []float32 `json:"embedding"`
	Index     int       `json:"index"`
}

type embedUsage struct {
	PromptTokens int `json:"prompt_tokens"`
	TotalTokens  int `json:"total_tokens"`
}

// openAICompatibleGenerate sends a chat completion request to any
// OpenAI-compatible endpoint and returns a GenerateResponse.
func openAICompatibleGenerate(
	ctx context.Context,
	client *http.Client,
	baseURL, apiKey, model string,
	headers map[string]string,
	req types.GenerateRequest,
	maxRetries int,
) (*types.GenerateResponse, error) {
	// Build messages
	messages := buildOpenAIMessages(req)

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
	// Disable thinking mode for Qwen3-style models.
	// Thinking mode outputs internal reasoning before the actual response,
	// which breaks structured output (JSON plans, skill results) and causes timeouts.
	if strings.Contains(strings.ToLower(model), "qwen") {
		body.ChatTemplateKwargs = map[string]interface{}{"enable_thinking": false}
	}
	if req.MaxTokens > 0 {
		body.MaxTokens = req.MaxTokens
	}
	if req.Temperature > 0 {
		temp := req.Temperature
		body.Temperature = &temp
	}
	body.ToolChoice = mapToolChoiceToOpenAI(req.ToolChoice)
	body.ParallelToolCalls = req.ParallelToolCalls

	payload, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	var lastErr error
	attempts := maxRetries + 1
	if attempts < 1 {
		attempts = 1
	}

	for attempt := 0; attempt < attempts; attempt++ {
		if attempt > 0 {
			backoff := time.Duration(1<<(attempt-1)) * time.Second
			if backoff > 30*time.Second {
				backoff = 30 * time.Second
			}
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff):
			}
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
			lastErr = fmt.Errorf("http request: %w", err)
			continue
		}

		respBody, err := io.ReadAll(resp.Body)
		func() { _ = resp.Body.Close() }()
		if err != nil {
			lastErr = fmt.Errorf("read response: %w", err)
			continue
		}

		if resp.StatusCode == http.StatusOK {
			var chatResp chatResponse
			if err := json.Unmarshal(respBody, &chatResp); err != nil {
				return nil, fmt.Errorf("unmarshal response: %w", err)
			}
			if chatResp.Error != nil {
				return nil, fmt.Errorf("api error: %s (%s)", chatResp.Error.Message, chatResp.Error.Type)
			}

			latency := time.Since(start)
			result := &types.GenerateResponse{
				Usage: types.TokenUsage{
					PromptTokens:     chatResp.Usage.PromptTokens,
					CompletionTokens: chatResp.Usage.CompletionTokens,
					TotalTokens:      chatResp.Usage.TotalTokens,
				},
				Latency: latency,
			}
			if len(chatResp.Choices) > 0 {
				choice := chatResp.Choices[0]
				result.Content = choice.Message.Content
				if result.Content == "" && choice.Message.ReasoningContent != "" {
					result.Content = choice.Message.ReasoningContent
				}
				result.FinishReason = choice.FinishReason
				for _, tc := range choice.Message.ToolCalls {
					result.ToolCalls = append(result.ToolCalls, types.ToolCall{
						ID:        tc.ID,
						Name:      tc.Function.Name,
						Arguments: tc.Function.Arguments,
					})
				}
			}
			return result, nil
		}

		// Non-retryable status codes
		if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
			return nil, fmt.Errorf("%w: status %d", ai.ErrAuthenticationFailed, resp.StatusCode)
		}
		if resp.StatusCode == http.StatusTooManyRequests {
			lastErr = fmt.Errorf("rate limited: status 429")
			continue
		}
		if resp.StatusCode >= 500 {
			lastErr = fmt.Errorf("%w: status %d", ai.ErrProviderUnavailable, resp.StatusCode)
			continue
		}

		// Other 4xx — don't retry
		return nil, fmt.Errorf("%w: status %d: %s", ai.ErrBadRequest, resp.StatusCode, string(respBody))
	}

	return nil, fmt.Errorf("request failed after %d attempts: %w", attempts, lastErr)
}

// openAICompatibleEmbed sends an embedding request to an OpenAI-compatible endpoint.
func openAICompatibleEmbed(
	ctx context.Context,
	client *http.Client,
	baseURL, apiKey, model string,
	headers map[string]string,
	texts []string,
	dimensions int,
) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, fmt.Errorf("embed: no input texts provided")
	}

	body := embedRequest{
		Model: model,
		Input: texts,
	}
	if dimensions > 0 {
		body.Dimensions = dimensions
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal embed request: %w", err)
	}

	endpoint := strings.TrimRight(baseURL, "/") + "/embeddings"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("create embed request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+apiKey)
	for k, v := range headers {
		httpReq.Header.Set(k, v)
	}

	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("embed http request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read embed response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("embed API error: status %d: %s", resp.StatusCode, string(respBody))
	}

	var embedResp embedResponse
	if err := json.Unmarshal(respBody, &embedResp); err != nil {
		return nil, fmt.Errorf("unmarshal embed response: %w", err)
	}
	if embedResp.Error != nil {
		return nil, fmt.Errorf("embed API error: %s (%s)", embedResp.Error.Message, embedResp.Error.Type)
	}

	// Sort by index to ensure order matches input
	results := make([][]float32, len(embedResp.Data))
	for _, d := range embedResp.Data {
		if d.Index < 0 || d.Index >= len(results) {
			continue
		}
		results[d.Index] = d.Embedding
	}

	return results, nil
}

// ----- Provider implementations -----

// OpenAICompatibleProvider implements ModelProvider for any OpenAI-compatible endpoint.
// Parameterized by ID prefix, capabilities, default base URL, and optional embed model.
// Used by OpenAI, ModelScope, SiliconFlow, and any custom OpenAI-compatible endpoint.
type OpenAICompatibleProvider struct {
	baseURL       string
	apiKey        string
	modelName     string
	client        *http.Client
	idPrefix      string
	capabilities  []types.CapabilityType
	embedModel    string // empty means embedding not supported
	errName       string // for error messages (e.g., "openai", "modelscope")
}

func newOpenAICompatible(baseURL, defaultURL, apiKey, modelName, idPrefix string, caps []types.CapabilityType, embedModel string) *OpenAICompatibleProvider {
	if baseURL == "" {
		baseURL = defaultURL
	}
	return &OpenAICompatibleProvider{
		baseURL:      baseURL,
		apiKey:       apiKey,
		modelName:    modelName,
		client:       &http.Client{Timeout: httpTimeoutSeconds * time.Second},
		idPrefix:     idPrefix,
		capabilities: caps,
		embedModel:   embedModel,
		errName:      idPrefix,
	}
}

func NewOpenAIProvider(baseURL, apiKey, modelName string) *OpenAICompatibleProvider {
	return newOpenAICompatible(baseURL, defaultOpenAIBaseURL, apiKey, modelName, "openai",
		[]types.CapabilityType{types.CapTextGeneration, types.CapToolCalling, types.CapJSONMode, types.CapVision, types.CapEmbedding},
		"text-embedding-3-small")
}

func NewModelScopeProvider(baseURL, apiKey, modelName string) *OpenAICompatibleProvider {
	return newOpenAICompatible(baseURL, defaultModelScopeBaseURL, apiKey, modelName, "modelscope",
		[]types.CapabilityType{types.CapTextGeneration, types.CapToolCalling}, "")
}

func NewSiliconFlowProvider(baseURL, apiKey, modelName string) *OpenAICompatibleProvider {
	return newOpenAICompatible(baseURL, defaultSiliconFlowBaseURL, apiKey, modelName, "siliconflow",
		[]types.CapabilityType{types.CapTextGeneration, types.CapToolCalling}, "")
}

func (p *OpenAICompatibleProvider) ID() string { return p.idPrefix + "/" + p.modelName }

func (p *OpenAICompatibleProvider) Capabilities() []types.CapabilityType { return p.capabilities }

func (p *OpenAICompatibleProvider) Generate(ctx context.Context, req types.GenerateRequest) (*types.GenerateResponse, error) {
	if p.apiKey == "" {
		return nil, fmt.Errorf("%s: API key not configured", p.errName)
	}
	return openAICompatibleGenerate(ctx, p.client, p.baseURL, p.apiKey, p.modelName, nil, req, 0)
}

func (p *OpenAICompatibleProvider) GenerateStream(ctx context.Context, req types.GenerateRequest) (<-chan types.StreamChunk, error) {
	if p.apiKey == "" {
		return nil, fmt.Errorf("%s: API key not configured", p.errName)
	}
	return openAICompatibleStream(ctx, p.client, p.baseURL, p.apiKey, p.modelName, nil, req, 0)
}

func (p *OpenAICompatibleProvider) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	if p.embedModel == "" {
		return nil, ai.ErrCapabilityNotSupported
	}
	if p.apiKey == "" {
		return nil, fmt.Errorf("%s: API key not configured", p.errName)
	}
	return openAICompatibleEmbed(ctx, p.client, p.baseURL, p.apiKey, p.embedModel, nil, texts, 0)
}

// ClaudeProvider implements ModelProvider for Anthropic Claude models.
// Uses the native Anthropic Messages API (/v1/messages).
type ClaudeProvider struct {
	baseURL   string
	apiKey    string
	modelName string
	client    *http.Client
}

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

func (p *ClaudeProvider) ID() string { return "anthropic/" + p.modelName }

func (p *ClaudeProvider) Capabilities() []types.CapabilityType {
	return []types.CapabilityType{types.CapTextGeneration, types.CapToolCalling, types.CapVision, types.CapStreaming}
}

func (p *ClaudeProvider) Generate(ctx context.Context, req types.GenerateRequest) (*types.GenerateResponse, error) {
	if p.apiKey == "" {
		return nil, fmt.Errorf("claude: API key not configured")
	}
	return anthropicMessagesGenerate(ctx, p.client, p.baseURL, p.apiKey, p.modelName, req, 0)
}

func (p *ClaudeProvider) GenerateStream(ctx context.Context, req types.GenerateRequest) (<-chan types.StreamChunk, error) {
	if p.apiKey == "" {
		return nil, fmt.Errorf("claude: API key not configured")
	}
	return anthropicMessagesStream(ctx, p.client, p.baseURL, p.apiKey, p.modelName, req, 0)
}

func (p *ClaudeProvider) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	return nil, ai.ErrCapabilityNotSupported
}
