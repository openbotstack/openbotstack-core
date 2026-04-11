package providers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/openbotstack/openbotstack-core/ai"
	"github.com/openbotstack/openbotstack-core/control/skills"
)

func TestClaudeProviderID(t *testing.T) {
	provider := NewClaudeProvider("", "test-api-key", "claude-3-opus-20240229")
	if provider.ID() != "anthropic/claude-3-opus-20240229" {
		t.Errorf("Expected ID 'anthropic/claude-3-opus-20240229', got '%s'", provider.ID())
	}
}

func TestClaudeProviderCapabilities(t *testing.T) {
	provider := NewClaudeProvider("", "test-api-key", "claude-3-opus-20240229")
	caps := provider.Capabilities()

	expected := []skills.CapabilityType{
		skills.CapTextGeneration,
		skills.CapToolCalling,
		skills.CapVision,
	}

	if len(caps) != len(expected) {
		t.Fatalf("Expected %d capabilities, got %d", len(expected), len(caps))
	}

	for i, cap := range expected {
		if caps[i] != cap {
			t.Errorf("Expected capability %s at index %d, got %s", cap, i, caps[i])
		}
	}
}

func TestOpenAIProviderID(t *testing.T) {
	provider := NewOpenAIProvider("", "test-api-key", "gpt-4o")
	if provider.ID() != "openai/gpt-4o" {
		t.Errorf("Expected ID 'openai/gpt-4o', got '%s'", provider.ID())
	}
}

func TestOpenAIProviderCapabilities(t *testing.T) {
	provider := NewOpenAIProvider("", "test-api-key", "gpt-4o")
	caps := provider.Capabilities()

	expected := []skills.CapabilityType{
		skills.CapTextGeneration,
		skills.CapToolCalling,
		skills.CapJSONMode,
		skills.CapVision,
	}

	if len(caps) != len(expected) {
		t.Fatalf("Expected %d capabilities, got %d", len(expected), len(caps))
	}
}

func TestModelScopeProviderID(t *testing.T) {
	provider := NewModelScopeProvider("", "test-api-key", "qwen-max")
	if provider.ID() != "modelscope/qwen-max" {
		t.Errorf("Expected ID 'modelscope/qwen-max', got '%s'", provider.ID())
	}
}

func TestSiliconFlowProviderID(t *testing.T) {
	provider := NewSiliconFlowProvider("", "test-api-key", "deepseek-v3")
	if provider.ID() != "siliconflow/deepseek-v3" {
		t.Errorf("Expected ID 'siliconflow/deepseek-v3', got '%s'", provider.ID())
	}
}

func TestProviderGenerateNoAPIKey(t *testing.T) {
	provider := NewOpenAIProvider("", "", "gpt-4o")
	_, err := provider.Generate(context.Background(), skills.GenerateRequest{
		Messages: []skills.Message{{Role: "user", Content: "Hello"}},
	})
	if err == nil {
		t.Error("Expected error for empty API key, got nil")
	}
}

// TestOpenAICompatibleGenerate tests the real HTTP call path with httptest.
func TestOpenAICompatibleGenerate(t *testing.T) {
	mockResp := chatResponse{
		Choices: []chatChoice{
			{
				Message: chatResponseMessage{
					Role:    "assistant",
					Content: "Hello from mock!",
				},
				FinishReason: "stop",
			},
		},
		Usage: chatUsage{
			PromptTokens:     5,
			CompletionTokens: 4,
			TotalTokens:      9,
		},
	}

	var receivedRequest chatRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request format
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/chat/completions" {
			t.Errorf("Expected /chat/completions, got %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("Expected Bearer test-key, got %s", r.Header.Get("Authorization"))
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Expected application/json Content-Type")
		}

		if err := json.NewDecoder(r.Body).Decode(&receivedRequest); err != nil {
			t.Fatalf("Failed to decode request: %v", err)
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(mockResp); err != nil {
			t.Fatalf("Failed to encode response: %v", err)
		}
	}))
	defer server.Close()

	provider := NewOpenAIProvider(server.URL, "test-key", "gpt-4o-test")

	req := skills.GenerateRequest{
		Messages: []skills.Message{
			{Role: "system", Content: "You are helpful."},
			{Role: "user", Content: "Hello"},
		},
		MaxTokens:   100,
		Temperature: 0.7,
	}

	resp, err := provider.Generate(context.Background(), req)
	if err != nil {
		t.Fatalf("Generate returned unexpected error: %v", err)
	}

	if resp.Content != "Hello from mock!" {
		t.Errorf("Expected 'Hello from mock!', got '%s'", resp.Content)
	}
	if resp.FinishReason != "stop" {
		t.Errorf("Expected finish_reason 'stop', got '%s'", resp.FinishReason)
	}
	if resp.Usage.TotalTokens != 9 {
		t.Errorf("Expected 9 total tokens, got %d", resp.Usage.TotalTokens)
	}
	if resp.Latency <= 0 {
		t.Error("Expected positive latency")
	}

	// Verify request was properly formatted
	if receivedRequest.Model != "gpt-4o-test" {
		t.Errorf("Expected model 'gpt-4o-test', got '%s'", receivedRequest.Model)
	}
	if len(receivedRequest.Messages) != 2 {
		t.Fatalf("Expected 2 messages, got %d", len(receivedRequest.Messages))
	}
	if receivedRequest.Messages[0].Role != "system" {
		t.Errorf("Expected first message role 'system', got '%s'", receivedRequest.Messages[0].Role)
	}
}

