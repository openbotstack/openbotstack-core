package providers

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/openbotstack/openbotstack-core/ai"
	"github.com/openbotstack/openbotstack-core/control/skills"
)

// streamChatRequest extends chatRequest with streaming fields.
type streamChatRequest struct {
	chatRequest
	Stream        bool          `json:"stream"`
	StreamOptions *streamOptions `json:"stream_options,omitempty"`
}

type streamOptions struct {
	IncludeUsage bool `json:"include_usage"`
}

// streamChunk represents a single SSE data payload.
type streamChunk struct {
	Choices []streamChoice `json:"choices"`
	Usage   *chatUsage     `json:"usage,omitempty"`
}

type streamChoice struct {
	Delta        streamDelta `json:"delta"`
	FinishReason *string     `json:"finish_reason"`
}

type streamDelta struct {
	Role      string             `json:"role,omitempty"`
	Content   string             `json:"content,omitempty"`
	ToolCalls []streamToolCallDelta `json:"tool_calls,omitempty"`
}

// streamToolCallDelta represents an incremental tool call update in SSE streaming.
type streamToolCallDelta struct {
	Index    int                   `json:"index"`
	ID       string                `json:"id,omitempty"`
	Type     string                `json:"type,omitempty"`
	Function streamFunctionDelta   `json:"function,omitempty"`
}

type streamFunctionDelta struct {
	Name      string `json:"name,omitempty"`
	Arguments string `json:"arguments,omitempty"`
}

// openAICompatibleStream performs a streaming chat completion request.
// maxRetries controls retries for the initial HTTP POST only (before SSE streaming begins).
// Once the stream starts, there are no retries.
func openAICompatibleStream(
	ctx context.Context,
	client *http.Client,
	baseURL, apiKey, model string,
	headers map[string]string,
	req skills.GenerateRequest,
	maxRetries int,
) (<-chan skills.StreamChunk, error) {
	// Build request body
	messages := make([]chatMessage, 0, len(req.Messages))
	for _, m := range req.Messages {
		messages = append(messages, chatMessage{Role: m.Role, Content: m.Content, Name: m.Name})
	}
	var tools []chatTool
	for _, t := range req.Tools {
		tools = append(tools, chatTool{
			Type:     "function",
			Function: chatFunction{Name: t.Name, Description: t.Description, Parameters: t.Parameters},
		})
	}

	body := streamChatRequest{
		chatRequest: chatRequest{
			Model:    model,
			Messages: messages,
			Tools:    tools,
		},
		Stream:        true,
		StreamOptions: &streamOptions{IncludeUsage: true},
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
		return nil, fmt.Errorf("marshal stream request: %w", err)
	}

	// Execute HTTP request (with retry for initial connection)
	var resp *http.Response
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
		httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(string(payload)))
		if err != nil {
			return nil, fmt.Errorf("create stream request: %w", err)
		}
		httpReq.Header.Set("Content-Type", "application/json")
		if apiKey != "" {
			httpReq.Header.Set("Authorization", "Bearer "+apiKey)
		}
		for k, v := range headers {
			httpReq.Header.Set(k, v)
		}

		resp, lastErr = client.Do(httpReq)
		if lastErr != nil {
			continue
		}
		if resp.StatusCode == http.StatusOK {
			break
		}

		if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
			func() { _ = resp.Body.Close() }()
			return nil, fmt.Errorf("%w: status %d", ai.ErrAuthenticationFailed, resp.StatusCode)
		}
		if resp.StatusCode < 500 && resp.StatusCode != http.StatusTooManyRequests {
			func() { _ = resp.Body.Close() }()
			return nil, fmt.Errorf("%w: status %d", ai.ErrBadRequest, resp.StatusCode)
		}

		func() { _ = resp.Body.Close() }()
		lastErr = fmt.Errorf("stream request failed with status %d", resp.StatusCode)
	}

	if resp == nil {
		return nil, fmt.Errorf("%w: %v", ai.ErrProviderUnavailable, lastErr)
	}

	// Start goroutine to read SSE stream
	ch := make(chan skills.StreamChunk, 64)
	go func() {
		defer func() { _ = resp.Body.Close() }()
		defer close(ch)

		// Tool call accumulator: index → accumulated state
		toolCallAccum := make(map[int]*skills.ToolCall)
		toolCallCount := 0

		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			line := scanner.Text()

			if line == "" {
				continue
			}

			if !strings.HasPrefix(line, "data: ") {
				continue
			}

			data := strings.TrimPrefix(line, "data: ")

			if data == "[DONE]" {
				return
			}

			var chunk streamChunk
			if err := json.Unmarshal([]byte(data), &chunk); err != nil {
				slog.Warn("streaming: malformed SSE data, skipping", "error", err, "data", data)
				continue
			}

			sc := skills.StreamChunk{}
			if len(chunk.Choices) > 0 {
				sc.Content = chunk.Choices[0].Delta.Content
				if chunk.Choices[0].FinishReason != nil {
					sc.FinishReason = *chunk.Choices[0].FinishReason
				}
				if len(chunk.Choices[0].Delta.ToolCalls) > 0 {
					for _, tc := range chunk.Choices[0].Delta.ToolCalls {
						idx := tc.Index
						if existing, ok := toolCallAccum[idx]; ok {
							if tc.ID != "" {
								existing.ID = tc.ID
							}
							if tc.Function.Name != "" {
								existing.Name = tc.Function.Name
							}
							existing.Arguments += tc.Function.Arguments
						} else {
							toolCallAccum[idx] = &skills.ToolCall{
								ID:        tc.ID,
								Name:      tc.Function.Name,
								Arguments: tc.Function.Arguments,
							}
							if idx >= toolCallCount {
								toolCallCount = idx + 1
							}
						}
					}
				}
				// Emit accumulated tool call state on every chunk
				if toolCallCount > 0 {
					sc.ToolCalls = make([]skills.ToolCall, 0, toolCallCount)
					for i := 0; i < toolCallCount; i++ {
						if tc, ok := toolCallAccum[i]; ok {
							sc.ToolCalls = append(sc.ToolCalls, *tc)
						}
					}
				}
			}
			if chunk.Usage != nil {
				sc.Usage = skills.TokenUsage{
					PromptTokens:     chunk.Usage.PromptTokens,
					CompletionTokens: chunk.Usage.CompletionTokens,
					TotalTokens:      chunk.Usage.TotalTokens,
				}
			}

			select {
			case ch <- sc:
			case <-ctx.Done():
				// Non-blocking send: consumer may have already stopped reading.
				select {
				case ch <- skills.StreamChunk{Error: ctx.Err()}:
				default:
				}
				return
			}
		}

		if err := scanner.Err(); err != nil && err != io.EOF {
			select {
			case ch <- skills.StreamChunk{Error: fmt.Errorf("stream read error: %w", err)}:
			default:
			}
		}
	}()

	return ch, nil
}
