package assistants_test

import (
	"testing"

	"github.com/openbotstack/openbotstack-core/control/assistants"
)

func TestNewStateMachine(t *testing.T) {
	sm := assistants.NewStateMachine(3)
	if sm == nil {
		t.Fatal("NewStateMachine returned nil")
	}
	if sm.CurrentState() != assistants.StateIdle {
		t.Errorf("Expected initial state Idle, got %s", sm.CurrentState())
	}
}

func TestMaxReflections(t *testing.T) {
	sm := assistants.NewStateMachine(5)
	if sm.MaxReflections() != 5 {
		t.Errorf("Expected max reflections 5, got %d", sm.MaxReflections())
	}
}

func TestValidTransitions(t *testing.T) {
	tests := []struct {
		name  string
		from  assistants.AgentState
		to    assistants.AgentState
		valid bool
	}{
		{"idle to planning", assistants.StateIdle, assistants.StatePlanning, true},
		{"planning to executing", assistants.StatePlanning, assistants.StateExecuting, true},
		{"executing to reflecting", assistants.StateExecuting, assistants.StateReflecting, true},
		{"reflecting to executing", assistants.StateReflecting, assistants.StateExecuting, true},
		{"reflecting to finalizing", assistants.StateReflecting, assistants.StateFinalizing, true},
		{"finalizing to completed", assistants.StateFinalizing, assistants.StateCompleted, true},
		{"any to failed", assistants.StateExecuting, assistants.StateFailed, true},
		// Invalid transitions
		{"idle to executing", assistants.StateIdle, assistants.StateExecuting, false},
		{"completed to planning", assistants.StateCompleted, assistants.StatePlanning, false},
		{"planning to completed", assistants.StatePlanning, assistants.StateCompleted, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sm := assistants.NewStateMachine(3)
			// Navigate to the 'from' state
			if err := navigateToState(sm, tt.from); err != nil {
				t.Skipf("Cannot navigate to state %s: %v", tt.from, err)
			}

			valid := sm.CanTransitionTo(tt.to)
			if valid != tt.valid {
				t.Errorf("CanTransitionTo(%s) = %v, want %v", tt.to, valid, tt.valid)
			}
		})
	}
}

func TestTransitionToUpdatesState(t *testing.T) {
	sm := assistants.NewStateMachine(3)

	err := sm.TransitionTo(assistants.StatePlanning, "starting")
	if err != nil {
		t.Fatalf("TransitionTo failed: %v", err)
	}

	if sm.CurrentState() != assistants.StatePlanning {
		t.Errorf("Expected state Planning, got %s", sm.CurrentState())
	}
}

func TestInvalidTransitionReturnsError(t *testing.T) {
	sm := assistants.NewStateMachine(3)

	err := sm.TransitionTo(assistants.StateCompleted, "skip")
	if err == nil {
		t.Error("Expected error for invalid transition, got nil")
	}
	if err != assistants.ErrInvalidTransition {
		t.Errorf("Expected ErrInvalidTransition, got %v", err)
	}
}

func TestReflectionCounting(t *testing.T) {
	sm := assistants.NewStateMachine(3)

	// Navigate: idle -> planning -> executing -> reflecting
	_ = sm.TransitionTo(assistants.StatePlanning, "plan")
	_ = sm.TransitionTo(assistants.StateExecuting, "execute")
	_ = sm.TransitionTo(assistants.StateReflecting, "reflect1")

	if sm.ReflectionCount() != 1 {
		t.Errorf("Expected reflection count 1, got %d", sm.ReflectionCount())
	}

	// Reflect again: reflecting -> executing -> reflecting
	_ = sm.TransitionTo(assistants.StateExecuting, "retry")
	_ = sm.TransitionTo(assistants.StateReflecting, "reflect2")

	if sm.ReflectionCount() != 2 {
		t.Errorf("Expected reflection count 2, got %d", sm.ReflectionCount())
	}
}

func TestMaxReflectionsExceeded(t *testing.T) {
	sm := assistants.NewStateMachine(2) // max 2 reflections

	// Navigate to reflecting twice
	_ = sm.TransitionTo(assistants.StatePlanning, "plan")
	_ = sm.TransitionTo(assistants.StateExecuting, "exec1")
	_ = sm.TransitionTo(assistants.StateReflecting, "reflect1") // count=1
	_ = sm.TransitionTo(assistants.StateExecuting, "exec2")
	_ = sm.TransitionTo(assistants.StateReflecting, "reflect2") // count=2

	// Third attempt should fail
	_ = sm.TransitionTo(assistants.StateExecuting, "exec3")
	err := sm.TransitionTo(assistants.StateReflecting, "reflect3")

	if err != assistants.ErrMaxReflectionsExceeded {
		t.Errorf("Expected ErrMaxReflectionsExceeded, got %v", err)
	}
}

func TestHistoryTracking(t *testing.T) {
	sm := assistants.NewStateMachine(3)

	_ = sm.TransitionTo(assistants.StatePlanning, "start")
	_ = sm.TransitionTo(assistants.StateExecuting, "run")

	history := sm.History()
	if len(history) != 2 {
		t.Fatalf("Expected 2 history entries, got %d", len(history))
	}

	if history[0].From != assistants.StateIdle || history[0].To != assistants.StatePlanning {
		t.Errorf("First transition incorrect: %+v", history[0])
	}
	if history[1].From != assistants.StatePlanning || history[1].To != assistants.StateExecuting {
		t.Errorf("Second transition incorrect: %+v", history[1])
	}
}

func TestSerialize(t *testing.T) {
	sm := assistants.NewStateMachine(3)
	_ = sm.TransitionTo(assistants.StatePlanning, "start")

	data, err := sm.Serialize()
	if err != nil {
		t.Fatalf("Serialize failed: %v", err)
	}
	if len(data) == 0 {
		t.Error("Serialize returned empty data")
	}
}

// Helper to navigate to a target state
func navigateToState(sm assistants.AgentStateMachine, target assistants.AgentState) error {
	path := map[assistants.AgentState][]assistants.AgentState{
		assistants.StateIdle:       {},
		assistants.StatePlanning:   {assistants.StatePlanning},
		assistants.StateExecuting:  {assistants.StatePlanning, assistants.StateExecuting},
		assistants.StateReflecting: {assistants.StatePlanning, assistants.StateExecuting, assistants.StateReflecting},
		assistants.StateFinalizing: {assistants.StatePlanning, assistants.StateExecuting, assistants.StateReflecting, assistants.StateFinalizing},
		assistants.StateCompleted:  {assistants.StatePlanning, assistants.StateExecuting, assistants.StateReflecting, assistants.StateFinalizing, assistants.StateCompleted},
		assistants.StateFailed:     {assistants.StatePlanning, assistants.StateFailed},
	}

	for _, next := range path[target] {
		if err := sm.TransitionTo(next, "nav"); err != nil {
			return err
		}
	}
	return nil
}
