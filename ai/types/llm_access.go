package types

import (
	"context"
	"time"
)

// LLMRequest is a restricted LLM request for builtin tools.
type LLMRequest struct {
	SystemPrompt string         `json:"system_prompt,omitempty"`
	Contents     []ContentBlock `json:"contents"`
	MaxTokens    int            `json:"max_tokens"`
	Temperature  float64        `json:"temperature,omitempty"`
}

// LLMResponse is the result of a restricted LLM call from a builtin tool.
type LLMResponse struct {
	Content   string        `json:"content"`
	Usage     TokenUsage    `json:"usage"`
	ModelUsed string        `json:"model_used"`
	Latency   time.Duration `json:"latency"`
}

// LLMAccess is a restricted LLM interface for builtin tools.
// Enforces: vision-capable model routing, no tool-calling, token caps, timeouts.
type LLMAccess interface {
	Generate(ctx context.Context, req LLMRequest) (*LLMResponse, error)
}

// LLMAccessFunc is a function adapter for LLMAccess.
type LLMAccessFunc func(ctx context.Context, req LLMRequest) (*LLMResponse, error)

func (f LLMAccessFunc) Generate(ctx context.Context, req LLMRequest) (*LLMResponse, error) {
	return f(ctx, req)
}
