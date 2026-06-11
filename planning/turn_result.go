package planning

// TurnToolResult captures the outcome of a single tool/skill execution within a reasoning turn.
// Used by the ReasoningLoop to feed structured results back to the planner,
// replacing the legacy XML <observation> injection hack.
type TurnToolResult struct {
	StepName string `json:"step_name"`
	StepType string `json:"step_type"`           // "tool" | "skill"
	Success  bool   `json:"success"`
	Summary  string `json:"summary,omitempty"`    // Human-readable one-line summary
	Output   string `json:"output,omitempty"`     // Truncated output (max 500 chars)
	Error    string `json:"error,omitempty"`      // Error message if failed
}
