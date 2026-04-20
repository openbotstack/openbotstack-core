// Package policy implements a rule-based policy enforcement engine.
//
// PolicyEnforcer evaluates governance rules against execution requests,
// returning allow or deny decisions based on tenant-scoped rule sets.
// Rules are evaluated in priority order (highest first), and the first
// matching rule determines the outcome. Default policy is allow.
package policy

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
)

// PolicyRule defines a single governance rule.
type PolicyRule struct {
	ID         string
	TenantID   string
	Effect     string // "allow" or "deny"
	Action     string // e.g., "skill.execute", "model.generate", "*"
	Resource   string // e.g., "skill/summarize", "model/*", "*"
	Conditions map[string]string
	Priority   int // higher = evaluated first
}

// Enforcer evaluates rules and returns allow/deny decisions.
type Enforcer struct {
	mu   sync.RWMutex
	rules map[string]map[string]PolicyRule // tenantID -> ruleID -> rule
}

// NewEnforcer creates a new policy enforcer.
func NewEnforcer() *Enforcer {
	return &Enforcer{
		rules: make(map[string]map[string]PolicyRule),
	}
}

// Evaluate checks all applicable rules for the given context.
// Returns nil if allowed, an error if denied or inputs are invalid.
func (e *Enforcer) Evaluate(ctx context.Context, tenantID, action, resource string, attributes map[string]string) error {
	if ctx == nil {
		return fmt.Errorf("policy: context is required")
	}
	if tenantID == "" {
		return fmt.Errorf("policy: tenantID is required")
	}
	if action == "" {
		return fmt.Errorf("policy: action is required")
	}

	e.mu.RLock()
	defer e.mu.RUnlock()

	tenantRules := e.rules[tenantID]
	if len(tenantRules) == 0 {
		return nil // default allow when no rules
	}

	// Sort rules by priority (highest first)
	sorted := make([]PolicyRule, 0, len(tenantRules))
	for _, r := range tenantRules {
		sorted = append(sorted, r)
	}
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Priority > sorted[j].Priority
	})

	for _, rule := range sorted {
		if !matchAction(rule.Action, action) {
			continue
		}
		if !matchResource(rule.Resource, resource) {
			continue
		}
		if !matchConditions(rule.Conditions, attributes) {
			continue
		}

		// Rule matches
		switch rule.Effect {
		case "deny":
			return fmt.Errorf("policy: denied by rule %s", rule.ID)
		case "allow":
			return nil
		}
	}

	return nil // default allow
}

// AddRule adds a governance rule.
func (e *Enforcer) AddRule(rule PolicyRule) error {
	if rule.ID == "" {
		return fmt.Errorf("policy: rule ID is required")
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	if _, ok := e.rules[rule.TenantID]; !ok {
		e.rules[rule.TenantID] = make(map[string]PolicyRule)
	}

	if _, exists := e.rules[rule.TenantID][rule.ID]; exists {
		return fmt.Errorf("policy: rule %s already exists", rule.ID)
	}

	e.rules[rule.TenantID][rule.ID] = rule
	return nil
}

// RemoveRule removes a rule by ID.
func (e *Enforcer) RemoveRule(ruleID string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	for tid, rules := range e.rules {
		if _, ok := rules[ruleID]; ok {
			delete(e.rules[tid], ruleID)
			return nil
		}
	}
	return fmt.Errorf("policy: rule %s not found", ruleID)
}

// ListRules returns all rules for a tenant.
func (e *Enforcer) ListRules(tenantID string) ([]PolicyRule, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	tenantRules := e.rules[tenantID]
	if len(tenantRules) == 0 {
		return []PolicyRule{}, nil
	}

	result := make([]PolicyRule, 0, len(tenantRules))
	for _, r := range tenantRules {
		result = append(result, r)
	}
	return result, nil
}

// matchAction checks if a rule's action pattern matches the given action.
func matchAction(pattern, action string) bool {
	if pattern == "*" {
		return true
	}
	return pattern == action
}

// matchResource checks if a rule's resource pattern matches the given resource.
func matchResource(pattern, resource string) bool {
	if pattern == "*" {
		return true
	}
	if strings.HasSuffix(pattern, "/*") {
		prefix := strings.TrimSuffix(pattern, "/*")
		return strings.HasPrefix(resource, prefix+"/") || resource == prefix
	}
	return pattern == resource
}

// matchConditions checks if all rule conditions match the given attributes.
// Supports:
//   - Exact match: "role": "admin"
//   - Time window: "time_after": "18:00", "time_before": "09:00" (both required)
func matchConditions(conditions, attributes map[string]string) bool {
	if len(conditions) == 0 {
		return true
	}

	// Handle time window conditions specially
	if after, hasAfter := conditions["time_after"]; hasAfter {
		before, hasBefore := conditions["time_before"]
		if hasBefore {
			currentTime, ok := attributes["current_time"]
			if !ok {
				return false
			}
			return matchTimeWindow(after, before, currentTime)
		}
	}

	// Exact match for all other conditions
	for key, val := range conditions {
		if key == "time_after" || key == "time_before" {
			continue
		}
		attrVal, ok := attributes[key]
		if !ok || attrVal != val {
			return false
		}
	}
	return true
}

// matchTimeWindow checks if currentTime falls within the after-before window.
// Supports overnight windows (e.g., 18:00-09:00).
func matchTimeWindow(after, before, currentTime string) bool {
	// Simple string comparison works for HH:MM format
	if after > before {
		// Overnight window: 18:00-09:00 means >= 18:00 OR < 09:00
		return currentTime >= after || currentTime < before
	}
	// Normal window: 09:00-17:00 means >= 09:00 AND < 17:00
	return currentTime >= after && currentTime < before
}
