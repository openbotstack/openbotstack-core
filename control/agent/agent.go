package agent

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/openbotstack/openbotstack-core/assistant"
	"github.com/openbotstack/openbotstack-core/audit"
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

// AgentConfig holds all dependencies for constructing a DefaultAgent.
// Required fields must be non-nil; optional fields may be nil.
type AgentConfig struct {
	Planner   planner.ExecutionPlanner
	Executor  PlanExecutor
	Registry  SkillRegistry
	Runtime   *assistant.AssistantRuntime

	// Optional dependencies (nil = feature disabled)
	ConversationStore ConversationStore
	ContextAssembler  corecontext.ContextAssembler
	AuditEmitter      *audit.AuditEmitter
	MaxHistoryMessages int // defaults to 50 if zero
}

// DefaultAgent is the standard Agent implementation.
type DefaultAgent struct {
	executionPlanner   planner.ExecutionPlanner
	executor           PlanExecutor
	registry           SkillRegistry
	runtime            *assistant.AssistantRuntime
	conversationStore  ConversationStore
	maxHistoryMessages int
	contextAssembler   corecontext.ContextAssembler
	auditEmitter       *audit.AuditEmitter
}

// NewDefaultAgent creates a new Agent from an AgentConfig.
func NewDefaultAgent(cfg AgentConfig) *DefaultAgent {
	maxHist := cfg.MaxHistoryMessages
	if maxHist <= 0 {
		maxHist = 50
	}
	return &DefaultAgent{
		executionPlanner:  cfg.Planner,
		executor:          cfg.Executor,
		registry:          cfg.Registry,
		runtime:           cfg.Runtime,
		conversationStore: cfg.ConversationStore,
		contextAssembler:  cfg.ContextAssembler,
		auditEmitter:      cfg.AuditEmitter,
		maxHistoryMessages: maxHist,
	}
}

// SetConversationStore configures the conversation memory backend (post-construction).
func (a *DefaultAgent) SetConversationStore(store ConversationStore) {
	a.conversationStore = store
}

// SetMaxHistoryMessages sets the maximum number of recent messages to inject into the planner.
func (a *DefaultAgent) SetMaxHistoryMessages(n int) {
	a.maxHistoryMessages = n
}

// SetContextAssembler configures the context assembler (post-construction).
func (a *DefaultAgent) SetContextAssembler(ca corecontext.ContextAssembler) {
	a.contextAssembler = ca
}

// SetAuditEmitter configures the audit emitter (post-construction).
func (a *DefaultAgent) SetAuditEmitter(e *audit.AuditEmitter) {
	a.auditEmitter = e
}

// HandleMessage implements Agent.
func (a *DefaultAgent) HandleMessage(ctx context.Context, req MessageRequest) (*MessageResponse, error) {
	// Auto-generate session ID if not provided
	if req.SessionID == "" {
		req.SessionID = uuid.NewString()
	}

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
	if a.contextAssembler != nil && a.runtime != nil {
		skillMsgs := MessagesToSkillMsgs(conversationHistory)
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
			if a.auditEmitter != nil {
				a.auditEmitter.Emit(ctx, audit.AuditEvent{
					Action:  "agent.context_assembly_failed",
					Outcome: "failure",
					Metadata: map[string]string{"error": err.Error()},
				})
			}
			slog.WarnContext(ctx, "agent: context assembly failed, using raw history",
				"error", err)
		} else if assembled != nil && len(assembled.Messages) > 0 {
			conversationHistory = SkillMsgsToMessages(assembled.Messages)
		}
	}

	// Step 3: Plan via LLM
	if a.runtime == nil {
		return nil, fmt.Errorf("agent: runtime is required for execution planning")
	}
	pCtx := &planner.PlannerContext{
		AssistantID: a.runtime.AssistantID,
		Soul:        a.runtime.Soul,
		Skills:      skillDescriptors,
		UserRequest: req.Message,
	}
	execPlan, err := a.executionPlanner.Plan(ctx, pCtx)
	if err != nil {
		return nil, fmt.Errorf("agent: planning failed: %w", err)
	}
	if err := execPlan.Validate(); err != nil {
		return nil, fmt.Errorf("agent: invalid plan: %w", err)
	}

	skillID := firstStepName(execPlan)

	// Step 4: Execute via Executor
	meta := ExecutionMeta{
		TenantID:  req.TenantID,
		UserID:    req.UserID,
		SessionID: req.SessionID,
	}

	result, err := a.executor.ExecuteFromPlan(ctx, execPlan, meta)
	if err != nil {
		resp := &MessageResponse{
			SessionID: req.SessionID,
			Message:   fmt.Sprintf("Error executing skill %s: %v", skillID, err),
			SkillUsed: skillID,
			Plan:      execPlan,
		}
		// Store messages even on error (best-effort)
		a.storeMessages(ctx, req, resp)
		return resp, nil
	}

	// Step 5: Build response
	resp := &MessageResponse{
		SessionID: req.SessionID,
		Message:   string(result.Output),
		SkillUsed: skillID,
		Plan:      execPlan,
	}

	// Step 6: Store messages (best-effort, does not block response)
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
		if a.auditEmitter != nil {
			a.auditEmitter.Emit(ctx, audit.AuditEvent{
				Action:   "agent.store_user_message_failed",
				Outcome:  "failure",
				Resource: req.SessionID,
				Metadata: map[string]string{"error": err.Error()},
			})
		}
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
		if a.auditEmitter != nil {
			a.auditEmitter.Emit(ctx, audit.AuditEvent{
				Action:   "agent.store_assistant_message_failed",
				Outcome:  "failure",
				Resource: req.SessionID,
				Metadata: map[string]string{"error": err.Error()},
			})
		}
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
