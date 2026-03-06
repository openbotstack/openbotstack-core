package agent_test

import (
	"testing"

	"github.com/openbotstack/openbotstack-core/agent"
)

func TestNewStateMachine(t *testing.T) {
	sm := agent.NewStateMachine(3)
	if sm == nil {
		t.Fatal("NewStateMachine returned nil")
	}
	if sm.CurrentState() != agent.StateIdle {
		t.Errorf("Expected initial state Idle, got %s", sm.CurrentState())
	}
}

func TestMaxReflections(t *testing.T) {
	sm := agent.NewStateMachine(5)
	if sm.MaxReflections() != 5 {
		t.Errorf("Expected max reflections 5, got %d", sm.MaxReflections())
	}
}

func TestValidTransitions(t *testing.T) {
	tests := []struct {
		name  string
		from  agent.AgentState
		to    agent.AgentState
		valid bool
	}{
		{"idle to planning", agent.StateIdle, agent.StatePlanning, true},
		{"planning to executing", agent.StatePlanning, agent.StateExecuting, true},
		{"executing to reflecting", agent.StateExecuting, agent.StateReflecting, true},
		{"reflecting to executing", agent.StateReflecting, agent.StateExecuting, true},
		{"reflecting to finalizing", agent.StateReflecting, agent.StateFinalizing, true},
		{"finalizing to completed", agent.StateFinalizing, agent.StateCompleted, true},
		{"any to failed", agent.StateExecuting, agent.StateFailed, true},
		// Invalid transitions
		{"idle to executing", agent.StateIdle, agent.StateExecuting, false},
		{"completed to planning", agent.StateCompleted, agent.StatePlanning, false},
		{"planning to completed", agent.StatePlanning, agent.StateCompleted, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sm := agent.NewStateMachine(3)
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
	sm := agent.NewStateMachine(3)

	err := sm.TransitionTo(agent.StatePlanning, "starting")
	if err != nil {
		t.Fatalf("TransitionTo failed: %v", err)
	}

	if sm.CurrentState() != agent.StatePlanning {
		t.Errorf("Expected state Planning, got %s", sm.CurrentState())
	}
}

func TestInvalidTransitionReturnsError(t *testing.T) {
	sm := agent.NewStateMachine(3)

	err := sm.TransitionTo(agent.StateCompleted, "skip")
	if err == nil {
		t.Error("Expected error for invalid transition, got nil")
	}
	if err != agent.ErrInvalidTransition {
		t.Errorf("Expected ErrInvalidTransition, got %v", err)
	}
}

func TestReflectionCounting(t *testing.T) {
	sm := agent.NewStateMachine(3)

	// Navigate: idle -> planning -> executing -> reflecting
	_ = sm.TransitionTo(agent.StatePlanning, "plan")
	_ = sm.TransitionTo(agent.StateExecuting, "execute")
	_ = sm.TransitionTo(agent.StateReflecting, "reflect1")

	if sm.ReflectionCount() != 1 {
		t.Errorf("Expected reflection count 1, got %d", sm.ReflectionCount())
	}

	// Reflect again: reflecting -> executing -> reflecting
	_ = sm.TransitionTo(agent.StateExecuting, "retry")
	_ = sm.TransitionTo(agent.StateReflecting, "reflect2")

	if sm.ReflectionCount() != 2 {
		t.Errorf("Expected reflection count 2, got %d", sm.ReflectionCount())
	}
}

func TestMaxReflectionsExceeded(t *testing.T) {
	sm := agent.NewStateMachine(2) // max 2 reflections

	// Navigate to reflecting twice
	_ = sm.TransitionTo(agent.StatePlanning, "plan")
	_ = sm.TransitionTo(agent.StateExecuting, "exec1")
	_ = sm.TransitionTo(agent.StateReflecting, "reflect1") // count=1
	_ = sm.TransitionTo(agent.StateExecuting, "exec2")
	_ = sm.TransitionTo(agent.StateReflecting, "reflect2") // count=2

	// Third attempt should fail
	_ = sm.TransitionTo(agent.StateExecuting, "exec3")
	err := sm.TransitionTo(agent.StateReflecting, "reflect3")

	if err != agent.ErrMaxReflectionsExceeded {
		t.Errorf("Expected ErrMaxReflectionsExceeded, got %v", err)
	}
}

func TestHistoryTracking(t *testing.T) {
	sm := agent.NewStateMachine(3)

	_ = sm.TransitionTo(agent.StatePlanning, "start")
	_ = sm.TransitionTo(agent.StateExecuting, "run")

	history := sm.History()
	if len(history) != 2 {
		t.Fatalf("Expected 2 history entries, got %d", len(history))
	}

	if history[0].From != agent.StateIdle || history[0].To != agent.StatePlanning {
		t.Errorf("First transition incorrect: %+v", history[0])
	}
	if history[1].From != agent.StatePlanning || history[1].To != agent.StateExecuting {
		t.Errorf("Second transition incorrect: %+v", history[1])
	}
}

func TestSerialize(t *testing.T) {
	sm := agent.NewStateMachine(3)
	_ = sm.TransitionTo(agent.StatePlanning, "start")

	data, err := sm.Serialize()
	if err != nil {
		t.Fatalf("Serialize failed: %v", err)
	}
	if len(data) == 0 {
		t.Error("Serialize returned empty data")
	}
}

// Helper to navigate to a target state
func navigateToState(sm agent.AgentStateMachine, target agent.AgentState) error {
	path := map[agent.AgentState][]agent.AgentState{
		agent.StateIdle:       {},
		agent.StatePlanning:   {agent.StatePlanning},
		agent.StateExecuting:  {agent.StatePlanning, agent.StateExecuting},
		agent.StateReflecting: {agent.StatePlanning, agent.StateExecuting, agent.StateReflecting},
		agent.StateFinalizing: {agent.StatePlanning, agent.StateExecuting, agent.StateReflecting, agent.StateFinalizing},
		agent.StateCompleted:  {agent.StatePlanning, agent.StateExecuting, agent.StateReflecting, agent.StateFinalizing, agent.StateCompleted},
		agent.StateFailed:     {agent.StatePlanning, agent.StateFailed},
	}

	for _, next := range path[target] {
		if err := sm.TransitionTo(next, "nav"); err != nil {
			return err
		}
	}
	return nil
}
