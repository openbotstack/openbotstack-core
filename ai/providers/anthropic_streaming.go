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

// ----- Anthropic SSE event types -----

// anthropicSSEEvent represents a parsed SSE event from the Anthropic API.
type anthropicSSEEvent struct {
	Type  string          `json:"type"`
	Index int             `json:"index,omitempty"`
	Delta anthropicDelta  `json:"delta,omitempty"`

	// For message_start
	Message *anthropicMessagesResponse `json:"message,omitempty"`

	// For content_block_start
	ContentBlock *anthropicContentBlock `json:"content_block,omitempty"`

	// For message_delta
	StopReason string          `json:"stop_reason,omitempty"`
	Usage      *anthropicUsage `json:"usage,omitempty"`
}

// anthropicDelta represents a content delta in an SSE event.
type anthropicDelta struct {
	Type        string `json:"type,omitempty"`
	Text        string `json:"text,omitempty"`
	PartialJSON string `json:"partial_json,omitempty"`
	StopReason  string `json:"stop_reason,omitempty"`
}

// anthropicStreamToolAccum tracks tool call state during streaming.
type anthropicStreamToolAccum struct {
	ID   string
	Name string
	JSON strings.Builder
}

// anthropicMessagesStream performs a streaming request to the Anthropic Messages API.
func anthropicMessagesStream(
	ctx context.Context,
	client *http.Client,
	baseURL, apiKey, model string,
	req skills.GenerateRequest,
	maxRetries int,
) (<-chan skills.StreamChunk, error) {
	// Build request body (same as generate, but with stream=true)
	body := anthropicMessagesRequest{
		Model:  model,
		Stream: true,
	}

	var systemParts []string
	var convMessages []anthropicMessage
	for _, m := range req.Messages {
		if m.Role == "system" {
			systemParts = append(systemParts, m.Content)
		} else {
			convMessages = append(convMessages, anthropicMessage{
				Role:    m.Role,
				Content: m.Content,
			})
		}
	}
	body.System = strings.Join(systemParts, "\n")
	body.Messages = convMessages

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
		body.MaxTokens = 4096
	}
	if req.Temperature > 0 {
		temp := req.Temperature
		body.Temperature = &temp
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("anthropic stream: marshal request: %w", err)
	}

	// Execute HTTP request with retry
	var resp *http.Response
	var lastErr error
	var gotOK bool
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
		httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(string(payload)))
		if err != nil {
			return nil, fmt.Errorf("anthropic stream: create request: %w", err)
		}
		httpReq.Header.Set("Content-Type", "application/json")
		httpReq.Header.Set("x-api-key", apiKey)
		httpReq.Header.Set("anthropic-version", claudeAPIVersion)

		resp, lastErr = client.Do(httpReq)
		if lastErr != nil {
			continue
		}
		if resp.StatusCode == http.StatusOK {
			gotOK = true
			break
		}

		if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
			func() { _ = resp.Body.Close() }()
			return nil, fmt.Errorf("%w: anthropic stream status %d", ai.ErrAuthenticationFailed, resp.StatusCode)
		}
		if resp.StatusCode < 500 && resp.StatusCode != http.StatusTooManyRequests {
			func() { _ = resp.Body.Close() }()
			return nil, fmt.Errorf("%w: anthropic stream status %d", ai.ErrBadRequest, resp.StatusCode)
		}

		func() { _ = resp.Body.Close() }()
		lastErr = fmt.Errorf("anthropic stream: request failed with status %d", resp.StatusCode)
	}

	if !gotOK {
		return nil, fmt.Errorf("%w: %v", ai.ErrProviderUnavailable, lastErr)
	}

	// Start goroutine to read Anthropic SSE stream
	ch := make(chan skills.StreamChunk, 64)
	go func() {
		defer func() { _ = resp.Body.Close() }()
		defer close(ch)

		// Tool call accumulator: index → accumulated state
		toolAccum := make(map[int]*anthropicStreamToolAccum)
		toolOrder := make([]int, 0)

		scanner := bufio.NewScanner(resp.Body)
		var currentEvent string

		for scanner.Scan() {
			line := scanner.Text()

			if line == "" {
				currentEvent = ""
				continue
			}

			// Parse event type line
			if strings.HasPrefix(line, "event: ") {
				currentEvent = strings.TrimPrefix(line, "event: ")
				continue
			}

			// Parse data line
			if !strings.HasPrefix(line, "data: ") {
				continue
			}

			data := strings.TrimPrefix(line, "data: ")

			// Skip non-meaningful events
			switch currentEvent {
			case "message_start":
				// Parse to get input tokens, but don't emit a chunk
				continue
			case "ping":
				continue
			case "message_stop":
				return
			case "content_block_start":
				var evt struct {
					Index        int                    `json:"index"`
					ContentBlock *anthropicContentBlock `json:"content_block"`
				}
				if err := json.Unmarshal([]byte(data), &evt); err != nil {
					slog.Warn("anthropic stream: malformed content_block_start", "error", err)
					continue
				}
				if evt.ContentBlock != nil && evt.ContentBlock.Type == "tool_use" {
					toolAccum[evt.Index] = &anthropicStreamToolAccum{
						ID:   evt.ContentBlock.ID,
						Name: evt.ContentBlock.Name,
					}
					toolOrder = append(toolOrder, evt.Index)
				}
				continue
			case "content_block_stop":
				// Emit accumulated tool calls if we have any
				if len(toolAccum) > 0 {
					toolCalls := buildToolCalls(toolAccum, toolOrder)
					if len(toolCalls) > 0 {
						select {
						case ch <- skills.StreamChunk{ToolCalls: toolCalls}:
						case <-ctx.Done():
							sendErrorChunk(ch, ctx.Err())
							return
						}
					}
				}
				continue
			case "content_block_delta":
				var evt anthropicSSEEvent
				if err := json.Unmarshal([]byte(data), &evt); err != nil {
					slog.Warn("anthropic stream: malformed content_block_delta", "error", err)
					continue
				}

				sc := skills.StreamChunk{}
				if evt.Delta.Type == "text_delta" {
					sc.Content = evt.Delta.Text
				} else if evt.Delta.Type == "input_json_delta" {
					// Accumulate tool input JSON
					if accum, ok := toolAccum[evt.Index]; ok {
						accum.JSON.WriteString(evt.Delta.PartialJSON)
					}
					continue // Don't emit chunk for partial JSON
				}

				// Include current tool call state
				if len(toolAccum) > 0 {
					sc.ToolCalls = buildToolCalls(toolAccum, toolOrder)
				}

				select {
				case ch <- sc:
				case <-ctx.Done():
					sendErrorChunk(ch, ctx.Err())
					return
				}
				continue
			case "message_delta":
				var evt struct {
					Delta struct {
						StopReason string `json:"stop_reason"`
					} `json:"delta"`
					Usage *anthropicUsage `json:"usage"`
				}
				if err := json.Unmarshal([]byte(data), &evt); err != nil {
					slog.Warn("anthropic stream: malformed message_delta", "error", err)
					continue
				}

				sc := skills.StreamChunk{
					FinishReason: evt.Delta.StopReason,
				}
				if evt.Usage != nil {
					sc.Usage = skills.TokenUsage{
						CompletionTokens: evt.Usage.OutputTokens,
					}
				}
				// Final tool call state
				if len(toolAccum) > 0 {
					sc.ToolCalls = buildToolCalls(toolAccum, toolOrder)
				}

				select {
				case ch <- sc:
				case <-ctx.Done():
					sendErrorChunk(ch, ctx.Err())
					return
				}
				continue
			}
		}

		if err := scanner.Err(); err != nil && err != io.EOF {
			select {
			case ch <- skills.StreamChunk{Error: fmt.Errorf("anthropic stream read error: %w", err)}:
			default:
			}
		}
	}()

	return ch, nil
}

// buildToolCalls constructs the current tool call state from accumulators.
func buildToolCalls(accum map[int]*anthropicStreamToolAccum, order []int) []skills.ToolCall {
	if len(accum) == 0 {
		return nil
	}
	result := make([]skills.ToolCall, 0, len(accum))
	for _, idx := range order {
		if tc, ok := accum[idx]; ok {
			result = append(result, skills.ToolCall{
				ID:        tc.ID,
				Name:      tc.Name,
				Arguments: tc.JSON.String(),
			})
		}
	}
	return result
}

// sendErrorChunk sends an error chunk to the channel (non-blocking).
func sendErrorChunk(ch chan skills.StreamChunk, err error) {
	select {
	case ch <- skills.StreamChunk{Error: err}:
	default:
	}
}
