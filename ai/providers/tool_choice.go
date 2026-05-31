package providers

import (
	"log/slog"
	"fmt"

	"github.com/openbotstack/openbotstack-core/ai/types"
)

// openAIToolChoice represents tool_choice in OpenAI-specific format (function selection only).
type openAIToolChoice struct {
	Type     string `json:"type"`
	Function struct {
		Name string `json:"name"`
	} `json:"function"`
}

// anthropicToolChoice represents tool_choice in Anthropic format (any/tool modes only).
type anthropicToolChoice struct {
	Type string `json:"type"`
	Name string `json:"name,omitempty"`
}

// mapToolChoiceToOpenAI converts types.ToolChoice to OpenAI-compatible format.
// Returns nil for default behavior.
// OpenAI expects plain strings for auto/required/none, object only for function selection.
func mapToolChoiceToOpenAI(tc any) any {
	if tc == nil {
		return nil
	}

	switch v := tc.(type) {
	case types.ToolChoiceMode:
		switch v {
		case types.ToolChoiceAuto:
			return "auto"
		case types.ToolChoiceRequired:
			return "required"
		case types.ToolChoiceNone:
			return "none"
		default:
			slog.Warn("mapToolChoiceToOpenAI: unknown ToolChoiceMode, ignoring", "value", string(v))
			return nil
		}
	case types.ToolChoiceSpecific:
		result := openAIToolChoice{Type: "function"}
		result.Function.Name = v.Name
		return result
	default:
		slog.Warn("mapToolChoiceToOpenAI: unrecognized tool_choice type, ignoring", "type", fmt.Sprintf("%T", tc))
		return nil
	}
}

// mapToolChoiceToAnthropic converts types.ToolChoice to Anthropic format.
// Returns nil for default behavior.
// Anthropic accepts strings for auto/none, object for any/tool modes.
func mapToolChoiceToAnthropic(tc any) any {
	if tc == nil {
		return nil
	}

	switch v := tc.(type) {
	case types.ToolChoiceMode:
		switch v {
		case types.ToolChoiceAuto:
			return "auto"
		case types.ToolChoiceRequired:
			return anthropicToolChoice{Type: "any"}
		case types.ToolChoiceNone:
			return "none"
		default:
			slog.Warn("mapToolChoiceToAnthropic: unknown ToolChoiceMode, ignoring", "value", string(v))
			return nil
		}
	case types.ToolChoiceSpecific:
		return anthropicToolChoice{Type: "tool", Name: v.Name}
	default:
		slog.Warn("mapToolChoiceToAnthropic: unrecognized tool_choice type, ignoring", "type", fmt.Sprintf("%T", tc))
		return nil
	}
}
