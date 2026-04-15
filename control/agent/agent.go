package agent

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/openbotstack/openbotstack-core/assistant"
	corecontext "github.com/openbotstack/openbotstack-core/context"
	csSkills "github.com/openbotstack/openbotstack-core/control/skills"
	"github.com/openbotstack/openbotstack-core/execution"
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
	ExecuteFromPlan(ctx context.Context, plan *ExecutionPlan, meta ExecutionMeta) (*execution.ExecutionResult, error)
}

// ExecutionMeta contains metadata for execution tracking.
type ExecutionMeta struct {
	TenantID    string
	UserID      string
	SessionID   string
	RequestID   string
	AssistantID string
}

// DefaultAgent is the standard Agent implementation.
type DefaultAgent struct {
	planner            Planner
	executor           PlanExecutor
	registry           SkillRegistry
	runtime            *assistant.AssistantRuntime
	conversationStore  ConversationStore
	maxHistoryMessages int
	contextAssembler   corecontext.ContextAssembler // optional pre-planning context enrichment
}

// NewDefaultAgent creates a new Agent with the given dependencies.
func NewDefaultAgent(planner Planner, executor PlanExecutor, registry SkillRegistry, runtime *assistant.AssistantRuntime) *DefaultAgent {
	return &DefaultAgent{
		planner:            planner,
		executor:           executor,
		registry:           registry,
		runtime:            runtime,
		maxHistoryMessages: 50,
	}
}

// SetConversationStore configures the conversation memory backend.
// If set, the agent loads history before planning and stores messages after execution.
func (a *DefaultAgent) SetConversationStore(store ConversationStore) {
	a.conversationStore = store
}

// SetMaxHistoryMessages sets the maximum number of recent messages to inject into the planner.
func (a *DefaultAgent) SetMaxHistoryMessages(n int) {
	a.maxHistoryMessages = n
}

// SetContextAssembler configures the context assembler for pre-planning enrichment.
// If set, the assembler enriches the conversation history with persona and memory context.
func (a *DefaultAgent) SetContextAssembler(ca corecontext.ContextAssembler) {
	a.contextAssembler = ca
}

// HandleMessage implements Agent.
func (a *DefaultAgent) HandleMessage(ctx context.Context, req MessageRequest) (*MessageResponse, error) {
	// Step 1: Gather available skills
	skillDescriptors, err := a.gatherSkillDescriptors()
	if err != nil {
		return nil, fmt.Errorf("agent: failed to gather skills: %w", err)
	}

	if len(skillDescriptors) == 0 {
		return nil, ErrNoSkillsAvailable
	}

	// Step 2: Load conversation history
	var conversationHistory []Message
	if a.conversationStore != nil && req.SessionID != "" {
		conversationHistory = a.loadHistory(ctx, req)
	}

	// Step 2.5: Enrich history via ContextAssembler (best-effort)
	if a.contextAssembler != nil {
		skillMsgs := agentMsgsToSkillMsgs(conversationHistory)
		assembled, err := a.contextAssembler.Assemble(ctx,
			corecontext.AssistantContext{
				ProfileID:       a.runtime.AssistantID,
				EnabledSkillIDs: skillIDsFromDescriptors(skillDescriptors),
			},
			corecontext.UserRequest{
				Message:        req.Message,
				ConversationID: req.SessionID,
				TenantID:       req.TenantID,
				UserID:         req.UserID,
			},
			skillMsgs,
		)
		if err != nil {
			slog.WarnContext(ctx, "agent: context assembly failed, using raw history",
				"error", err)
		} else if assembled != nil && len(assembled.Messages) > 0 {
			conversationHistory = skillMsgsToAgentMsgs(assembled.Messages)
		}
	}

	// Step 3: Plan via LLM
	planReq := PlanRequest{
		UserMessage:         req.Message,
		AvailableSkills:     skillDescriptors,
		ConversationHistory: conversationHistory,
	}

	plan, err := a.planner.Plan(ctx, a.runtime, planReq)
	if err != nil {
		return nil, fmt.Errorf("agent: planning failed: %w", err)
	}

	// Step 4: Validate plan
	if err := plan.Validate(); err != nil {
		return nil, fmt.Errorf("agent: invalid plan: %w", err)
	}

	// Step 5: Execute via Executor
	meta := ExecutionMeta{
		TenantID:  req.TenantID,
		UserID:    req.UserID,
		SessionID: req.SessionID,
	}

	result, err := a.executor.ExecuteFromPlan(ctx, plan, meta)
	if err != nil {
		resp := &MessageResponse{
			SessionID: req.SessionID,
			Message:   fmt.Sprintf("Error executing skill %s: %v", plan.SkillID, err),
			SkillUsed: plan.SkillID,
			Plan:      plan,
		}
		// Store messages even on error (best-effort)
		a.storeMessages(ctx, req, resp)
		return resp, err
	}

	// Step 6: Build response
	resp := &MessageResponse{
		SessionID: req.SessionID,
		Message:   string(result.Output),
		SkillUsed: plan.SkillID,
		Plan:      plan,
	}

	// Step 7: Store messages (best-effort, does not block response)
	a.storeMessages(ctx, req, resp)

	return resp, nil
}

