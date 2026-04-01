package assistants

import (
	"encoding/json"
	"sync"
	"time"
)

// DefaultStateMachine is the default implementation of AgentStateMachine.
type DefaultStateMachine struct {
	mu              sync.RWMutex
	currentState    AgentState
	reflectionCount int
	maxReflections  int
	history         []StateTransition
}

// validTransitions defines the allowed state transitions.
var validTransitions = map[AgentState][]AgentState{
	StateIdle:       {StatePlanning, StateFailed},
	StatePlanning:   {StateExecuting, StateFailed},
	StateExecuting:  {StateReflecting, StateFailed},
	StateReflecting: {StateExecuting, StateFinalizing, StateFailed},
	StateFinalizing: {StateCompleted, StateFailed},
	StateCompleted:  {},
	StateFailed:     {},
}

// NewStateMachine creates a new state machine with the given max reflections.
func NewStateMachine(maxReflections int) *DefaultStateMachine {
	return &DefaultStateMachine{
		currentState:   StateIdle,
		maxReflections: maxReflections,
		history:        make([]StateTransition, 0),
	}
}

// CurrentState returns the current execution state.
func (sm *DefaultStateMachine) CurrentState() AgentState {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.currentState
}

// CanTransitionTo returns true if the transition is valid.
func (sm *DefaultStateMachine) CanTransitionTo(next AgentState) bool {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	allowed, ok := validTransitions[sm.currentState]
	if !ok {
		return false
	}

	for _, s := range allowed {
		if s == next {
			return true
		}
	}
	return false
}

// TransitionTo moves to the next state.
func (sm *DefaultStateMachine) TransitionTo(next AgentState, reason string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// Check if transition is allowed
	allowed := false
	for _, s := range validTransitions[sm.currentState] {
		if s == next {
			allowed = true
			break
		}
	}

	if !allowed {
		return ErrInvalidTransition
	}

	// Check reflection limit
	if next == StateReflecting {
		if sm.reflectionCount >= sm.maxReflections {
			return ErrMaxReflectionsExceeded
		}
		sm.reflectionCount++
	}

	// Record transition
	transition := StateTransition{
		From:      sm.currentState,
		To:        next,
		Reason:    reason,
		Timestamp: time.Now(),
	}
	sm.history = append(sm.history, transition)

	sm.currentState = next
	return nil
}

// ReflectionCount returns how many reflection cycles have occurred.
func (sm *DefaultStateMachine) ReflectionCount() int {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.reflectionCount
}

// MaxReflections returns the configured upper bound on reflections.
func (sm *DefaultStateMachine) MaxReflections() int {
	return sm.maxReflections
}

// Serialize returns the full state for persistence.
func (sm *DefaultStateMachine) Serialize() ([]byte, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	data := struct {
		CurrentState    AgentState        `json:"current_state"`
		ReflectionCount int               `json:"reflection_count"`
		MaxReflections  int               `json:"max_reflections"`
		History         []StateTransition `json:"history"`
	}{
		CurrentState:    sm.currentState,
		ReflectionCount: sm.reflectionCount,
		MaxReflections:  sm.maxReflections,
		History:         sm.history,
	}

	return json.Marshal(data)
}

// History returns the ordered list of state transitions.
func (sm *DefaultStateMachine) History() []StateTransition {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	// Return a copy to prevent mutation
	result := make([]StateTransition, len(sm.history))
	copy(result, sm.history)
	return result
}
