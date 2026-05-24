package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/openbotstack/openbotstack-core/control/skills"
)

// --- Anthropic Messages API Generate tests ---

func TestAnthropicMessagesGenerate(t *testing.T) {
	mockResp := anthropicMessagesResponse{
		ID:   "msg_test123",
		Type: "message",
		Role: "assistant",
		Content: []anthropicContentBlock{
			{Type: "text", Text: "Hello from Claude!"},
		},
		Model:      "claude-3-opus-20240229",
		StopReason: "end_turn",
		Usage: anthropicUsage{
			InputTokens:  10,
			OutputTokens: 8,
		},
	}

	var receivedReq anthropicMessagesRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("x-api-key") != "test-api-key" {
			t.Errorf("Expected x-api-key 'test-api-key', got '%s'", r.Header.Get("x-api-key"))
		}
		if r.Header.Get("anthropic-version") != "2023-06-01" {
			t.Errorf("Expected anthropic-version '2023-06-01', got '%s'", r.Header.Get("anthropic-version"))
		}
		if r.Header.Get("Authorization") != "" {
			t.Errorf("Anthropic provider should not use Bearer auth, got '%s'", r.Header.Get("Authorization"))
		}
		if r.URL.Path != "/messages" {
			t.Errorf("Expected /messages endpoint, got %s", r.URL.Path)
		}

		if err := json.NewDecoder(r.Body).Decode(&receivedReq); err != nil {
			t.Fatalf("Failed to decode request: %v", err)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockResp)
	}))
	defer server.Close()

	provider := NewClaudeProvider(server.URL, "test-api-key", "claude-3-opus-20240229")
	resp, err := provider.Generate(context.Background(), skills.GenerateRequest{
		Messages: []skills.Message{
			{Role: "system", Content: "You are helpful."},
			{Role: "user", Content: "Hello"},
		},
		MaxTokens:   100,
		Temperature: 0.7,
	})
	if err != nil {
		t.Fatalf("Generate returned unexpected error: %v", err)
	}

	if resp.Content != "Hello from Claude!" {
		t.Errorf("Expected 'Hello from Claude!', got '%s'", resp.Content)
	}
	if resp.FinishReason != "end_turn" {
		t.Errorf("Expected finish_reason 'end_turn', got '%s'", resp.FinishReason)
	}
	if resp.Usage.PromptTokens != 10 {
		t.Errorf("Expected 10 prompt tokens, got %d", resp.Usage.PromptTokens)
	}
	if resp.Usage.CompletionTokens != 8 {
		t.Errorf("Expected 8 completion tokens, got %d", resp.Usage.CompletionTokens)
	}

	// Verify request format
	if receivedReq.Model != "claude-3-opus-20240229" {
		t.Errorf("Expected model 'claude-3-opus-20240229', got '%s'", receivedReq.Model)
	}
	if receivedReq.MaxTokens != 100 {
		t.Errorf("Expected max_tokens 100, got %d", receivedReq.MaxTokens)
	}
	if receivedReq.System != "You are helpful." {
		t.Errorf("Expected system 'You are helpful.', got '%s'", receivedReq.System)
	}
	if len(receivedReq.Messages) != 1 {
		t.Fatalf("Expected 1 message (user only), got %d", len(receivedReq.Messages))
	}
	if receivedReq.Messages[0].Role != "user" {
		t.Errorf("Expected first message role 'user', got '%s'", receivedReq.Messages[0].Role)
	}
}

