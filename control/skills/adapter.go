package skills

import (
	"encoding/json"
	"fmt"
)

// NormalizeArguments parses tool call arguments from JSON string to map.
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
