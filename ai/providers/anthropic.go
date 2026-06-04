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

// ----- Anthropic Messages API request/response types -----

// anthropicMessagesRequest is the request body for POST /v1/messages.
type anthropicMessagesRequest struct {
	Model                string                    `json:"model"`
	MaxTokens            int                       `json:"max_tokens"`
	System               string                    `json:"system,omitempty"`
	Messages             []anthropicMessage        `json:"messages"`
	Tools                []anthropicToolDefinition `json:"tools,omitempty"`
	ToolChoice           any                       `json:"tool_choice,omitempty"`
	DisableParallelUse   *bool                     `json:"disable_parallel_tool_use,omitempty"`
	Stream               bool                      `json:"stream,omitempty"`

	Temperature *float64 `json:"temperature,omitempty"`
}

// anthropicMessage represents a single message in the Anthropic format.
type anthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// anthropicToolDefinition represents a tool in the Anthropic format.
type anthropicToolDefinition struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	InputSchema *types.JSONSchema `json:"input_schema,omitempty"`
}

// anthropicMessagesResponse is the response from POST /v1/messages.
type anthropicMessagesResponse struct {
	ID         string                   `json:"id"`
	Type       string                   `json:"type"`
	Role       string                   `json:"role"`
	Content    []anthropicContentBlock  `json:"content"`
	Model      string                   `json:"model"`
	StopReason string                   `json:"stop_reason"`
	Usage      anthropicUsage           `json:"usage"`
	Error      *anthropicErrorResponse `json:"error,omitempty"`
}

// anthropicContentBlock is a content block in the Anthropic response.
type anthropicContentBlock struct {
	Type  string          `json:"type"`
	Text  string          `json:"text,omitempty"`
	ID    string          `json:"id,omitempty"`
	Name  string          `json:"name,omitempty"`
	Input json.RawMessage `json:"input,omitempty"`
}

// anthropicUsage tracks token usage in the Anthropic format.
type anthropicUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// anthropicErrorResponse represents an error from the Anthropic API.
type anthropicErrorResponse struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

// anthropicMessagesGenerate sends a request to the Anthropic Messages API.
func anthropicMessagesGenerate(
	ctx context.Context,
	client *http.Client,
	baseURL, apiKey, model string,
	req types.GenerateRequest,
	maxRetries int,
) (*types.GenerateResponse, error) {
	// Build Anthropic-format request
	body := anthropicMessagesRequest{
		Model: model,
	}

	// Extract system messages and separate from conversation messages
	var systemParts []string
	var convMessages []anthropicMessage
	for _, m := range req.Messages {
		if m.Role == "system" {
			systemParts = append(systemParts, types.FlattenToText(m.Contents))
		} else {
			convMessages = append(convMessages, anthropicMessage{
				Role:    m.Role,
				Content: types.FlattenToText(m.Contents),
			})
		}
	}
	body.System = strings.Join(systemParts, "\n")
	body.Messages = convMessages

	// Convert tools
	for _, t := range req.Tools {
		body.Tools = append(body.Tools, anthropicToolDefinition{
			Name:        t.Name,
			Description: t.Description,
			InputSchema: t.Parameters,
		})
	}

	if req.MaxTokens > 0 {
		body.MaxTokens = req.MaxTokens
	} else {
		body.MaxTokens = 4096 // Anthropic requires max_tokens
	}
	if req.Temperature > 0 {
		temp := req.Temperature
		body.Temperature = &temp
	}
	body.ToolChoice = mapToolChoiceToAnthropic(req.ToolChoice)
	// Anthropic uses inverted logic: disable_parallel_tool_use=true means no parallel
	if req.ParallelToolCalls != nil && !*req.ParallelToolCalls {
		disabled := true
		body.DisableParallelUse = &disabled
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("anthropic: marshal request: %w", err)
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

		endpoint := strings.TrimRight(baseURL, "/") + "/messages"
		httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
		if err != nil {
			return nil, fmt.Errorf("anthropic: create request: %w", err)
		}

		httpReq.Header.Set("Content-Type", "application/json")
		httpReq.Header.Set("x-api-key", apiKey)
		httpReq.Header.Set("anthropic-version", claudeAPIVersion)

		start := time.Now()
		resp, err := client.Do(httpReq)
		if err != nil {
			lastErr = fmt.Errorf("anthropic: http request: %w", err)
			continue
		}

		respBody, err := io.ReadAll(resp.Body)
		func() { _ = resp.Body.Close() }()
		if err != nil {
			lastErr = fmt.Errorf("anthropic: read response: %w", err)
			continue
		}

		if resp.StatusCode == http.StatusOK {
			var apiResp anthropicMessagesResponse
			if err := json.Unmarshal(respBody, &apiResp); err != nil {
				return nil, fmt.Errorf("anthropic: unmarshal response: %w", err)
			}
			if apiResp.Error != nil {
				return nil, fmt.Errorf("anthropic: api error: %s (%s)", apiResp.Error.Message, apiResp.Error.Type)
			}

			latency := time.Since(start)
			result := &types.GenerateResponse{
				Usage: types.TokenUsage{
					PromptTokens:     apiResp.Usage.InputTokens,
					CompletionTokens: apiResp.Usage.OutputTokens,
					TotalTokens:      apiResp.Usage.InputTokens + apiResp.Usage.OutputTokens,
				},
				Latency:      latency,
				FinishReason: apiResp.StopReason,
			}

			// Extract content and tool calls from content blocks
			for _, block := range apiResp.Content {
				switch block.Type {
				case "text":
					result.Content += block.Text
				case "tool_use":
					result.ToolCalls = append(result.ToolCalls, types.ToolCall{
						ID:        block.ID,
						Name:      block.Name,
						Arguments: string(block.Input),
					})
				}
			}

			return result, nil
		}

		// Non-retryable status codes
		if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
			return nil, fmt.Errorf("%w: anthropic status %d", ai.ErrAuthenticationFailed, resp.StatusCode)
		}
		if resp.StatusCode == http.StatusTooManyRequests {
			lastErr = fmt.Errorf("anthropic: rate limited: status 429")
			continue
		}
		if resp.StatusCode >= 500 {
			lastErr = fmt.Errorf("%w: anthropic status %d", ai.ErrProviderUnavailable, resp.StatusCode)
			continue
		}

		return nil, fmt.Errorf("%w: anthropic status %d: %s", ai.ErrBadRequest, resp.StatusCode, string(respBody))
	}

	return nil, fmt.Errorf("anthropic: request failed after %d attempts: %w", attempts, lastErr)
}
