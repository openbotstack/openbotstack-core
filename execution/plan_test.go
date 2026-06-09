package execution

import (
	"testing"
)

func TestExecutionStep_ParallelFields(t *testing.T) {
	step := ExecutionStep{
		Name:           "search",
		Type:           StepTypeSkill,
		Parallelizable: true,
		ParallelGroup:  "group_a",
	}
	if !step.Parallelizable {
		t.Error("expected Parallelizable=true")
	}
	if step.ParallelGroup != "group_a" {
		t.Errorf("expected ParallelGroup='group_a', got %q", step.ParallelGroup)
	}
}

func TestExecutionStep_DefaultParallelFalse(t *testing.T) {
	step := ExecutionStep{Name: "search", Type: StepTypeSkill}
	if step.Parallelizable {
		t.Error("default Parallelizable should be false")
	}
	if step.ParallelGroup != "" {
		t.Errorf("default ParallelGroup should be empty, got %q", step.ParallelGroup)
	}
}

func TestExecutionPlan_ParallelSteps(t *testing.T) {
	plan := ExecutionPlan{
		Steps: []ExecutionStep{
			{Name: "a", Type: StepTypeSkill, Parallelizable: true, ParallelGroup: "batch"},
			{Name: "b", Type: StepTypeSkill, Parallelizable: true, ParallelGroup: "batch"},
			{Name: "c", Type: StepTypeSkill}, // sequential
		},
	}
	if err := plan.Validate(); err != nil {
		t.Fatalf("valid plan should pass: %v", err)
	}

	// Verify grouping logic
	groups := GroupParallelSteps(plan.Steps)
	if len(groups) != 2 {
		t.Fatalf("expected 2 groups, got %d", len(groups))
	}
	// First group: a,b parallel
	if len(groups[0]) != 2 {
		t.Errorf("first group should have 2 steps, got %d", len(groups[0]))
	}
	// Second group: c sequential
	if len(groups[1]) != 1 {
		t.Errorf("second group should have 1 step, got %d", len(groups[1]))
	}
}

func TestExecutionPlan_AllSequential(t *testing.T) {
	plan := ExecutionPlan{
		Steps: []ExecutionStep{
			{Name: "a", Type: StepTypeSkill},
			{Name: "b", Type: StepTypeSkill},
		},
	}
	groups := GroupParallelSteps(plan.Steps)
	if len(groups) != 2 {
		t.Fatalf("expected 2 individual groups, got %d", len(groups))
	}
}

func TestExecutionPlan_AllParallel(t *testing.T) {
	plan := ExecutionPlan{
		Steps: []ExecutionStep{
			{Name: "a", Type: StepTypeSkill, Parallelizable: true, ParallelGroup: "batch"},
			{Name: "b", Type: StepTypeSkill, Parallelizable: true, ParallelGroup: "batch"},
			{Name: "c", Type: StepTypeSkill, Parallelizable: true, ParallelGroup: "batch"},
		},
	}
	groups := GroupParallelSteps(plan.Steps)
	if len(groups) != 1 {
		t.Fatalf("expected 1 parallel group, got %d", len(groups))
	}
	if len(groups[0]) != 3 {
		t.Errorf("group should have 3 steps, got %d", len(groups[0]))
	}
}

func TestExecutionPlan_MultipleParallelGroups(t *testing.T) {
	plan := ExecutionPlan{
		Steps: []ExecutionStep{
			{Name: "a", Type: StepTypeSkill, Parallelizable: true, ParallelGroup: "g1"},
			{Name: "b", Type: StepTypeSkill, Parallelizable: true, ParallelGroup: "g1"},
			{Name: "c", Type: StepTypeSkill}, // sequential separator
			{Name: "d", Type: StepTypeSkill, Parallelizable: true, ParallelGroup: "g2"},
			{Name: "e", Type: StepTypeSkill, Parallelizable: true, ParallelGroup: "g2"},
		},
	}
	groups := GroupParallelSteps(plan.Steps)
	if len(groups) != 3 {
		t.Fatalf("expected 3 groups, got %d", len(groups))
	}
	if len(groups[0]) != 2 {
		t.Errorf("group 0 should have 2, got %d", len(groups[0]))
	}
	if len(groups[1]) != 1 {
		t.Errorf("group 1 should have 1, got %d", len(groups[1]))
	}
	if len(groups[2]) != 2 {
		t.Errorf("group 2 should have 2, got %d", len(groups[2]))
	}
}