// loadHistory retrieves conversation history + summary for the current session.
func (a *DefaultAgent) loadHistory(ctx context.Context, req MessageRequest) []Message {
	var history []Message

	// Load summary if available
	summary, err := a.conversationStore.GetSummary(ctx, req.TenantID, req.UserID, req.SessionID)
	if err == nil && summary != "" {
		history = append(history, Message{
			Role:    "system",
			Content: "Previous conversation summary:\n" + summary,
		})
	}

	// Load recent messages
	msgs, err := a.conversationStore.GetHistory(ctx, req.TenantID, req.UserID, req.SessionID, a.maxHistoryMessages)
	if err == nil && len(msgs) > 0 {
		history = append(history, msgs...)
	}

	return history
}

// storeMessages persists user message and assistant response (best-effort).
func (a *DefaultAgent) storeMessages(ctx context.Context, req MessageRequest, resp *MessageResponse) {
	if a.conversationStore == nil || req.SessionID == "" {
		return
	}

	now := time.Now().UTC().Format(time.RFC3339Nano)

	// Store user message
	if err := a.conversationStore.AppendMessage(ctx, SessionMessage{
		TenantID:  req.TenantID,
		UserID:    req.UserID,
		SessionID: req.SessionID,
		Role:      "user",
		Content:   req.Message,
		Timestamp: now,
	}); err != nil {
		slog.WarnContext(ctx, "agent: failed to store user message",
			"session_id", req.SessionID, "error", err)
	}

	// Store assistant response
	if err := a.conversationStore.AppendMessage(ctx, SessionMessage{
		TenantID:  req.TenantID,
		UserID:    req.UserID,
		SessionID: req.SessionID,
		Role:      "assistant",
		Content:   resp.Message,
		Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
	}); err != nil {
		slog.WarnContext(ctx, "agent: failed to store assistant message",
			"session_id", req.SessionID, "error", err)
	}
}

// gatherSkillDescriptors converts registered skills to descriptors for the planner.
func (a *DefaultAgent) gatherSkillDescriptors() ([]SkillDescriptor, error) {
	skillIDs := a.registry.List()
	descriptors := make([]SkillDescriptor, 0, len(skillIDs))

	for _, id := range skillIDs {
		s, err := a.registry.Get(id)
		if err != nil {
			continue // skip unavailable skills
		}
		descriptors = append(descriptors, SkillDescriptorFromSkill(s))
	}

	return descriptors, nil
}

// skillIDsFromDescriptors extracts skill IDs from descriptors.
func skillIDsFromDescriptors(descs []SkillDescriptor) []string {
	ids := make([]string, 0, len(descs))
	for _, d := range descs {
		ids = append(ids, d.ID)
	}
	return ids
}

// agentMsgsToSkillMsgs converts agent.Message slice to control/skills.Message slice.
func agentMsgsToSkillMsgs(msgs []Message) []csSkills.Message {
	result := make([]csSkills.Message, 0, len(msgs))
	for _, m := range msgs {
		result = append(result, csSkills.Message{
			Role:    m.Role,
			Content: m.Content,
		})
	}
	return result
}

// skillMsgsToAgentMsgs converts control/skills.Message slice to agent.Message slice.
func skillMsgsToAgentMsgs(msgs []csSkills.Message) []Message {
	result := make([]Message, 0, len(msgs))
	for _, m := range msgs {
		result = append(result, Message{
			Role:    m.Role,
			Content: m.Content,
		})
	}
	return result
}
