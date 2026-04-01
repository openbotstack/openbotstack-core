package skills

import (
	"time"
)

// CapabilityType defines a model capability.
type CapabilityType string

const (
	// CapTextGeneration indicates the model can generate text.
	CapTextGeneration CapabilityType = "text_generation"

	// CapToolCalling indicates the model supports function/tool calling.
	CapToolCalling CapabilityType = "tool_calling"

	// CapJSONMode indicates the model can output structured JSON.
	CapJSONMode CapabilityType = "json_mode"

	// CapEmbedding indicates the model can generate embeddings.
	CapEmbedding CapabilityType = "embedding"

	// CapVision indicates the model supports image input.
	CapVision CapabilityType = "vision"
)

// ModelConstraints defines routing constraints for model selection.
type ModelConstraints struct {
	// MaxLatencyMs is the maximum acceptable latency in milliseconds.
	MaxLatencyMs int64

	// Privacy indicates data handling requirements.
	// Values: "public", "internal", "private"
	Privacy string

	// PreferredProvider hints at a specific provider if available.
	PreferredProvider string
}

// GenerateRequest is the input to a model generation call.
type GenerateRequest struct {
	// Messages is the conversation history.
	Messages []Message

	// Tools is the list of available tools for tool calling.
	Tools []ToolDefinition

	// MaxTokens limits the response length.
	MaxTokens int

	// Temperature controls randomness (0.0-2.0).
	Temperature float64

	// JSONSchema for structured output (if CapJSONMode).
	JSONSchema *JSONSchema
}

// Message represents a single message in the conversation.
type Message struct {
	Role    string // "system", "user", "assistant", "tool"
	Content string
	Name    string // for tool messages
}

// ToolDefinition describes a tool available to the model.
type ToolDefinition struct {
	Name        string
	Description string
	Parameters  *JSONSchema
}

// JSONSchema is a simplified JSON Schema representation.
type JSONSchema struct {
	Type       string                 `json:"type,omitempty"`
	Properties map[string]*JSONSchema `json:"properties,omitempty"`
	Required   []string               `json:"required,omitempty"`
}

// GenerateResponse is the output from a model generation call.
type GenerateResponse struct {
	// Content is the generated text.
	Content string

	// ToolCalls contains any tool invocations.
	ToolCalls []ToolCall

	// Usage tracks token consumption.
	Usage TokenUsage

	// FinishReason indicates why generation stopped.
	FinishReason string

	// Latency is the actual response time.
	Latency time.Duration
}

// ToolCall represents a single tool invocation by the model.
type ToolCall struct {
	ID        string
	Name      string
	Arguments string // JSON string
}

// TokenUsage tracks token consumption.
type TokenUsage struct {
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
}
