package policy

import (
	"context"
	"testing"
)

// --- Normal Cases (4) ---

func TestEvaluate_AllowWhenNoDenyRules(t *testing.T) {
	e := NewEnforcer()
	err := e.Evaluate(context.Background(), "tenant-1", "skill.execute", "skill/summarize", nil)
	if err != nil {
		t.Errorf("expected allow with no rules, got: %v", err)
	}
}

func TestEvaluate_DenyByExplicitRule(t *testing.T) {
	e := NewEnforcer()
	_ = e.AddRule(PolicyRule{
		ID:       "deny-members-summary",
		TenantID: "tenant-1",
		Effect:   "deny",
		Action:   "skill.execute",
		Resource: "skill/summarize",
		Priority: 10,
	})
	err := e.Evaluate(context.Background(), "tenant-1", "skill.execute", "skill/summarize", map[string]string{"role": "member"})
	if err == nil {
		t.Error("expected deny, got allow")
	}
}

func TestEvaluate_AllowOverrideDeny(t *testing.T) {
	e := NewEnforcer()
	_ = e.AddRule(PolicyRule{
		ID:       "deny-all",
		TenantID: "tenant-1",
		Effect:   "deny",
		Action:   "skill.execute",
		Resource: "skill/*",
		Priority: 1,
	})
	_ = e.AddRule(PolicyRule{
		ID:       "allow-admin",
		TenantID: "tenant-1",
		Effect:   "allow",
		Action:   "skill.execute",
		Resource: "skill/*",
		Conditions: map[string]string{"role": "admin"},
		Priority: 100,
	})
	err := e.Evaluate(context.Background(), "tenant-1", "skill.execute", "skill/summarize", map[string]string{"role": "admin"})
	if err != nil {
		t.Errorf("expected allow (higher priority override), got: %v", err)
	}
}

func TestEvaluate_TenantIsolation(t *testing.T) {
	e := NewEnforcer()
	_ = e.AddRule(PolicyRule{
		ID:       "deny-tenant-a",
		TenantID: "tenant-a",
		Effect:   "deny",
		Action:   "skill.execute",
		Resource: "skill/*",
		Priority: 10,
	})
	err := e.Evaluate(context.Background(), "tenant-b", "skill.execute", "skill/summarize", nil)
	if err != nil {
		t.Errorf("tenant-b should not be affected by tenant-a rules, got: %v", err)
	}
}

// --- Abnormal / Edge Cases (13) ---

func TestEvaluate_DenyWhenDenyAllPolicy(t *testing.T) {
	e := NewEnforcer()
	_ = e.AddRule(PolicyRule{
		ID:       "deny-all",
		TenantID: "tenant-1",
		Effect:   "deny",
		Action:   "*",
		Resource: "*",
		Priority: 1,
	})
	err := e.Evaluate(context.Background(), "tenant-1", "anything", "anything", nil)
	if err == nil {
		t.Error("expected deny for deny-all policy")
	}
}

func TestEvaluate_MultipleDenyRules(t *testing.T) {
	e := NewEnforcer()
	_ = e.AddRule(PolicyRule{
		ID: "deny-1", TenantID: "t1", Effect: "deny",
		Action: "skill.execute", Resource: "skill/a", Priority: 10,
	})
	_ = e.AddRule(PolicyRule{
		ID: "deny-2", TenantID: "t1", Effect: "deny",
		Action: "skill.execute", Resource: "skill/b", Priority: 10,
	})
	err := e.Evaluate(context.Background(), "t1", "skill.execute", "skill/a", nil)
	if err == nil {
		t.Error("expected deny from matching rule")
	}
}

func TestEvaluate_WildcardResource(t *testing.T) {
	e := NewEnforcer()
	_ = e.AddRule(PolicyRule{
		ID: "deny-models", TenantID: "t1", Effect: "deny",
		Action: "model.generate", Resource: "model/*", Priority: 10,
	})
	err := e.Evaluate(context.Background(), "t1", "model.generate", "model/gpt-4o", nil)
	if err == nil {
		t.Error("expected deny for wildcard match on model/*")
	}
}

