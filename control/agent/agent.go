package agent

import (
	"context"

	"github.com/openbotstack/openbotstack-core/assistant"
	corecontext "github.com/openbotstack/openbotstack-core/context"
	csSkills "github.com/openbotstack/openbotstack-core/control/skills"
	"github.com/openbotstack/openbotstack-core/execution"
	"github.com/openbotstack/openbotstack-core/planner"
	"github.com/openbotstack/openbotstack-core/registry/skills"
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

// SkillRegistry provides access to available skills.
type SkillRegistry interface {
	// List returns all registered skill IDs.
	List() []string

	// Get retrieves a skill by ID.
	Get(id string) (skills.Skill, error)
}

// PlanExecutor executes validated execution plans.
type PlanExecutor interface {
	// ExecuteFromPlan runs a skill based on the execution plan.
	ExecuteFromPlan(ctx context.Context, plan *execution.ExecutionPlan, meta ExecutionMeta) (*execution.ExecutionResult, error)
}

// ExecutionMeta contains metadata for execution tracking.
type ExecutionMeta struct {
	TenantID    string
	UserID      string
	SessionID   string
	RequestID   string
	AssistantID string
}

// AgentConfig holds all dependencies for constructing an agent.
// Required fields must be non-nil; optional fields may be nil.
type AgentConfig struct {
	Planner   planner.ExecutionPlanner
	Executor  PlanExecutor
	Registry  SkillRegistry
	Runtime   *assistant.AssistantRuntime

	// ContextAssembler enriches conversation history via memory retrieval.
	// Optional — nil = no enrichment.
	ContextAssembler corecontext.ContextAssembler

	MaxHistoryMessages int // defaults to 50 if zero
}


// MessagesToSkillMsgs converts agent.Message slice to control/skills.Message slice.
// The skills.Message type includes a Name field for tool messages; conversion drops names.
func MessagesToSkillMsgs(msgs []Message) []csSkills.Message {
	result := make([]csSkills.Message, 0, len(msgs))
	for _, m := range msgs {
		result = append(result, csSkills.Message{
			Role:    m.Role,
			Content: m.Content,
		})
	}
	return result
}

// SkillMsgsToMessages converts control/skills.Message slice to agent.Message slice.
func SkillMsgsToMessages(msgs []csSkills.Message) []Message {
	result := make([]Message, 0, len(msgs))
	for _, m := range msgs {
		result = append(result, Message{
			Role:    m.Role,
			Content: m.Content,
		})
	}
	return result
}
