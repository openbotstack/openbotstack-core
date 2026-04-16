package execution_test

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/openbotstack/openbotstack-core/execution"
)

// --- ExecutionPlan.Validate ---

func TestExecutionPlanValidate_ValidPlan(t *testing.T) {
	plan := &execution.ExecutionPlan{
		AssistantID: "asst-1",
		Steps: []execution.ExecutionStep{
			{Name: "step-1", Type: execution.StepTypeSkill, Arguments: map[string]any{"key": "value"}},
		},
	}
	if err := plan.Validate(); err != nil {
		t.Fatalf("expected nil, got error: %v", err)
	}
}

func TestExecutionPlanValidate_Errors(t *testing.T) {
	tests := []struct {
		name    string
		plan    *execution.ExecutionPlan
		wantErr string
	}{
		{
			name:    "empty steps",
			plan:    &execution.ExecutionPlan{Steps: []execution.ExecutionStep{}},
			wantErr: "plan must have at least one step",
		},
		{
			name: "nil steps",
			plan: &execution.ExecutionPlan{Steps: nil},
			wantErr: "plan must have at least one step",
		},
		{
			name: "empty name",
			plan: &execution.ExecutionPlan{
				Steps: []execution.ExecutionStep{
					{Name: "", Type: execution.StepTypeSkill},
				},
			},
			wantErr: "step 0 has empty name",
		},
		{
			name: "invalid type",
			plan: &execution.ExecutionPlan{
				Steps: []execution.ExecutionStep{
					{Name: "bad-step", Type: execution.StepType("unknown")},
				},
			},
			wantErr: "step 0 has invalid type: unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.plan.Validate()
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tt.wantErr)
			}
			if err.Error() != tt.wantErr {
				t.Errorf("expected error %q, got %q", tt.wantErr, err.Error())
			}
		})
	}
}

// --- ExecutionStep.ArgumentsJSON ---

func TestExecutionStepArgumentsJSON(t *testing.T) {
	t.Run("with arguments", func(t *testing.T) {
		args := map[string]any{"foo": "bar", "num": float64(42)}
		step := &execution.ExecutionStep{Arguments: args}

		got, err := step.ArgumentsJSON()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Verify it round-trips back to the same keys/values
		var parsed map[string]any
		if err := json.Unmarshal(got, &parsed); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}
		if parsed["foo"] != "bar" {
			t.Errorf("expected foo=bar, got %v", parsed["foo"])
		}
		if parsed["num"] != float64(42) {
			t.Errorf("expected num=42, got %v", parsed["num"])
		}
	})

	t.Run("nil arguments returns empty object", func(t *testing.T) {
		step := &execution.ExecutionStep{Arguments: nil}

		got, err := step.ArgumentsJSON()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if string(got) != "{}" {
			t.Errorf("expected '{}', got %q", string(got))
		}
	})
}

// --- ExecutionContext concurrency ---

func TestExecutionContext_Concurrency(t *testing.T) {
	ctx := context.Background()
	ec := execution.NewExecutionContext(ctx, "req-1", "asst-1", "sess-1", "tenant-1", "user-1")

	const goroutines = 100
	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func(idx int) {
			defer wg.Done()
			ec.AddResult(execution.StepResult{
				StepName: "step",
				Type:     "skill",
				Output:   idx,
				Duration: time.Millisecond,
			})
		}(i)
	}
	wg.Wait()

	results := ec.Results()
	if len(results) != goroutines {
		t.Fatalf("expected %d results, got %d", goroutines, len(results))
	}

	// Verify all IDs are present (no lost writes)
	seen := make(map[int]bool)
	for _, r := range results {
		seen[r.Output.(int)] = true
	}
	for i := 0; i < goroutines; i++ {
		if !seen[i] {
			t.Errorf("missing result from goroutine %d", i)
		}
	}
}

// --- ExecutionStatus constants ---

func TestExecutionStatus_Constants(t *testing.T) {
	tests := []struct {
		name  string
		value execution.ExecutionStatus
		want  string
	}{
		{"success", execution.StatusSuccess, "success"},
		{"failed", execution.StatusFailed, "failed"},
		{"timeout", execution.StatusTimeout, "timeout"},
		{"canceled", execution.StatusCanceled, "canceled"},
		{"rejected", execution.StatusRejected, "rejected"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.value) != tt.want {
				t.Errorf("expected %q, got %q", tt.want, string(tt.value))
			}
		})
	}
}

// --- Sentinel errors exist ---

func TestSentinelErrors(t *testing.T) {
	errs := map[string]error{
		"ErrSkillNotLoaded":   execution.ErrSkillNotLoaded,
		"ErrExecutionTimeout": execution.ErrExecutionTimeout,
		"ErrResourceExhausted": execution.ErrResourceExhausted,
		"ErrPolicyRejected":   execution.ErrPolicyRejected,
	}
	for name, err := range errs {
		t.Run(name, func(t *testing.T) {
			if err == nil {
				t.Fatalf("%s is nil", name)
			}
		})
	}
}

// --- StepType constants ---

func TestStepType_Constants(t *testing.T) {
	if execution.StepTypeSkill != "skill" {
		t.Errorf("StepTypeSkill: expected 'skill', got %q", execution.StepTypeSkill)
	}
	if execution.StepTypeTool != "tool" {
		t.Errorf("StepTypeTool: expected 'tool', got %q", execution.StepTypeTool)
	}
}

// --- ExecutionContext field propagation ---

func TestNewExecutionContext_Fields(t *testing.T) {
	ctx := context.Background()
	ec := execution.NewExecutionContext(ctx, "r1", "a1", "s1", "t1", "u1")

	if ec.RequestID != "r1" {
		t.Errorf("RequestID: expected 'r1', got %q", ec.RequestID)
	}
	if ec.AssistantID != "a1" {
		t.Errorf("AssistantID: expected 'a1', got %q", ec.AssistantID)
	}
	if ec.SessionID != "s1" {
		t.Errorf("SessionID: expected 's1', got %q", ec.SessionID)
	}
	if ec.TenantID != "t1" {
		t.Errorf("TenantID: expected 't1', got %q", ec.TenantID)
	}
	if ec.UserID != "u1" {
		t.Errorf("UserID: expected 'u1', got %q", ec.UserID)
	}
	if ec.StartedAt.IsZero() {
		t.Error("StartedAt should not be zero")
	}
	if len(ec.Results()) != 0 {
		t.Error("new context should have zero results")
	}
}

// --- ExecutionPlan JSON serialization ---

func TestExecutionPlan_JSONRoundTrip(t *testing.T) {
	plan := &execution.ExecutionPlan{
		AssistantID: "asst-42",
		Steps: []execution.ExecutionStep{
			{Name: "step-a", Type: execution.StepTypeSkill, Arguments: map[string]any{"x": 1}},
			{Name: "step-b", Type: execution.StepTypeTool, Arguments: nil},
		},
		Reasoning: "test reasoning",
	}

	data, err := json.Marshal(plan)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded execution.ExecutionPlan
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if decoded.AssistantID != plan.AssistantID {
		t.Errorf("AssistantID mismatch: %q vs %q", decoded.AssistantID, plan.AssistantID)
	}
	if len(decoded.Steps) != 2 {
		t.Fatalf("expected 2 steps, got %d", len(decoded.Steps))
	}
	if decoded.Steps[0].Name != "step-a" || decoded.Steps[1].Name != "step-b" {
		t.Errorf("step names mismatch")
	}
	if decoded.Reasoning != plan.Reasoning {
		t.Errorf("Reasoning mismatch")
	}
}