func TestExecutionStep_ParallelizableNoGroup(t *testing.T) {
	// Parallelizable=true but no group => treated as sequential
	steps := []ExecutionStep{
		{Name: "a", Type: StepTypeSkill, Parallelizable: true, ParallelGroup: ""},
		{Name: "b", Type: StepTypeSkill, Parallelizable: true, ParallelGroup: ""},
	}
	groups := GroupParallelSteps(steps)
	if len(groups) != 2 {
		t.Fatalf("expected 2 individual groups (no group set), got %d", len(groups))
	}
}

func TestExecutionPlan_Empty(t *testing.T) {
	groups := GroupParallelSteps(nil)
	if len(groups) != 0 {
		t.Errorf("nil steps should return empty, got %d", len(groups))
	}
}

// --- G12: non-consecutive same-group edge case ---

func TestGroupParallelSteps_NonConsecutiveSameGroup(t *testing.T) {
	steps := []ExecutionStep{
		{Name: "a", Parallelizable: true, ParallelGroup: "g1"},
		{Name: "b", Parallelizable: true, ParallelGroup: "g1"},
		{Name: "seq", Type: StepTypeSkill},
		{Name: "c", Parallelizable: true, ParallelGroup: "g1"},
	}

	groups := GroupParallelSteps(steps)

	if len(groups) != 3 {
		t.Fatalf("expected 3 groups (a+b, seq, c), got %d", len(groups))
	}
	if len(groups[0]) != 2 {
		t.Errorf("group 0: expected 2 steps (a,b), got %d", len(groups[0]))
	}
	if groups[1][0].Name != "seq" {
		t.Errorf("group 1: expected 'seq', got %q", groups[1][0].Name)
	}
	if len(groups[2]) != 1 {
		t.Errorf("group 2: expected 1 step (c), got %d", len(groups[2]))
	}
}

func TestExecutionPlan_DuplicateStepNames(t *testing.T) {
	plan := ExecutionPlan{
		Steps: []ExecutionStep{
			{Name: "fetch", Type: StepTypeTool},
			{Name: "fetch", Type: StepTypeTool},
		},
	}
	err := plan.Validate()
	if err == nil {
		t.Fatal("expected error for duplicate step names")
	}
	if want := "duplicate step name"; !containsString(err.Error(), want) {
		t.Errorf("error = %q, want to contain %q", err.Error(), want)
	}
}

func TestExecutionPlan_UniqueStepNames(t *testing.T) {
	plan := ExecutionPlan{
		Steps: []ExecutionStep{
			{Name: "fetch", Type: StepTypeTool},
			{Name: "analyze", Type: StepTypeSkill},
		},
	}
	if err := plan.Validate(); err != nil {
		t.Fatalf("unique names should pass: %v", err)
	}
}

func containsString(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 || containsAt(s, sub))
}

