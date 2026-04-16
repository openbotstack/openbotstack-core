package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/openbotstack/openbotstack-core/ai/providers"
	"github.com/openbotstack/openbotstack-core/assistant"
	"github.com/openbotstack/openbotstack-core/control/skills"
)

// Planner uses an LLM to analyze user intent and select appropriate skills.
//
// The Planner is the ONLY component that decides which skill to invoke.
// It produces a structured ExecutionPlan that the Executor will run.
//
// Deprecated: Use planner.ExecutionPlanner from the planner package instead.
// This interface supports only single-skill selection. The planner package
// supports multi-step execution plans with validation and bounded limits.
// Migration path: replace agent.LLMPlanner with planner.LLMPlanner and
// adapt DefaultAgent to consume execution.ExecutionPlan.
type Planner interface {
	// Plan analyzes user intent and produces an execution plan.
	// Returns an ExecutionPlan specifying which skill to call and with what arguments.
	Plan(ctx context.Context, runtime *assistant.AssistantRuntime, req PlanRequest) (*ExecutionPlan, error)
}

// LLMPlanner implements Planner using the Model Router for skill selection.
//
// Deprecated: Use planner.LLMPlanner from the planner package instead.
// This implementation only supports single-skill selection without
// execution limits, persona injection, or memory context.
type LLMPlanner struct {
	router providers.ModelRouter
}

// NewLLMPlanner creates a new LLM-based planner.
func NewLLMPlanner(router providers.ModelRouter) *LLMPlanner {
	return &LLMPlanner{router: router}
}

// Plan implements Planner.
func (p *LLMPlanner) Plan(ctx context.Context, runtime *assistant.AssistantRuntime, req PlanRequest) (*ExecutionPlan, error) {
	if len(req.AvailableSkills) == 0 {
		return nil, ErrNoSkillsAvailable
	}

	// Build the prompt for the LLM
	prompt := p.buildPrompt(req)

	// Call the LLM using the model router
	mReq := skills.GenerateRequest{
		Messages: []skills.Message{
			{Role: "user", Content: prompt},
		},
	}
	provider, err := p.router.Route(
		[]skills.CapabilityType{skills.CapTextGeneration},
		skills.ModelConstraints{},
	)
	if err != nil {
		return nil, fmt.Errorf("%w: routing failed: %v", ErrPlanningFailed, err)
	}

	response, err := provider.Generate(ctx, mReq)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrPlanningFailed, err)
	}

	// Parse the LLM response into an ExecutionPlan
	plan, err := p.parseResponse(response.Content)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to parse LLM response: %v", ErrPlanningFailed, err)
	}

	if err := plan.Validate(); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrPlanningFailed, err)
	}

	return plan, nil
}

// buildPrompt constructs the LLM prompt for skill selection.
func (p *LLMPlanner) buildPrompt(req PlanRequest) string {
	var sb strings.Builder

	sb.WriteString("You are an AI assistant that selects the most appropriate skill to handle a user request.\n\n")

	sb.WriteString("Available skills:\n")
	for _, skill := range req.AvailableSkills {
		_, _ = fmt.Fprintf(&sb, "- %s (%s): %s\n", skill.ID, skill.Name, skill.Description)
		if skill.InputSchema != nil {
			schemaJSON, _ := json.Marshal(skill.InputSchema)
			_, _ = fmt.Fprintf(&sb, "  Input schema: %s\n", string(schemaJSON))
		}
	}

	// Inject conversation history context
	if len(req.ConversationHistory) > 0 {
		sb.WriteString("\nPrevious conversation:\n")
		for _, msg := range req.ConversationHistory {
			_, _ = fmt.Fprintf(&sb, "[%s]: %s\n", msg.Role, msg.Content)
		}
	}

	sb.WriteString("\nUser message: ")
	sb.WriteString(req.UserMessage)
	sb.WriteString("\n\n")

	sb.WriteString(`Respond with a JSON object containing:
1. "skill_id": the ID of the skill to use
2. "arguments": a JSON object with the skill's input arguments
3. "reasoning": brief explanation of why this skill was chosen

Example response:
{"skill_id": "core/summarize", "arguments": {"text": "...", "max_length": 200}, "reasoning": "User wants to summarize text"}

Respond ONLY with the JSON object, no other text.`)

	return sb.String()
}

// parseResponse extracts an ExecutionPlan from the LLM response.
func (p *LLMPlanner) parseResponse(response string) (*ExecutionPlan, error) {
	// Clean up the response - LLMs sometimes wrap JSON in markdown
	response = strings.TrimSpace(response)
	response = strings.TrimPrefix(response, "```json")
	response = strings.TrimPrefix(response, "```")
	response = strings.TrimSuffix(response, "```")
	response = strings.TrimSpace(response)

	var plan ExecutionPlan
	if err := json.Unmarshal([]byte(response), &plan); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w (response: %s)", err, response)
	}

	return &plan, nil
}
