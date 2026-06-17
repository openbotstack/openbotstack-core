package execution

import (
	"encoding/json"
	"fmt"

	"github.com/google/uuid"

	aitypes "github.com/openbotstack/openbotstack-core/ai/types"
	"github.com/openbotstack/openbotstack-core/execution/template"
	"github.com/openbotstack/openbotstack-core/planning"
)

// TurnToolResult is a type alias for planning.TurnToolResult.
// Kept here for backward compatibility — all existing code referencing
// execution.TurnToolResult continues to compile without changes.
type TurnToolResult = planning.TurnToolResult

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

	// OutputSchema is the JSON Schema declaring this step's expected output
	// structure (ADR-036 Phase 1). Populated by the planner from the skill's
	// manifest so the harness can run deterministic Schema Verify on
	// StepResult.Output without consulting a registry at execution time — the
	// plan is self-contained. nil = no schema declared → Verify is a no-op
	// (backward compatible; builtin tools and schema-less skills skip Verify).
	OutputSchema *aitypes.JSONSchema `json:"output_schema,omitempty"`

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
// Returns an error if any template reference cannot be resolved — the caller MUST
// fail the step rather than dispatch it with a literal {{...}} string.
func (s *ExecutionStep) ResolveArguments(results map[string]any) error {
	if s.Arguments == nil || len(results) == 0 {
		return nil
	}
	for key, val := range s.Arguments {
		strVal, ok := val.(string)
		if !ok {
			continue
		}
		resolved, err := template.Resolve(strVal, results)
		if err != nil {
			return fmt.Errorf("arg %q: %w", key, err)
		}
		s.Arguments[key] = resolved
	}
	return nil
}

// Prepare returns a copy of the step ready for dispatch: it clones s (so the
// frozen plan's step is never mutated), coerces numeric-string arguments, and
// resolves {{...}} templates against results. This is the single place that
// owns "get a step ready to run" — callers should use Prepare rather than
// Clone+Coerce+Resolve separately, which historically led to steps being
// resolved redundantly (and inconsistently) across the dispatch path.
//
// On a resolution error the original step is left untouched and the error is
// returned.
func (s *ExecutionStep) Prepare(results map[string]any) (*ExecutionStep, error) {
	clone := s.Clone()
	clone.CoerceStringNumbers()
	if err := clone.ResolveArguments(results); err != nil {
		return nil, fmt.Errorf("step %q: %w", clone.Name, err)
	}
	return clone, nil
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
	// ID is auto-generated during Validate() if empty.
	// Used for plan lineage tracking in replanning.
	ID string `json:"id,omitempty"`

	// ParentID references the plan this one replaces (set during replanning).
	ParentID string `json:"parent_id,omitempty"`

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

	// Auto-generate plan ID if empty.
	if p.ID == "" {
		p.ID = uuid.NewString()
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