func containsAt(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

// --- Phase 1: Hardened Plan Tests ---

func TestValidate_StepTypeLLM(t *testing.T) {
	plan := ExecutionPlan{
		Steps: []ExecutionStep{
			{Name: "reason", Type: StepTypeLLM},
		},
	}
	if err := plan.Validate(); err != nil {
		t.Fatalf("StepTypeLLM should be valid: %v", err)
	}
}

func TestValidate_InvalidType(t *testing.T) {
	plan := ExecutionPlan{
		Steps: []ExecutionStep{
			{Name: "bad", Type: StepType("unknown")},
		},
	}
	err := plan.Validate()
	if err == nil {
		t.Fatal("expected error for invalid step type")
	}
}

func TestValidate_AutoGeneratesStepID(t *testing.T) {
	plan := ExecutionPlan{
		Steps: []ExecutionStep{
			{Name: "step1", Type: StepTypeTool},
			{Name: "step2", Type: StepTypeSkill},
		},
	}
	if err := plan.Validate(); err != nil {
		t.Fatalf("Validate: %v", err)
	}
	for _, s := range plan.Steps {
		if s.StepID == "" {
			t.Errorf("step %q should have auto-generated StepID", s.Name)
		}
	}
}

func TestValidate_PreservesExistingStepID(t *testing.T) {
	plan := ExecutionPlan{
		Steps: []ExecutionStep{
			{Name: "step1", Type: StepTypeTool, StepID: "custom-id-123"},
		},
	}
	if err := plan.Validate(); err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if plan.Steps[0].StepID != "custom-id-123" {
		t.Errorf("expected preserved StepID, got %q", plan.Steps[0].StepID)
	}
}

func TestValidate_FreezesPlan(t *testing.T) {
	plan := ExecutionPlan{
		Steps: []ExecutionStep{
			{Name: "step1", Type: StepTypeTool},
		},
	}
	if plan.IsFrozen() {
		t.Error("plan should not be frozen before Validate")
	}
	if err := plan.Validate(); err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if !plan.IsFrozen() {
		t.Error("plan should be frozen after Validate")
	}
}

func TestValidate_DoubleValidateRejected(t *testing.T) {
	plan := ExecutionPlan{
		Steps: []ExecutionStep{
			{Name: "step1", Type: StepTypeTool},
		},
	}
	if err := plan.Validate(); err != nil {
		t.Fatalf("first Validate: %v", err)
	}
	err := plan.Validate()
	if err == nil {
		t.Fatal("second Validate should fail on frozen plan")
	}
}

func TestExecutionStep_TimeoutField(t *testing.T) {
	step := ExecutionStep{
		Name:    "slow-op",
		Type:    StepTypeTool,
		Timeout: 5000,
	}
	if step.Timeout != 5000 {
		t.Errorf("expected Timeout=5000, got %d", step.Timeout)
	}
}

func TestExecutionStep_ExpectedOutputField(t *testing.T) {
	step := ExecutionStep{
		Name:           "analyze",
		Type:           StepTypeLLM,
		ExpectedOutput: "structured analysis with 3-5 differential diagnoses",
	}
	if step.ExpectedOutput == "" {
		t.Error("ExpectedOutput should be set")
	}
}

func TestValidate_AutoGeneratesPlanID(t *testing.T) {
	plan := ExecutionPlan{
		Steps: []ExecutionStep{
			{Name: "step1", Type: StepTypeTool},
		},
	}
	if plan.ID != "" {
		t.Errorf("plan ID should be empty before Validate, got %q", plan.ID)
	}
	if err := plan.Validate(); err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if plan.ID == "" {
		t.Error("plan ID should be auto-generated after Validate")
	}
}

func TestValidate_PreservesExistingPlanID(t *testing.T) {
	plan := ExecutionPlan{
		Steps: []ExecutionStep{
			{Name: "step1", Type: StepTypeTool},
		},
	}
	plan.ID = "custom-plan-id-789"
	if err := plan.Validate(); err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if plan.ID != "custom-plan-id-789" {
		t.Errorf("expected preserved plan ID, got %q", plan.ID)
	}
}

func TestExecutionPlan_ParentID(t *testing.T) {
	plan := ExecutionPlan{
		Steps: []ExecutionStep{
			{Name: "step1", Type: StepTypeTool},
		},
	}
	plan.ParentID = "parent-plan-001"
	if err := plan.Validate(); err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if plan.ParentID != "parent-plan-001" {
		t.Errorf("ParentID should be preserved, got %q", plan.ParentID)
	}
}

func TestValidate_AllStepTypes(t *testing.T) {
	plan := ExecutionPlan{
		Steps: []ExecutionStep{
			{Name: "tool-step", Type: StepTypeTool},
			{Name: "skill-step", Type: StepTypeSkill},
			{Name: "llm-step", Type: StepTypeLLM},
		},
	}
	if err := plan.Validate(); err != nil {
		t.Fatalf("all step types should be valid: %v", err)
	}
	if !plan.IsFrozen() {
		t.Error("plan should be frozen")
	}
}