func TestAnthropicMessagesGenerateWithToolCalls(t *testing.T) {
	mockResp := anthropicMessagesResponse{
		ID:   "msg_tool_test",
		Type: "message",
		Role: "assistant",
		Content: []anthropicContentBlock{
			{Type: "text", Text: "Let me check the weather."},
			{
				Type:  "tool_use",
				ID:    "toolu_abc123",
				Name:  "get_weather",
				Input: json.RawMessage(`{"location":"San Francisco"}`),
			},
		},
		Model:      "claude-3-opus-20240229",
		StopReason: "tool_use",
		Usage:      anthropicUsage{InputTokens: 20, OutputTokens: 30},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockResp)
	}))
	defer server.Close()

	provider := NewClaudeProvider(server.URL, "test-key", "claude-3-opus-20240229")
	resp, err := provider.Generate(context.Background(), skills.GenerateRequest{
		Messages: []skills.Message{{Role: "user", Content: "What's the weather?"}},
		Tools: []skills.ToolDefinition{
			{Name: "get_weather", Description: "Get weather for a location"},
		},
	})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if resp.Content != "Let me check the weather." {
		t.Errorf("Expected text content, got '%s'", resp.Content)
	}
	if len(resp.ToolCalls) != 1 {
		t.Fatalf("Expected 1 tool call, got %d", len(resp.ToolCalls))
	}
	if resp.ToolCalls[0].ID != "toolu_abc123" {
		t.Errorf("Expected tool call ID 'toolu_abc123', got '%s'", resp.ToolCalls[0].ID)
	}
	if resp.ToolCalls[0].Name != "get_weather" {
		t.Errorf("Expected tool name 'get_weather', got '%s'", resp.ToolCalls[0].Name)
	}
	if resp.ToolCalls[0].Arguments != `{"location":"San Francisco"}` {
		t.Errorf("Expected tool arguments, got '%s'", resp.ToolCalls[0].Arguments)
	}
	if resp.FinishReason != "tool_use" {
		t.Errorf("Expected finish_reason 'tool_use', got '%s'", resp.FinishReason)
	}
}

func TestAnthropicMessagesGenerateNoAPIKey(t *testing.T) {
	provider := NewClaudeProvider("", "", "claude-3-opus-20240229")
	_, err := provider.Generate(context.Background(), skills.GenerateRequest{
		Messages: []skills.Message{{Role: "user", Content: "Hello"}},
	})
	if err == nil {
		t.Error("Expected error for empty API key, got nil")
	}
}

func TestAnthropicMessagesGenerateAuthError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{
				"type":    "authentication_error",
				"message": "invalid x-api-key",
			},
		})
	}))
	defer server.Close()

	provider := NewClaudeProvider(server.URL, "bad-key", "claude-3-opus-20240229")
	_, err := provider.Generate(context.Background(), skills.GenerateRequest{
		Messages: []skills.Message{{Role: "user", Content: "Hello"}},
	})
	if err == nil {
		t.Error("Expected error for 401 response")
	}
}

func TestAnthropicMessagesGenerateRateLimit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer server.Close()

	provider := NewClaudeProvider(server.URL, "key", "claude-3-opus-20240229")
	_, err := provider.Generate(context.Background(), skills.GenerateRequest{
		Messages: []skills.Message{{Role: "user", Content: "Hello"}},
	})
	if err == nil {
		t.Error("Expected error for rate limit")
	}
}

// --- Anthropic Streaming Tests ---

func TestAnthropicStreamingText(t *testing.T) {
	sseData := "event: message_start\n"+
		"data: {\"type\":\"message_start\",\"message\":{\"id\":\"msg_1\",\"type\":\"message\",\"role\":\"assistant\",\"content\":[],\"model\":\"claude-3\",\"usage\":{\"input_tokens\":10}}}\n\n"+
		"event: content_block_start\n"+
		"data: {\"type\":\"content_block_start\",\"index\":0,\"content_block\":{\"type\":\"text\",\"text\":\"\"}}\n\n"+
		"event: content_block_delta\n"+
		"data: {\"type\":\"content_block_delta\",\"index\":0,\"delta\":{\"type\":\"text_delta\",\"text\":\"Hello\"}}\n\n"+
		"event: content_block_delta\n"+
		"data: {\"type\":\"content_block_delta\",\"index\":0,\"delta\":{\"type\":\"text_delta\",\"text\":\" world\"}}\n\n"+
		"event: content_block_stop\n"+
		"data: {\"type\":\"content_block_stop\",\"index\":0}\n\n"+
		"event: message_delta\n"+
		"data: {\"type\":\"message_delta\",\"delta\":{\"stop_reason\":\"end_turn\"},\"usage\":{\"output_tokens\":8}}\n\n"+
		"event: message_stop\n"+
		"data: {\"type\":\"message_stop\"}\n\n"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("x-api-key") != "test-key" {
			t.Errorf("Expected x-api-key header")
		}
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, sseData)
	}))
	defer server.Close()

	provider := NewClaudeProvider(server.URL, "test-key", "claude-3-opus")
	var sp StreamingModelProvider = provider

	ch, err := sp.GenerateStream(context.Background(), skills.GenerateRequest{
		Messages: []skills.Message{{Role: "user", Content: "Hello"}},
	})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	var textContent string
	var gotFinishReason string
	var gotUsage skills.TokenUsage
	for chunk := range ch {
		if chunk.Error != nil {
			t.Fatalf("Stream error: %v", chunk.Error)
		}
		textContent += chunk.Content
		if chunk.FinishReason != "" {
			gotFinishReason = chunk.FinishReason
		}
		if chunk.Usage.CompletionTokens > 0 {
			gotUsage = chunk.Usage
		}
	}

	if textContent != "Hello world" {
		t.Errorf("Expected 'Hello world', got '%s'", textContent)
	}
	if gotFinishReason != "end_turn" {
		t.Errorf("Expected finish_reason 'end_turn', got '%s'", gotFinishReason)
	}
	if gotUsage.CompletionTokens != 8 {
		t.Errorf("Expected 8 completion tokens, got %d", gotUsage.CompletionTokens)
	}
}

