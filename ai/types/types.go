package types

import (
	"encoding/json"
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

	// CapStreaming indicates the model supports streaming responses.
	CapStreaming CapabilityType = "streaming"
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

// ToolChoiceMode controls how the model selects tools.
type ToolChoiceMode string

const (
	// ToolChoiceAuto lets the model decide whether to call a tool.
	ToolChoiceAuto ToolChoiceMode = "auto"

	// ToolChoiceRequired forces the model to call at least one tool.
	ToolChoiceRequired ToolChoiceMode = "required"

	// ToolChoiceNone prevents the model from calling any tool.
	ToolChoiceNone ToolChoiceMode = "none"
)

// ToolChoiceSpecific selects a specific tool by name.
type ToolChoiceSpecific struct {
	Name string `json:"name"`
}

// Message represents a single message in the conversation.
type Message struct {
	Role    string // "system", "user", "assistant", "tool"
	Content string
	Name    string // for tool messages
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

	// ToolChoice controls tool selection behavior.
	// Accepts: ToolChoiceMode (auto/required/none), ToolChoiceSpecific, or nil.
	// nil means provider default (typically auto).
	ToolChoice any

	// ParallelToolCalls enables the model to call multiple tools in a single turn.
	// nil means provider default (typically true for supported models).
	ParallelToolCalls *bool
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

// StreamChunk represents a single chunk in a streaming response.
type StreamChunk struct {
	// Content is the incremental text content (delta, not accumulated).
	Content string
	// ToolCalls is the accumulated tool calls state at this point in the stream.
	// Each chunk contains the full accumulated state, not just the delta.
	// On final chunk, ToolCalls is complete and ready to use.
	ToolCalls []ToolCall
	// FinishReason is populated only on the final chunk.
	FinishReason string
	// Usage is populated only on the final chunk (if provider supports it).
	Usage TokenUsage
	// Error is non-nil on stream error. This is the final chunk before channel close.
	Error error
}

// ToolDefinition describes a tool available to the model.
type ToolDefinition struct {
	Name        string
	Description string
	Parameters  *JSONSchema
}

// JSONSchema represents a JSON Schema definition compatible with OpenAI,
// Anthropic, and MCP tool schemas. All fields are optional (omitempty) for
// backward compatibility — existing schemas with only Type/Properties/Required
// serialize identically to before this expansion.
type JSONSchema struct {
	// Core fields (original)
	Type       string                 `json:"type,omitempty"`
	Properties map[string]*JSONSchema `json:"properties,omitempty"`
	Required   []string               `json:"required,omitempty"`

	// Metadata
	Description string `json:"description,omitempty"`
	Title       string `json:"title,omitempty"`
	Default     any    `json:"default,omitempty"`
	Examples    []any  `json:"examples,omitempty"`

	// Constraints — string
	MinLength *int   `json:"minLength,omitempty"`
	MaxLength *int   `json:"maxLength,omitempty"`
	Pattern   string `json:"pattern,omitempty"`

	// Constraints — numeric
	Minimum *float64 `json:"minimum,omitempty"`
	Maximum *float64 `json:"maximum,omitempty"`

	// Constraints — enum
	Enum []any `json:"enum,omitempty"`

	// Constraints — array
	Items *JSONSchema `json:"items,omitempty"`

	// Constraints — object
	AdditionalProperties *bool `json:"additionalProperties,omitempty"`

	// Composition
	AnyOf []*JSONSchema `json:"anyOf,omitempty"`
	OneOf []*JSONSchema `json:"oneOf,omitempty"`
	AllOf []*JSONSchema `json:"allOf,omitempty"`

	// Definitions (for $ref support)
	Defs map[string]*JSONSchema `json:"$defs,omitempty"`

	// JSON Schema 2020-12 additions
	Const       *ConstValue   `json:"const,omitempty"`
	PrefixItems []*JSONSchema `json:"prefixItems,omitempty"`
	If          *JSONSchema   `json:"if,omitempty"`
	Then        *JSONSchema   `json:"then,omitempty"`
	Else        *JSONSchema   `json:"else,omitempty"`
	Schema      string        `json:"$schema,omitempty"`
}

// ConstValue wraps a constant value for JSON Schema const validation.
// A nil pointer means the const constraint is not set.
// A non-nil pointer with Val=nil represents const: null.
type ConstValue struct {
	Val any
}

// MarshalJSON implements json.Marshaler so that ConstValue{Val: x} serializes as x.
func (c *ConstValue) MarshalJSON() ([]byte, error) {
	return json.Marshal(c.Val)
}

// UnmarshalJSON implements json.Unmarshaler so that "hello" deserializes to &ConstValue{Val: "hello"}.
func (c *ConstValue) UnmarshalJSON(data []byte) error {
	var v any
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	c.Val = v
	return nil
}
