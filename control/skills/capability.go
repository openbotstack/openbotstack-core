package skills

import (
	aitypes "github.com/openbotstack/openbotstack-core/ai/types"
)

// LLM wire types — canonical definitions moved to ai/types.
// These aliases maintain backward compatibility during migration.

type CapabilityType = aitypes.CapabilityType

const (
	CapTextGeneration = aitypes.CapTextGeneration
	CapToolCalling    = aitypes.CapToolCalling
	CapJSONMode       = aitypes.CapJSONMode
	CapEmbedding      = aitypes.CapEmbedding
	CapVision         = aitypes.CapVision
	CapStreaming      = aitypes.CapStreaming
)

type ModelConstraints = aitypes.ModelConstraints

type ToolChoiceMode = aitypes.ToolChoiceMode
type ToolChoiceSpecific = aitypes.ToolChoiceSpecific

const (
	ToolChoiceAuto     = aitypes.ToolChoiceAuto
	ToolChoiceRequired = aitypes.ToolChoiceRequired
	ToolChoiceNone     = aitypes.ToolChoiceNone
)

type Message = aitypes.Message

type SkillDescriptor struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema *aitypes.JSONSchema    `json:"input_schema,omitempty"`
	Kind        string                 `json:"kind,omitempty"`
	SourceID    string                 `json:"source_id,omitempty"`
}

type ToolDefinition = aitypes.ToolDefinition
type JSONSchema = aitypes.JSONSchema
type ConstValue = aitypes.ConstValue
type GenerateRequest = aitypes.GenerateRequest
type GenerateResponse = aitypes.GenerateResponse
type ToolCall = aitypes.ToolCall
type TokenUsage = aitypes.TokenUsage
type StreamChunk = aitypes.StreamChunk
