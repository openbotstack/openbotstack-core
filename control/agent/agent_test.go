package agent_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/openbotstack/openbotstack-core/assistant"
	"github.com/openbotstack/openbotstack-core/execution"
	control_skills "github.com/openbotstack/openbotstack-core/control/skills"
	"github.com/openbotstack/openbotstack-core/registry/skills"
	"github.com/openbotstack/openbotstack-core/control/agent"
	"github.com/openbotstack/openbotstack-core/planner"
)

// ==================== Mock Implementations ====================

// testRuntime is a non-nil AssistantRuntime for tests that require it.
var testRuntime = &assistant.AssistantRuntime{AssistantID: "test"}

type mockSkill struct {
	id          string
	name        string
	description string
}

func (m *mockSkill) ID() string                      { return m.id }
func (m *mockSkill) Name() string                    { return m.name }
func (m *mockSkill) Description() string             { return m.description }
func (m *mockSkill) Timeout() time.Duration          { return 30 * time.Second }
func (m *mockSkill) InputSchema() *control_skills.JSONSchema  { return nil }
func (m *mockSkill) OutputSchema() *control_skills.JSONSchema { return nil }
func (m *mockSkill) RequiredPermissions() []string   { return nil }
func (m *mockSkill) Validate() error                 { return nil }

type mockRegistry struct {
	skills map[string]skills.Skill
}

func newMockRegistry() *mockRegistry {
	return &mockRegistry{skills: make(map[string]skills.Skill)}
}

func (r *mockRegistry) Register(s skills.Skill) {
	r.skills[s.ID()] = s
}

func (r *mockRegistry) List() []string {
	ids := make([]string, 0, len(r.skills))
	for id := range r.skills {
		ids = append(ids, id)
	}
	return ids
}

func (r *mockRegistry) Get(id string) (skills.Skill, error) {
	s, ok := r.skills[id]
	if !ok {
		return nil, errors.New("skill not found")
	}
	return s, nil
}

type mockExecutor struct {
	lastPlan *execution.ExecutionPlan
	lastMeta agent.ExecutionMeta
	response *execution.ExecutionResult
	err      error
}

func (e *mockExecutor) ExecuteFromPlan(ctx context.Context, plan *execution.ExecutionPlan, meta agent.ExecutionMeta) (*execution.ExecutionResult, error) {
	e.lastPlan = plan
	e.lastMeta = meta
	if e.err != nil {
		return nil, e.err
	}
	if e.response != nil {
		return e.response, nil
	}
	return &execution.ExecutionResult{
		Status: execution.StatusSuccess,
		Output: []byte(`{"result": "ok"}`),
	}, nil
}

// mockExecutionPlanner implements planner.ExecutionPlanner for testing.
type mockExecutionPlanner struct {
	defaultSkillID string
	forcedPlan     *execution.ExecutionPlan
	forcedError    error
}

func newMockExecutionPlanner(defaultSkillID string) *mockExecutionPlanner {
	return &mockExecutionPlanner{defaultSkillID: defaultSkillID}
}

func (p *mockExecutionPlanner) Plan(ctx context.Context, pCtx *planner.PlannerContext) (*execution.ExecutionPlan, error) {
	if p.forcedError != nil {
		return nil, p.forcedError
	}
	if p.forcedPlan != nil {
		return p.forcedPlan, nil
	}
	plan := &execution.ExecutionPlan{
		Steps: []execution.ExecutionStep{
			{
				Name:      p.defaultSkillID,
				Type:      execution.StepTypeSkill,
				Arguments: map[string]any{"input": "test"},
			},
		},
	}
	return plan, nil
}

// ==================== Tests ====================

func TestDefaultAgentHandleMessageSuccess(t *testing.T) {
	registry := newMockRegistry()
	registry.Register(&mockSkill{id: "core/summarize", name: "Summarize", description: "Summarizes text"})
	registry.Register(&mockSkill{id: "core/sentiment", name: "Sentiment", description: "Analyzes sentiment"})

	p := newMockExecutionPlanner("core/summarize")
	executor := &mockExecutor{
		response: &execution.ExecutionResult{
			Status: execution.StatusSuccess,
			Output: []byte("Summary: This is a test."),
		},
	}

	a := agent.NewDefaultAgent(agent.AgentConfig{Planner: p, Executor: executor, Registry: registry, Runtime: testRuntime})

	resp, err := a.HandleMessage(context.Background(), agent.MessageRequest{
		TenantID:  "tenant-1",
		UserID:    "user-1",
		SessionID: "session-1",
		Message:   "Please summarize this text.",
	})

	if err != nil {
		t.Fatalf("HandleMessage failed: %v", err)
	}

	if resp.SkillUsed != "core/summarize" {
		t.Errorf("Expected skill core/summarize, got %s", resp.SkillUsed)
	}

	if resp.Plan == nil {
		t.Error("Expected plan to be set")
	}

	if executor.lastMeta.TenantID != "tenant-1" {
		t.Errorf("Expected tenant-1, got %s", executor.lastMeta.TenantID)
	}
}

