package skills

import (
	"encoding/json"
	"fmt"
)

// SkillInfo provides the minimal interface needed to convert a skill to a tool.
// This avoids importing registry/skills which would create a circular dependency.
type SkillInfo interface {
	ID() string
	Description() string
	InputSchema() *JSONSchema
}

// SkillsToOpenAITools converts skills to ToolDefinition format
// compatible with OpenAI function calling API.
func SkillsToOpenAITools(skills []SkillInfo) []ToolDefinition {
	return skillsToTools(skills)
}

// SkillsToAnthropicTools converts skills to ToolDefinition format
// compatible with Anthropic tool use API.
func SkillsToAnthropicTools(skills []SkillInfo) []ToolDefinition {
	return skillsToTools(skills)
}

// NormalizeArguments parses tool call arguments from JSON string to map.
// Handles both JSON string (from OpenAI) and empty string (no args).
func NormalizeArguments(args string) (map[string]any, error) {
	if args == "" {
		return map[string]any{}, nil
	}
	var result map[string]any
	if err := json.Unmarshal([]byte(args), &result); err != nil {
		return nil, fmt.Errorf("arguments must be a JSON object: %w", err)
	}
	if result == nil {
		return map[string]any{}, nil
	}
	return result, nil
}

func skillsToTools(skills []SkillInfo) []ToolDefinition {
	tools := make([]ToolDefinition, 0, len(skills))
	for _, s := range skills {
		tools = append(tools, ToolDefinition{
			Name:        s.ID(),
			Description: s.Description(),
			Parameters:  s.InputSchema(),
		})
	}
	return tools
}
