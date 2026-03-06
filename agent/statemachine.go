package agent

// AgentStateMachine manages the deterministic lifecycle of a single request.
//
// Key invariants:
//   - Exactly one active state at any time
//   - Transitions are explicit and audited
//   - Reflection is bounded (max iterations)
//   - All state is serializable for persistence
type AgentStateMachine interface {
	// CurrentState returns the current execution state.
	CurrentState() AgentState

	// CanTransitionTo returns true if the transition is valid.
	CanTransitionTo(next AgentState) bool

	// TransitionTo moves to the next state.
	// Returns error if transition is invalid.
	// Emits audit event on success.
	TransitionTo(next AgentState, reason string) error

	// ReflectionCount returns how many reflection cycles have occurred.
	ReflectionCount() int

	// MaxReflections returns the configured upper bound on reflections.
	MaxReflections() int

	// Serialize returns the full state for persistence.
	Serialize() ([]byte, error)

	// History returns the ordered list of state transitions.
	History() []StateTransition
}
