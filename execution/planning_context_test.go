package execution_test

import (
	"context"
	"testing"

	"github.com/openbotstack/openbotstack-core/execution"
	"github.com/openbotstack/openbotstack-core/planning"
)

// TestPlannerContext_TypeSafe verifies that ExecutionContext stores and retrieves
// PlannerContext as a concrete *planning.PlannerContext, not an untyped any.
// This is the TDD RED test: it will fail to compile until execution_context.go
// is updated to use the properly typed field.
func TestPlannerContext_TypeSafe(t *testing.T) {
	ec := execution.NewExecutionContext(context.Background(), "req-1", "asst-1", "sess-1", "tenant-1", "user-1")

	// Set a properly-typed PlannerContext
	pc := &planning.PlannerContext{
		AssistantID: "asst-1",
		UserRequest: "hello",
	}
	ec.SetPlannerContext(pc)

	// Retrieve and verify we get *planning.PlannerContext directly — no type assertion needed.
	got := ec.PlannerContext()
	if got == nil {
		t.Fatal("PlannerContext() returned nil after SetPlannerContext")
	}
	if got.AssistantID != "asst-1" {
		t.Errorf("AssistantID = %q, want %q", got.AssistantID, "asst-1")
	}
	if got.UserRequest != "hello" {
		t.Errorf("UserRequest = %q, want %q", got.UserRequest, "hello")
	}
}

// TestPlannerContext_NilByDefault verifies that PlannerContext returns nil
// when no context has been set.
func TestPlannerContext_NilByDefault(t *testing.T) {
	ec := execution.NewExecutionContext(context.Background(), "req-1", "asst-1", "sess-1", "tenant-1", "user-1")

	got := ec.PlannerContext()
	if got != nil {
		t.Errorf("PlannerContext() = %v, want nil", got)
	}
}

// TestTurnToolResult_BackwardCompatible verifies that execution.TurnToolResult
// is still accessible (via type alias) and has the same fields.
func TestTurnToolResult_BackwardCompatible(t *testing.T) {
	tr := execution.TurnToolResult{
		StepName: "fetch_data",
		StepType: "tool",
		Success:  true,
		Summary:  "fetched successfully",
		Output:   `{"key": "value"}`,
	}

	if tr.StepName != "fetch_data" {
		t.Errorf("StepName = %q, want %q", tr.StepName, "fetch_data")
	}
	if !tr.Success {
		t.Error("Success = false, want true")
	}
}
