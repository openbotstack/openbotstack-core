package agent

import (
	"context"

	"github.com/openbotstack/openbotstack-core/memory/abstraction"
)

// Agent orchestrates the planning and execution of skills.
//
// The Agent lifecycle:
//  1. Receives MessageRequest from Router
//  2. Loads conversation history from ConversationStore
//  3. Gathers available skills from registry
//  4. Delegates to Planner for skill selection (LLM call with history context)
//  5. Receives ExecutionPlan from Planner
//  6. Forwards plan to Executor
//  7. Stores user message and assistant response
//  8. Returns MessageResponse to Router
type Agent interface {
	// HandleMessage processes a user message and returns a response.
	HandleMessage(ctx context.Context, req MessageRequest) (*MessageResponse, error)
}

// MemoryConfigurable is an optional interface that agents may implement to
// receive memory-related configuration after construction. This decouples
// the ServerBuilder from concrete agent types.
type MemoryConfigurable interface {
	SetMaxHistoryMessages(max int)
	SetMemoryManager(mgr abstraction.MemoryManager)
}

// ExecutionMeta contains metadata for execution tracking.
type ExecutionMeta struct {
	TenantID    string
	UserID      string
	SessionID   string
	RequestID   string
	AssistantID string
}