// TestOpenAICompatibleGenerateWithToolCalls tests tool calling response parsing.
func TestOpenAICompatibleGenerateWithToolCalls(t *testing.T) {
	mockResp := chatResponse{
		Choices: []chatChoice{
			{
				Message: chatResponseMessage{
					Role: "assistant",
					ToolCalls: []chatToolCall{
						{
							ID:   "call_abc123",
							Type: "function",
							Function: chatFunctionCall{
								Name:      "get_weather",
								Arguments: `{"location":"San Francisco"}`,
							},
						},
					},
				},
				FinishReason: "tool_calls",
			},
		},
		Usage: chatUsage{TotalTokens: 20},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockResp) //nolint:errcheck
	}))
	defer server.Close()

	provider := NewOpenAIProvider(server.URL, "test-key", "gpt-4o")
	resp, err := provider.Generate(context.Background(), skills.GenerateRequest{
		Messages: []skills.Message{{Role: "user", Content: "What's the weather?"}},
		Tools: []skills.ToolDefinition{
			{Name: "get_weather", Description: "Get weather for a location"},
		},
	})
	if err != nil {
		t.Fatalf("Generate returned unexpected error: %v", err)
	}

	if len(resp.ToolCalls) != 1 {
		t.Fatalf("Expected 1 tool call, got %d", len(resp.ToolCalls))
	}
	if resp.ToolCalls[0].Name != "get_weather" {
		t.Errorf("Expected tool name 'get_weather', got '%s'", resp.ToolCalls[0].Name)
	}
	if resp.ToolCalls[0].ID != "call_abc123" {
		t.Errorf("Expected tool call ID 'call_abc123', got '%s'", resp.ToolCalls[0].ID)
	}
	if resp.FinishReason != "tool_calls" {
		t.Errorf("Expected finish_reason 'tool_calls', got '%s'", resp.FinishReason)
	}
}

// TestOpenAICompatibleGenerateAPIError tests error response handling.
func TestOpenAICompatibleGenerateAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]string{
				"message": "Invalid API key",
				"type":    "authentication_error",
			},
		}) //nolint:errcheck
	}))
	defer server.Close()

	provider := NewOpenAIProvider(server.URL, "bad-key", "gpt-4o")
	_, err := provider.Generate(context.Background(), skills.GenerateRequest{
		Messages: []skills.Message{{Role: "user", Content: "Hello"}},
	})
	if err == nil {
		t.Error("Expected error for 401 response, got nil")
	}
}

func TestSyncRetryOn500(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if callCount < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		mockResp := chatResponse{
			Choices: []chatChoice{{Message: chatResponseMessage{Content: "success"}, FinishReason: "stop"}},
		}
		json.NewEncoder(w).Encode(mockResp)
	}))
	defer server.Close()

	p := &openAIProvider{
		baseURL: server.URL, apiKey: "key", modelName: "model",
		client:       &http.Client{Timeout: 10 * time.Second},
		maxRetries:   3,
		capabilities: []skills.CapabilityType{skills.CapTextGeneration},
	}
	resp, err := p.Generate(context.Background(), skills.GenerateRequest{
		Messages: []skills.Message{{Role: "user", Content: "hi"}},
	})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if resp.Content != "success" {
		t.Errorf("Expected 'success', got '%s'", resp.Content)
	}
	if callCount != 3 {
		t.Errorf("Expected 3 calls (2 failures + 1 success), got %d", callCount)
	}
}

func TestSyncRetryExhaustion(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	p := &openAIProvider{
		baseURL: server.URL, apiKey: "key", modelName: "model",
		client:       &http.Client{Timeout: 10 * time.Second},
		maxRetries:   2,
		capabilities: []skills.CapabilityType{skills.CapTextGeneration},
	}
	_, err := p.Generate(context.Background(), skills.GenerateRequest{
		Messages: []skills.Message{{Role: "user", Content: "hi"}},
	})
	if err == nil {
		t.Error("Expected error after retry exhaustion")
	}
}

func TestSyncNoRetryOn400(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer server.Close()

	p := &openAIProvider{
		baseURL: server.URL, apiKey: "key", modelName: "model",
		client:       &http.Client{Timeout: 10 * time.Second},
		maxRetries:   3,
		capabilities: []skills.CapabilityType{skills.CapTextGeneration},
	}
	_, err := p.Generate(context.Background(), skills.GenerateRequest{
		Messages: []skills.Message{{Role: "user", Content: "hi"}},
	})
	if err == nil {
		t.Error("Expected error for 400")
	}
	if callCount != 1 {
		t.Errorf("Expected 1 call (no retry for 4xx), got %d", callCount)
	}
}

func TestSyncTypedErrors(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		wantErr    error
	}{
		{"401 unauthorized", 401, ai.ErrAuthenticationFailed},
		{"403 forbidden", 403, ai.ErrAuthenticationFailed},
		{"400 bad request", 400, ai.ErrBadRequest},
		{"500 server error", 500, ai.ErrProviderUnavailable},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
			}))
			defer server.Close()

			p := &openAIProvider{
				baseURL: server.URL, apiKey: "key", modelName: "model",
				client:       &http.Client{Timeout: 10 * time.Second},
				capabilities: []skills.CapabilityType{skills.CapTextGeneration},
			}
			_, err := p.Generate(context.Background(), skills.GenerateRequest{
				Messages: []skills.Message{{Role: "user", Content: "hi"}},
			})
			if err == nil {
				t.Fatal("Expected error")
			}
			if !strings.Contains(err.Error(), tt.wantErr.Error()) {
				t.Errorf("Expected error containing '%v', got '%v'", tt.wantErr, err)
			}
		})
	}
}
