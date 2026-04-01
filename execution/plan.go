package execution

import (
	"encoding/json"
	"fmt"
)

// StepType indicates whether a step is a skill or a tool.
type StepType string

const (
	StepTypeSkill StepType = "skill"
	StepTypeTool  StepType = "tool"
)

// ExecutionStep represents a single action in an execution plan.
type ExecutionStep struct {
	Name      string         `json:"name"`
	Type      StepType       `json:"type"`
	Arguments map[string]any `json:"arguments"`
}

// ArgumentsJSON returns the arguments serialized as JSON bytes.
func (s *ExecutionStep) ArgumentsJSON() ([]byte, error) {
	if s.Arguments == nil {
		return []byte("{}"), nil
	}
	return json.Marshal(s.Arguments)
}

// ExecutionPlan specifies a sequence of steps to achieve a goal.
type ExecutionPlan struct {
	AssistantID string          `json:"assistant_id"`
	Steps       []ExecutionStep `json:"steps"`
	Reasoning   string          `json:"reasoning,omitempty"`
}

// Validate checks if the execution plan is well-formed.
func (p *ExecutionPlan) Validate() error {
	if len(p.Steps) == 0 {
		return fmt.Errorf("plan must have at least one step")
	}
	for i, step := range p.Steps {
		if step.Name == "" {
			return fmt.Errorf("step %d has empty name", i)
		}
		if step.Type != StepTypeSkill && step.Type != StepTypeTool {
			return fmt.Errorf("step %d has invalid type: %s", i, step.Type)
		}
	}
	return nil
}
