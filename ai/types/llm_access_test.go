package types

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestLLMAccessFunc_Adapter(t *testing.T) {
	called := false
	var capturedReq LLMRequest

	fn := LLMAccessFunc(func(ctx context.Context, req LLMRequest) (*LLMResponse, error) {
		called = true
		capturedReq = req
		return &LLMResponse{
			Content:   "generated text",
			Usage:     TokenUsage{PromptTokens: 10, CompletionTokens: 5, TotalTokens: 15},
			ModelUsed: "test-model",
			Latency:   50 * time.Millisecond,
		}, nil
	})

	var _ LLMAccess = fn

	req := LLMRequest{
		SystemPrompt: "you are a helper",
		Contents:     []ContentBlock{NewTextBlock("hello")},
		MaxTokens:    100,
		Temperature:  0.7,
	}

	resp, err := fn.Generate(context.Background(), req)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if !called {
		t.Error("adapter function was not called")
	}
	if capturedReq.SystemPrompt != "you are a helper" {
		t.Errorf("SystemPrompt = %q, want %q", capturedReq.SystemPrompt, "you are a helper")
	}
	if len(capturedReq.Contents) != 1 || capturedReq.Contents[0].Text != "hello" {
		t.Errorf("Contents = %+v, want single text block 'hello'", capturedReq.Contents)
	}
	if resp.Content != "generated text" {
		t.Errorf("Content = %q, want %q", resp.Content, "generated text")
	}
	if resp.Usage.TotalTokens != 15 {
		t.Errorf("Usage.TotalTokens = %d, want 15", resp.Usage.TotalTokens)
	}
	if resp.ModelUsed != "test-model" {
		t.Errorf("ModelUsed = %q, want %q", resp.ModelUsed, "test-model")
	}
	if resp.Latency != 50*time.Millisecond {
		t.Errorf("Latency = %v, want 50ms", resp.Latency)
	}
}

func TestLLMRequest_Contents(t *testing.T) {
	req := LLMRequest{
		Contents: []ContentBlock{
			NewTextBlock("describe this"),
			NewImageBlock("https://example.com/photo.png"),
		},
		MaxTokens: 256,
	}
	if len(req.Contents) != 2 {
		t.Fatalf("len(Contents) = %d, want 2", len(req.Contents))
	}
	if req.Contents[0].Type != "text" {
		t.Errorf("Contents[0].Type = %q, want %q", req.Contents[0].Type, "text")
	}
	if req.Contents[1].Type != "image" {
		t.Errorf("Contents[1].Type = %q, want %q", req.Contents[1].Type, "image")
	}
}

func TestLLMRequest_OptionalFields(t *testing.T) {
	req := LLMRequest{
		Contents:  []ContentBlock{NewTextBlock("ping")},
		MaxTokens: 50,
	}
	if req.SystemPrompt != "" {
		t.Errorf("SystemPrompt = %q, want empty", req.SystemPrompt)
	}
	if req.Temperature != 0 {
		t.Errorf("Temperature = %f, want 0", req.Temperature)
	}
}

func TestLLMResponse_Fields(t *testing.T) {
	resp := LLMResponse{
		Content: "result",
		Usage: TokenUsage{
			PromptTokens:     100,
			CompletionTokens: 50,
			TotalTokens:      150,
		},
		ModelUsed: "gpt-4",
		Latency:   200 * time.Millisecond,
	}
	if resp.Content != "result" {
		t.Errorf("Content = %q, want %q", resp.Content, "result")
	}
	if resp.Usage.PromptTokens != 100 {
		t.Errorf("Usage.PromptTokens = %d, want 100", resp.Usage.PromptTokens)
	}
	if resp.ModelUsed != "gpt-4" {
		t.Errorf("ModelUsed = %q, want %q", resp.ModelUsed, "gpt-4")
	}
	if resp.Latency != 200*time.Millisecond {
		t.Errorf("Latency = %v, want 200ms", resp.Latency)
	}
}

func TestLLMAccessFunc_ReturnsError(t *testing.T) {
	testErr := errors.New("llm failed")
	fn := LLMAccessFunc(func(ctx context.Context, req LLMRequest) (*LLMResponse, error) {
		return nil, testErr
	})
	_, err := fn.Generate(context.Background(), LLMRequest{})
	if err != testErr {
		t.Errorf("error = %v, want testErr", err)
	}
}
