// Package agent defines the agent state machine and lifecycle for OpenBotStack.
//
// The agent state machine manages the deterministic lifecycle of a single request,
// enforcing bounded reflection and explicit state transitions.
package agent

import "time"

// AgentState represents the current phase of agent execution.
type AgentState string

const (
	// StateIdle indicates the agent is awaiting a request.
	StateIdle AgentState = "idle"

	// StatePlanning indicates the agent is decomposing a goal into steps.
	StatePlanning AgentState = "planning"

	// StateExecuting indicates the agent is delegating to runtime.
	StateExecuting AgentState = "executing"

	// StateReflecting indicates the agent is evaluating execution results.
	StateReflecting AgentState = "reflecting"

	// StateFinalizing indicates the agent is preparing the response.
	StateFinalizing AgentState = "finalizing"

	// StateCompleted indicates execution finished successfully.
	StateCompleted AgentState = "completed"

	// StateFailed indicates an unrecoverable error occurred.
	StateFailed AgentState = "failed"
)

// StateTransition records a single state change.
type StateTransition struct {
	From      AgentState
	To        AgentState
	Reason    string
	Timestamp time.Time
}
