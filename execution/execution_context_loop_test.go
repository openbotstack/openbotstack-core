package execution

import (
	"context"
	"testing"
)

func TestExecutionContext_LoopFieldsZeroValue(t *testing.T) {
	// Verify that new loop fields have zero values that don't affect existing code.
	ctx := NewExecutionContext(context.Background(), "req1", "asst1", "sess1", "tenant1", "user1")

	if ctx.LoopMode != "" {
		t.Errorf("LoopMode should be empty by default, got %q", ctx.LoopMode)
	}
	if ctx.CurrentTaskIndex != 0 {
		t.Errorf("CurrentTaskIndex should be 0 by default, got %d", ctx.CurrentTaskIndex)
	}
	if ctx.CurrentTurn != 0 {
		t.Errorf("CurrentTurn should be 0 by default, got %d", ctx.CurrentTurn)
	}
}

func TestExecutionContext_LoopFieldsSettable(t *testing.T) {
	ctx := NewExecutionContext(context.Background(), "req1", "asst1", "sess1", "tenant1", "user1")

	ctx.LoopMode = "dual_loop"
	ctx.CurrentTaskIndex = 3
	ctx.CurrentTurn = 7

	if ctx.LoopMode != "dual_loop" {
		t.Errorf("LoopMode = %q, want %q", ctx.LoopMode, "dual_loop")
	}
	if ctx.CurrentTaskIndex != 3 {
		t.Errorf("CurrentTaskIndex = %d, want %d", ctx.CurrentTaskIndex, 3)
	}
	if ctx.CurrentTurn != 7 {
		t.Errorf("CurrentTurn = %d, want %d", ctx.CurrentTurn, 7)
	}
}

func TestExecutionContext_ExistingFieldsStillWork(t *testing.T) {
	// Ensure existing fields are not affected by the new additions.
	ctx := NewExecutionContext(context.Background(), "req1", "asst1", "sess1", "tenant1", "user1")

	if ctx.RequestID != "req1" {
		t.Errorf("RequestID = %q, want %q", ctx.RequestID, "req1")
	}
	if ctx.AssistantID != "asst1" {
		t.Errorf("AssistantID = %q, want %q", ctx.AssistantID, "asst1")
	}
	if ctx.SessionID != "sess1" {
		t.Errorf("SessionID = %q, want %q", ctx.SessionID, "sess1")
	}
	if ctx.TenantID != "tenant1" {
		t.Errorf("TenantID = %q, want %q", ctx.TenantID, "tenant1")
	}
	if ctx.UserID != "user1" {
		t.Errorf("UserID = %q, want %q", ctx.UserID, "user1")
	}

	// AddResult still works
	ctx.AddResult(StepResult{StepName: "test", Type: "tool"})
	results := ctx.Results()
	if len(results) != 1 {
		t.Fatalf("Results() returned %d items, want 1", len(results))
	}
	if results[0].StepName != "test" {
		t.Errorf("StepName = %q, want %q", results[0].StepName, "test")
	}
}