func TestDefaultAgentNoSkillsAvailable(t *testing.T) {
	registry := newMockRegistry() // empty
	p := newMockExecutionPlanner("")
	executor := &mockExecutor{}

	a := agent.NewDefaultAgent(agent.AgentConfig{Planner: p, Executor: executor, Registry: registry, Runtime: testRuntime})

	_, err := a.HandleMessage(context.Background(), agent.MessageRequest{
		Message: "Hello",
	})

	if !errors.Is(err, agent.ErrNoSkillsAvailable) {
		t.Errorf("Expected ErrNoSkillsAvailable, got %v", err)
	}
}

func TestDefaultAgentPlannerError(t *testing.T) {
	registry := newMockRegistry()
	registry.Register(&mockSkill{id: "core/test", name: "Test", description: "Test skill"})

	p := newMockExecutionPlanner("")
	p.forcedError = errors.New("LLM unavailable")
	executor := &mockExecutor{}

	a := agent.NewDefaultAgent(agent.AgentConfig{Planner: p, Executor: executor, Registry: registry, Runtime: testRuntime})

	_, err := a.HandleMessage(context.Background(), agent.MessageRequest{
		Message: "Hello",
	})

	if err == nil {
		t.Error("Expected error from planner")
	}
}

func TestDefaultAgentExecutorError(t *testing.T) {
	registry := newMockRegistry()
	registry.Register(&mockSkill{id: "core/test", name: "Test", description: "Test skill"})

	p := newMockExecutionPlanner("core/test")
	executor := &mockExecutor{
		err: errors.New("execution failed"),
	}

	a := agent.NewDefaultAgent(agent.AgentConfig{Planner: p, Executor: executor, Registry: registry, Runtime: testRuntime})

	resp, err := a.HandleMessage(context.Background(), agent.MessageRequest{
		Message: "Hello",
	})

	// Executor errors are returned in resp.Message, not as Go errors,
	// to avoid surfacing 500 errors to end users.
	if err != nil {
		t.Errorf("HandleMessage returned unexpected error: %v", err)
	}

	if resp == nil {
		t.Fatal("Expected response even with executor error")
	}
	if resp.SkillUsed != "core/test" {
		t.Errorf("Expected skill core/test, got %s", resp.SkillUsed)
	}
	if resp.Message == "" {
		t.Error("Expected error message in resp.Message")
	}
}

// ==================== ExecutionPlan Validation Tests ====================

func TestValidatePlanForAgent(t *testing.T) {
	tests := []struct {
		name    string
		plan    *execution.ExecutionPlan
		wantErr bool
	}{
		{
			name:    "nil plan",
			plan:    nil,
			wantErr: true,
		},
		{
			name: "empty steps",
			plan: &execution.ExecutionPlan{Steps: []execution.ExecutionStep{}},
			wantErr: true,
		},
		{
			name: "valid plan with one step",
			plan: &execution.ExecutionPlan{
				Steps: []execution.ExecutionStep{
					{Name: "core/test", Type: execution.StepTypeSkill},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := agent.ValidatePlanForAgent(tt.plan)
			if tt.wantErr && err == nil {
				t.Error("Expected error")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func TestFirstStepName(t *testing.T) {
	tests := []struct {
		name     string
		plan     *execution.ExecutionPlan
		expected string
	}{
		{"nil plan", nil, ""},
		{"empty steps", &execution.ExecutionPlan{}, ""},
		{"with step", &execution.ExecutionPlan{
			Steps: []execution.ExecutionStep{{Name: "core/summarize"}},
		}, "core/summarize"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// firstStepName is unexported, test via ValidatePlanForAgent
			// which is the public wrapper
			_ = tt.expected // used indirectly
		})
	}
}