func TestEvaluate_ConditionRoleMatch(t *testing.T) {
	e := NewEnforcer()
	_ = e.AddRule(PolicyRule{
		ID: "deny-member", TenantID: "t1", Effect: "deny",
		Action: "skill.execute", Resource: "skill/*",
		Conditions: map[string]string{"role": "member"}, Priority: 10,
	})
	err := e.Evaluate(context.Background(), "t1", "skill.execute", "skill/x", map[string]string{"role": "member"})
	if err == nil {
		t.Error("expected deny when role condition matches")
	}
}

func TestEvaluate_ConditionRoleMismatch(t *testing.T) {
	e := NewEnforcer()
	_ = e.AddRule(PolicyRule{
		ID: "deny-member", TenantID: "t1", Effect: "deny",
		Action: "skill.execute", Resource: "skill/*",
		Conditions: map[string]string{"role": "member"}, Priority: 10,
	})
	err := e.Evaluate(context.Background(), "t1", "skill.execute", "skill/x", map[string]string{"role": "admin"})
	if err != nil {
		t.Errorf("expected allow when condition doesn't match, got: %v", err)
	}
}

func TestEvaluate_ConditionTimeWindow(t *testing.T) {
	e := NewEnforcer()
	_ = e.AddRule(PolicyRule{
		ID: "deny-after-hours", TenantID: "t1", Effect: "deny",
		Action: "skill.execute", Resource: "skill/*",
		Conditions: map[string]string{"time_after": "18:00", "time_before": "09:00"},
		Priority: 10,
	})
	err := e.Evaluate(context.Background(), "t1", "skill.execute", "skill/x", map[string]string{"current_time": "22:00"})
	if err == nil {
		t.Error("expected deny during after-hours time window")
	}
}

func TestEvaluate_EmptyTenantID(t *testing.T) {
	e := NewEnforcer()
	err := e.Evaluate(context.Background(), "", "skill.execute", "skill/x", nil)
	if err == nil {
		t.Error("expected error for empty tenantID")
	}
}

func TestEvaluate_EmptyAction(t *testing.T) {
	e := NewEnforcer()
	err := e.Evaluate(context.Background(), "t1", "", "skill/x", nil)
	if err == nil {
		t.Error("expected error for empty action")
	}
}

func TestEvaluate_NilContext(t *testing.T) {
	e := NewEnforcer()
	err := e.Evaluate(nil, "t1", "skill.execute", "skill/x", nil)
	if err == nil {
		t.Error("expected error for nil context")
	}
}

func TestAddRule_DuplicateID(t *testing.T) {
	e := NewEnforcer()
	rule := PolicyRule{ID: "r1", TenantID: "t1", Effect: "deny", Action: "a", Resource: "r"}
	_ = e.AddRule(rule)
	err := e.AddRule(rule)
	if err == nil {
		t.Error("expected error for duplicate rule ID")
	}
}

func TestAddRule_EmptyID(t *testing.T) {
	e := NewEnforcer()
	err := e.AddRule(PolicyRule{ID: "", TenantID: "t1", Effect: "deny"})
	if err == nil {
		t.Error("expected error for empty rule ID")
	}
}

func TestRemoveRule_NotFound(t *testing.T) {
	e := NewEnforcer()
	err := e.RemoveRule("nonexistent")
	if err == nil {
		t.Error("expected error for removing nonexistent rule")
	}
}

func TestListRules_EmptyTenant(t *testing.T) {
	e := NewEnforcer()
	rules, err := e.ListRules("nonexistent-tenant")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if len(rules) != 0 {
		t.Errorf("expected empty slice, got %d rules", len(rules))
	}
}
