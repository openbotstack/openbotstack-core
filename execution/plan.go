package execution

import (
	"encoding/json"
	"fmt"

	"github.com/google/uuid"

	"github.com/openbotstack/openbotstack-core/execution/template"
)

// StepType indicates whether a step is a skill or a tool.
type StepType string

const (
	StepTypeSkill StepType = "skill"
	StepTypeTool  StepType = "tool"
	StepTypeLLM   StepType = "llm" // iterative LLM reasoning within a single step
)

// ExecutionStep represents a single action in an execution plan.
type ExecutionStep struct {
	Name      string         `json:"name"`
	Type      StepType       `json:"type"`
	Arguments map[string]any `json:"arguments"`

	// StepID is an auto-generated unique identifier for audit tracing.
	// Populated during Validate() if empty.
	StepID string `json:"step_id,omitempty"`

	// Timeout is the per-step wall-clock timeout in milliseconds.
	// Zero means no per-step timeout (uses session-level timeout).
	Timeout int64 `json:"timeout_ms,omitempty"`

	// ExpectedOutput is a human-readable description of the expected result.
	// Used for documentation and audit, not enforced programmatically.
	ExpectedOutput string `json:"expected_output,omitempty"`

	// Parallelizable indicates this step can run concurrently with other steps
	// in the same ParallelGroup. Steps with the same non-empty ParallelGroup
	// may be dispatched in parallel by the executor.
	Parallelizable bool   `json:"parallelizable,omitempty"`
	ParallelGroup  string `json:"parallel_group,omitempty"`

	// RiskLevel is the skill's risk classification propagated from manifest.
	// Values: "info" (default), "sensitive", "clinical", "critical".
	RiskLevel string `json:"risk_level,omitempty"`
}

// Clone returns a shallow copy with a cloned Arguments map so mutations
// (CoerceStringNumbers, ResolveArguments) don't affect the original plan step.
func (s *ExecutionStep) Clone() *ExecutionStep {
	cp := *s
	if s.Arguments != nil {
		cp.Arguments = make(map[string]any, len(s.Arguments))
		for k, v := range s.Arguments {
			cp.Arguments[k] = v
		}
	}
	return &cp
}

// ArgumentsJSON returns the arguments serialized as JSON bytes.
func (s *ExecutionStep) ArgumentsJSON() ([]byte, error) {
	if s.Arguments == nil {
		return []byte("{}"), nil
	}
	return json.Marshal(s.Arguments)
}

// ResolveArguments replaces {{step_name}} and {{step_name.field}} references in
// string argument values with the corresponding results from previously executed steps.
// Delegates to the template package for resolution logic.
func (s *ExecutionStep) ResolveArguments(results map[string]any) {
	if s.Arguments == nil || len(results) == 0 {
		return
	}
	for key, val := range s.Arguments {
		strVal, ok := val.(string)
		if !ok {
			continue
		}
		s.Arguments[key] = template.Resolve(strVal, results)
	}
}

// CoerceStringNumbers converts string argument values that represent numbers
// into their native types. Delegates to the template package for coercion logic.
// Returns the number of values that were coerced.
func (s *ExecutionStep) CoerceStringNumbers() int {
	if s.Arguments == nil {
		return 0
	}
	return template.CoerceStringNumbers(s.Arguments)
}

// ExecutionPlan specifies a sequence of steps to achieve a goal.
// Once validated and frozen, the plan is immutable.
type ExecutionPlan struct {
	AssistantID string          `json:"assistant_id"`
	Steps       []ExecutionStep `json:"steps"`
	Reasoning   string          `json:"reasoning,omitempty"`

	frozen bool
}

// Freeze marks the plan as immutable. After freezing, the plan must not be modified.
func (p *ExecutionPlan) Freeze() { p.frozen = true }

// IsFrozen returns whether the plan has been frozen (made immutable).
func (p *ExecutionPlan) IsFrozen() bool { return p.frozen }

// Validate checks if the execution plan is well-formed, auto-generates StepIDs,
// and freezes the plan. Returns an error if the plan is already frozen.
func (p *ExecutionPlan) Validate() error {
	if p.frozen {
		return fmt.Errorf("plan is already frozen and cannot be re-validated")
	}
	if len(p.Steps) == 0 {
		return fmt.Errorf("plan must have at least one step")
	}
	seen := make(map[string]int, len(p.Steps))
	for i := range p.Steps {
		step := &p.Steps[i]
		if step.Name == "" {
			return fmt.Errorf("step %d has empty name", i)
		}
		switch step.Type {
		case StepTypeSkill, StepTypeTool, StepTypeLLM:
			// valid
		default:
			return fmt.Errorf("step %d has invalid type: %s", i, step.Type)
		}
		if prev, exists := seen[step.Name]; exists {
			return fmt.Errorf("duplicate step name %q at positions %d and %d", step.Name, prev, i)
		}
		seen[step.Name] = i

		// Auto-generate StepID if empty
		if step.StepID == "" {
			step.StepID = uuid.NewString()
		}
	}
	p.Freeze()
	return nil
}

// GroupParallelSteps groups consecutive steps into execution batches.
// Steps with the same ParallelGroup are batched together for parallel execution.
// Non-parallelizable steps and steps without a group form individual batches.
func GroupParallelSteps(steps []ExecutionStep) [][]ExecutionStep {
	if len(steps) == 0 {
		return nil
	}

	var groups [][]ExecutionStep
	i := 0
	for i < len(steps) {
		s := steps[i]
		if s.Parallelizable && s.ParallelGroup != "" {
			// Collect all consecutive steps with the same group
			var batch []ExecutionStep
			group := s.ParallelGroup
			for i < len(steps) && steps[i].Parallelizable && steps[i].ParallelGroup == group {
				batch = append(batch, steps[i])
				i++
			}
			groups = append(groups, batch)
		} else {
			groups = append(groups, []ExecutionStep{s})
			i++
		}
	}
	return groups
}
