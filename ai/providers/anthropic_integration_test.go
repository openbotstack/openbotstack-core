//go:build integration

package providers

import (
	"context"
	"testing"
	"time"

	"github.com/openbotstack/openbotstack-core/control/skills"
)

const (
	testAnthropicBaseURL = "http://10.10.100.20:3001/v1"
	testAnthropicAPIKey  = "sk-HxRAcVgnfBGnOpQj4icAwhyGkGtjmdNWsBwlRpAIUVbFjFrF"
	testAnthropicModel   = "Qwen3.6-35B"
)

func TestAnthropicIntegrationGenerate(t *testing.T) {
	provider := NewClaudeProvider(testAnthropicBaseURL, testAnthropicAPIKey, testAnthropicModel)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	resp, err := provider.Generate(ctx, skills.GenerateRequest{
		Messages: []skills.Message{
			{Role: "system", Content: "You are a helpful assistant. Reply concisely."},
			{Role: "user", Content: "Say 'Hello World' and nothing else."},
		},
		MaxTokens:   2048,
		Temperature: 0.1,
	})
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	t.Logf("Response: content=%q finish=%s usage=%+v latency=%v",
		resp.Content, resp.FinishReason, resp.Usage, resp.Latency)

	if resp.Content == "" {
		t.Error("Expected non-empty content")
	}
	if resp.FinishReason == "" {
		t.Error("Expected non-empty finish_reason")
	}
	if resp.Usage.PromptTokens == 0 {
		t.Error("Expected non-zero prompt tokens")
	}
}

func TestAnthropicIntegrationStreaming(t *testing.T) {
	provider := NewClaudeProvider(testAnthropicBaseURL, testAnthropicAPIKey, testAnthropicModel)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var sp StreamingModelProvider = provider
	ch, err := sp.GenerateStream(ctx, skills.GenerateRequest{
		Messages: []skills.Message{
			{Role: "user", Content: "Count from 1 to 5."},
		},
		MaxTokens:   2048,
		Temperature: 0.1,
	})
	if err != nil {
		t.Fatalf("GenerateStream failed: %v", err)
	}

	var fullContent string
	var gotFinish bool
	for chunk := range ch {
		if chunk.Error != nil {
			t.Fatalf("Stream error: %v", chunk.Error)
		}
		fullContent += chunk.Content
		if chunk.FinishReason != "" {
			gotFinish = true
			t.Logf("Finish reason: %s, usage: %+v", chunk.FinishReason, chunk.Usage)
		}
	}

	t.Logf("Streamed content: %q", fullContent)

	if fullContent == "" {
		t.Error("Expected non-empty streamed content")
	}
	if !gotFinish {
		t.Error("Expected finish_reason in stream")
	}
}

func TestAnthropicIntegrationToolCalls(t *testing.T) {
	provider := NewClaudeProvider(testAnthropicBaseURL, testAnthropicAPIKey, testAnthropicModel)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	resp, err := provider.Generate(ctx, skills.GenerateRequest{
		Messages: []skills.Message{
			{Role: "user", Content: "What is the weather in Tokyo?"},
		},
		Tools: []skills.ToolDefinition{
			{
				Name:        "get_weather",
				Description: "Get the current weather for a given city",
				Parameters: &skills.JSONSchema{
					Type: "object",
					Properties: map[string]*skills.JSONSchema{
						"city": {Type: "string"},
					},
					Required: []string{"city"},
				},
			},
		},
		MaxTokens:   2048,
	})
	if err != nil {
		t.Fatalf("Generate with tools failed: %v", err)
	}

	t.Logf("Content: %q", resp.Content)
	t.Logf("Tool calls: %d", len(resp.ToolCalls))
	t.Logf("Finish reason: %s", resp.FinishReason)

	if len(resp.ToolCalls) == 0 {
		t.Log("No tool calls returned (model may have answered directly)")
	} else {
		tc := resp.ToolCalls[0]
		t.Logf("Tool call: id=%s name=%s args=%s", tc.ID, tc.Name, tc.Arguments)
		if tc.Name != "get_weather" {
			t.Errorf("Expected tool name 'get_weather', got '%s'", tc.Name)
		}
	}
}
