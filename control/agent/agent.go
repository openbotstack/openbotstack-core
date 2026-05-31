package agent

import (
	"context"

	aitypes "github.com/openbotstack/openbotstack-core/ai/types"
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

// ExecutionMeta contains metadata for execution tracking.
type ExecutionMeta struct {
	TenantID    string
	UserID      string
	SessionID   string
	RequestID   string
	AssistantID string
}

// MessagesToSkillMsgs converts agent.Message slice to ai/types.Message slice.
// The aitypes.Message type includes a Name field for tool messages; conversion drops names.
func MessagesToSkillMsgs(msgs []Message) []aitypes.Message {
	result := make([]aitypes.Message, 0, len(msgs))
	for _, m := range msgs {
		result = append(result, aitypes.Message{
			Role:    m.Role,
			Content: m.Content,
		})
	}
	return result
}

// SkillMsgsToMessages converts ai/types.Message slice to agent.Message slice.
func SkillMsgsToMessages(msgs []aitypes.Message) []Message {
	result := make([]Message, 0, len(msgs))
	for _, m := range msgs {
		result = append(result, Message{
			Role:    m.Role,
			Content: m.Content,
		})
	}
	return result
}
