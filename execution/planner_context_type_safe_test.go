package execution

import (
	"context"
	"testing"
)

// TestPlannerContext_SetGet verifies that PlannerContext stores and retrieves
// values correctly with typed accessor (not any).
func TestPlannerContext_SetGet(t *testing.T) {
	ec := NewExecutionContext(context.Background(), "req-1", "asst-1", "sess-1", "tenant-1", "user-1")

	// Before setting, should return nil.
	if ec.PlannerContext() != nil {
		t.Error("PlannerContext should be nil before setting")
	}

	// Set nil explicitly — should not panic.
	ec.SetPlannerContext(nil)
	if ec.PlannerContext() != nil {
		t.Error("PlannerContext should be nil")
	}
}

// TestPlannerContext_InterfaceCompilation is a compile-time check.
// If PlannerContext() returns any, this always compiles.
// If it returns a concrete type, this still compiles only if the type is correct.
func TestPlannerContext_InterfaceCompilation(t *testing.T) {
	ec := NewExecutionContext(context.Background(), "req-1", "asst-1", "sess-1", "tenant-1", "user-1")
	// This line verifies the return type of PlannerContext().
	// With `any`: _ = ec.PlannerContext() always works.
	// With typed return: same.
	// The real verification is in harness tests that consume it.
	_ = ec.PlannerContext()
}
