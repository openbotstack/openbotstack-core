// Package planning provides shared types for the planning and execution subsystems.
//
// This package exists to break the circular dependency between the execution and
// planner packages: planner imports execution for plan types (ExecutionPlan,
// ExecutionStep), while execution needs the PlannerContext type. By extracting the
// shared types here, both packages can import planning without creating a cycle.
//
// Canonical types defined here:
//   - TurnToolResult: structured tool execution result (used by planner and execution)
//   - AssistantSoul: behavioral parameters of an assistant
//   - SearchResult: semantic search result entry
//   - ProgressFn: callback signature for progress events
//   - PlannerContext: unified state for generating an execution plan
package planning

// AssistantSoul defines the behavioral parameters of an assistant.
// It acts as the "inner logic" and "personality" that guides the LLM.
type AssistantSoul struct {
	SystemPrompt  string   `json:"system_prompt"`
	Personality   string   `json:"personality"`
	Instructions  string   `json:"instructions"`
	AllowedSkills []string `json:"allowed_skills"`
	AllowedTools  []string `json:"allowed_tools"`
}

// SearchResult represents a single entry found during a semantic search.
type SearchResult struct {
	Content []byte
	Score   float32
}

// ProgressFn is the callback signature for progress events.
type ProgressFn func(eventType, content string)