func TestAnthropicStreamingToolCalls(t *testing.T) {
	sseData := "event: message_start\n"+
		"data: {\"type\":\"message_start\",\"message\":{\"id\":\"msg_1\",\"type\":\"message\",\"role\":\"assistant\",\"content\":[],\"model\":\"claude-3\",\"usage\":{\"input_tokens\":15}}}\n\n"+
		"event: content_block_start\n"+
		"data: {\"type\":\"content_block_start\",\"index\":0,\"content_block\":{\"type\":\"text\",\"text\":\"\"}}\n\n"+
		"event: content_block_delta\n"+
		"data: {\"type\":\"content_block_delta\",\"index\":0,\"delta\":{\"type\":\"text_delta\",\"text\":\"Checking.\"}}\n\n"+
		"event: content_block_stop\n"+
		"data: {\"type\":\"content_block_stop\",\"index\":0}\n\n"+
		"event: content_block_start\n"+
		"data: {\"type\":\"content_block_start\",\"index\":1,\"content_block\":{\"type\":\"tool_use\",\"id\":\"toolu_1\",\"name\":\"get_weather\"}}\n\n"+
		"event: content_block_delta\n"+
		"data: {\"type\":\"content_block_delta\",\"index\":1,\"delta\":{\"type\":\"input_json_delta\",\"partial_json\":\"{\\\"loc\"}}\n\n"+
		"event: content_block_delta\n"+
		"data: {\"type\":\"content_block_delta\",\"index\":1,\"delta\":{\"type\":\"input_json_delta\",\"partial_json\":\"ation\\\":\\\"SF\\\"}\"}}\n\n"+
		"event: content_block_stop\n"+
		"data: {\"type\":\"content_block_stop\",\"index\":1}\n\n"+
		"event: message_delta\n"+
		"data: {\"type\":\"message_delta\",\"delta\":{\"stop_reason\":\"tool_use\"},\"usage\":{\"output_tokens\":20}}\n\n"+
		"event: message_stop\n"+
		"data: {\"type\":\"message_stop\"}\n\n"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, sseData)
	}))
	defer server.Close()

	provider := NewClaudeProvider(server.URL, "key", "claude-3")
	var sp StreamingModelProvider = provider

	ch, err := sp.GenerateStream(context.Background(), skills.GenerateRequest{
		Messages: []skills.Message{{Role: "user", Content: "Weather?"}},
	})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	var lastChunk skills.StreamChunk
	var textContent string
	for chunk := range ch {
		if chunk.Error != nil {
			t.Fatalf("Stream error: %v", chunk.Error)
		}
		textContent += chunk.Content
		lastChunk = chunk
	}

	if textContent != "Checking." {
		t.Errorf("Expected text 'Checking.', got '%s'", textContent)
	}
	if lastChunk.FinishReason != "tool_use" {
		t.Errorf("Expected finish_reason 'tool_use', got '%s'", lastChunk.FinishReason)
	}
	if len(lastChunk.ToolCalls) != 1 {
		t.Fatalf("Expected 1 tool call, got %d", len(lastChunk.ToolCalls))
	}
	tc := lastChunk.ToolCalls[0]
	if tc.ID != "toolu_1" {
		t.Errorf("Expected tool ID 'toolu_1', got '%s'", tc.ID)
	}
	if tc.Name != "get_weather" {
		t.Errorf("Expected tool name 'get_weather', got '%s'", tc.Name)
	}
	if tc.Arguments != `{"location":"SF"}` {
		t.Errorf("Expected accumulated arguments, got '%s'", tc.Arguments)
	}
}

func TestAnthropicStreamingContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "event: message_start\ndata: {\"type\":\"message_start\",\"message\":{\"id\":\"msg_1\",\"type\":\"message\",\"role\":\"assistant\",\"content\":[],\"model\":\"claude-3\",\"usage\":{\"input_tokens\":10}}}\n\n")
		w.(http.Flusher).Flush()
		// Keep writing content until client disconnects
		for {
			select {
			case <-r.Context().Done():
				return
			default:
				fmt.Fprint(w, "event: content_block_delta\ndata: {\"type\":\"content_block_delta\",\"index\":0,\"delta\":{\"type\":\"text_delta\",\"text\":\"x\"}}\n\n")
				w.(http.Flusher).Flush()
			}
		}
	}))
	defer server.Close()

	provider := NewClaudeProvider(server.URL, "key", "claude-3")
	var sp StreamingModelProvider = provider

	ch, _ := sp.GenerateStream(ctx, skills.GenerateRequest{
		Messages: []skills.Message{{Role: "user", Content: "Hi"}},
	})

	// Read one chunk to confirm streaming is working
	<-ch
	// Cancel the context — goroutine should detect this
	cancel()

	// Drain remaining chunks with timeout
	done := make(chan struct{})
	go func() {
		for range ch {
		}
		close(done)
	}()
	select {
	case <-done:
		// Channel closed after cancellation
	case <-time.After(5 * time.Second):
		t.Fatal("stream channel did not close after context cancellation")
	}
	}

func TestAnthropicStreamingServerErr(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, `{"error":{"type":"server_error","message":"internal error"}}`)
	}))
	defer server.Close()

	provider := NewClaudeProvider(server.URL, "key", "claude-3")
	var sp StreamingModelProvider = provider

	_, err := sp.GenerateStream(context.Background(), skills.GenerateRequest{
		Messages: []skills.Message{{Role: "user", Content: "Hi"}},
	})
	if err == nil {
		t.Error("Expected error for 500 response")
	}
}

// --- Request format tests ---

func TestAnthropicSystemMessageExtraction(t *testing.T) {
	var receivedReq anthropicMessagesRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&receivedReq)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(anthropicMessagesResponse{
			ID:   "msg_test",
			Type: "message",
			Role: "assistant",
			Content: []anthropicContentBlock{
				{Type: "text", Text: "ok"},
			},
			StopReason: "end_turn",
		})
	}))
	defer server.Close()

	provider := NewClaudeProvider(server.URL, "key", "claude-3")
	_, err := provider.Generate(context.Background(), skills.GenerateRequest{
		Messages: []skills.Message{
			{Role: "system", Content: "System prompt A"},
			{Role: "system", Content: "System prompt B"},
			{Role: "user", Content: "Hello"},
		},
	})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if receivedReq.System != "System prompt A\nSystem prompt B" {
		t.Errorf("Expected concatenated system prompts, got '%s'", receivedReq.System)
	}
	if len(receivedReq.Messages) != 1 {
		t.Fatalf("Expected 1 message, got %d", len(receivedReq.Messages))
	}
}

func TestAnthropicToolDefinitionFormat(t *testing.T) {
	var receivedReq anthropicMessagesRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&receivedReq)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(anthropicMessagesResponse{
			ID:   "msg_test",
			Type: "message",
			Role: "assistant",
			Content: []anthropicContentBlock{
				{Type: "text", Text: "ok"},
			},
			StopReason: "end_turn",
		})
	}))
	defer server.Close()

	provider := NewClaudeProvider(server.URL, "key", "claude-3")
	_, err := provider.Generate(context.Background(), skills.GenerateRequest{
		Messages: []skills.Message{{Role: "user", Content: "Test"}},
		Tools: []skills.ToolDefinition{
			{
				Name:        "search",
				Description: "Search the web",
				Parameters:  &skills.JSONSchema{Type: "object"},
			},
		},
	})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(receivedReq.Tools) != 1 {
		t.Fatalf("Expected 1 tool, got %d", len(receivedReq.Tools))
	}
	if receivedReq.Tools[0].Name != "search" {
		t.Errorf("Expected tool name 'search', got '%s'", receivedReq.Tools[0].Name)
	}
	if receivedReq.Tools[0].Description != "Search the web" {
		t.Errorf("Expected tool description, got '%s'", receivedReq.Tools[0].Description)
	}
}

// --- Interface compliance tests ---

