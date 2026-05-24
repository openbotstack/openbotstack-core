package providers

import (
	"log/slog"
	"fmt"

	skills "github.com/openbotstack/openbotstack-core/control/skills"
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

// mapToolChoiceToOpenAI converts skills.ToolChoice to OpenAI-compatible format.
// Returns nil for default behavior.
// OpenAI expects plain strings for auto/required/none, object only for function selection.
func mapToolChoiceToOpenAI(tc any) any {
	if tc == nil {
		return nil
	}

	switch v := tc.(type) {
	case skills.ToolChoiceMode:
		switch v {
		case skills.ToolChoiceAuto:
			return "auto"
		case skills.ToolChoiceRequired:
			return "required"
		case skills.ToolChoiceNone:
			return "none"
		default:
			slog.Warn("mapToolChoiceToOpenAI: unknown ToolChoiceMode, ignoring", "value", string(v))
			return nil
		}
	case skills.ToolChoiceSpecific:
		result := openAIToolChoice{Type: "function"}
		result.Function.Name = v.Name
		return result
	default:
		slog.Warn("mapToolChoiceToOpenAI: unrecognized tool_choice type, ignoring", "type", fmt.Sprintf("%T", tc))
		return nil
	}
}

// mapToolChoiceToAnthropic converts skills.ToolChoice to Anthropic format.
// Returns nil for default behavior.
// Anthropic accepts strings for auto/none, object for any/tool modes.
func mapToolChoiceToAnthropic(tc any) any {
	if tc == nil {
		return nil
	}

	switch v := tc.(type) {
	case skills.ToolChoiceMode:
		switch v {
		case skills.ToolChoiceAuto:
			return "auto"
		case skills.ToolChoiceRequired:
			return anthropicToolChoice{Type: "any"}
		case skills.ToolChoiceNone:
			return "none"
		default:
			slog.Warn("mapToolChoiceToAnthropic: unknown ToolChoiceMode, ignoring", "value", string(v))
			return nil
		}
	case skills.ToolChoiceSpecific:
		return anthropicToolChoice{Type: "tool", Name: v.Name}
	default:
		slog.Warn("mapToolChoiceToAnthropic: unrecognized tool_choice type, ignoring", "type", fmt.Sprintf("%T", tc))
		return nil
	}
}
