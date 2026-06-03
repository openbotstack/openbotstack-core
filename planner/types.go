package planner

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