func TestAnthropicStreamingInterface(t *testing.T) {
	provider := NewClaudeProvider("", "key", "claude-3")
	var _ StreamingModelProvider = provider
}

func TestAnthropicCapabilitiesIncludeStreaming(t *testing.T) {
	provider := NewClaudeProvider("", "key", "claude-3")
	caps := provider.Capabilities()

	hasStreaming := false
	for _, c := range caps {
		if c == skills.CapStreaming {
			hasStreaming = true
		}
	}
	if !hasStreaming {
		t.Error("ClaudeProvider should declare CapStreaming capability")
	}
}

// --- Anthropic streaming ToolChoice/ParallelToolCalls propagation (G5) ---

func TestAnthropicStreamingToolChoicePropagation(t *testing.T) {
	sseData := "event: message_start\n"+
		"data: {\"type\":\"message_start\",\"message\":{\"id\":\"msg_1\",\"type\":\"message\",\"role\":\"assistant\",\"content\":[],\"model\":\"claude-3\",\"usage\":{\"input_tokens\":5}}}\n\n"+
		"event: content_block_start\n"+
		"data: {\"type\":\"content_block_start\",\"index\":0,\"content_block\":{\"type\":\"text\",\"text\":\"\"}}\n\n"+
		"event: content_block_delta\n"+
		"data: {\"type\":\"content_block_delta\",\"index\":0,\"delta\":{\"type\":\"text_delta\",\"text\":\"ok\"}}\n\n"+
		"event: content_block_stop\n"+
		"data: {\"type\":\"content_block_stop\",\"index\":0}\n\n"+
		"event: message_delta\n"+
		"data: {\"type\":\"message_delta\",\"delta\":{\"stop_reason\":\"end_turn\"},\"usage\":{\"output_tokens\":2}}\n\n"+
		"event: message_stop\n"+
		"data: {\"type\":\"message_stop\"}\n\n"

	var receivedReq anthropicMessagesRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&receivedReq); err != nil {
			t.Fatalf("decode: %v", err)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, sseData)
	}))
	defer server.Close()

	provider := NewClaudeProvider(server.URL, "key", "claude-3")
	var sp StreamingModelProvider = provider

	toolChoice := skills.ToolChoiceAuto
	ch, err := sp.GenerateStream(context.Background(), skills.GenerateRequest{
		Messages:   []skills.Message{{Role: "user", Content: "test"}},
		ToolChoice: toolChoice,
	})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	for range ch {}

	if receivedReq.ToolChoice == nil {
		t.Fatal("ToolChoice should be set in streaming request")
	}
}

func TestAnthropicStreamingParallelToolCallsPropagation(t *testing.T) {
	sseData := "event: message_start\n"+
		"data: {\"type\":\"message_start\",\"message\":{\"id\":\"msg_1\",\"type\":\"message\",\"role\":\"assistant\",\"content\":[],\"model\":\"claude-3\",\"usage\":{\"input_tokens\":5}}}\n\n"+
		"event: content_block_start\n"+
		"data: {\"type\":\"content_block_start\",\"index\":0,\"content_block\":{\"type\":\"text\",\"text\":\"\"}}\n\n"+
		"event: content_block_delta\n"+
		"data: {\"type\":\"content_block_delta\",\"index\":0,\"delta\":{\"type\":\"text_delta\",\"text\":\"ok\"}}\n\n"+
		"event: content_block_stop\n"+
		"data: {\"type\":\"content_block_stop\",\"index\":0}\n\n"+
		"event: message_delta\n"+
		"data: {\"type\":\"message_delta\",\"delta\":{\"stop_reason\":\"end_turn\"},\"usage\":{\"output_tokens\":2}}\n\n"+
		"event: message_stop\n"+
		"data: {\"type\":\"message_stop\"}\n\n"

	var receivedReq anthropicMessagesRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&receivedReq); err != nil {
			t.Fatalf("decode: %v", err)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, sseData)
	}))
	defer server.Close()

	provider := NewClaudeProvider(server.URL, "key", "claude-3")
	var sp StreamingModelProvider = provider

	disabled := false
	ch, err := sp.GenerateStream(context.Background(), skills.GenerateRequest{
		Messages:         []skills.Message{{Role: "user", Content: "test"}},
		ParallelToolCalls: &disabled,
	})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	for range ch {}

	if receivedReq.DisableParallelUse == nil || !*receivedReq.DisableParallelUse {
		t.Error("DisableParallelUse should be true when ParallelToolCalls=false")
	}
}
