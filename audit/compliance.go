package audit

import "time"

// ComplianceReport aggregates audit data into a compliance-oriented summary.
// It provides a point-in-time snapshot of system health, policy adherence,
// execution reliability, and audit chain integrity for a given scope.
type ComplianceReport struct {
	// ID is a unique identifier for this report.
	ID string `json:"id"`

	// GeneratedAt is when the report was generated.
	GeneratedAt time.Time `json:"generated_at"`

	// Scope defines the reporting scope (tenant, global).
	Scope ComplianceScope `json:"scope"`

	// Period is the time range covered by the report.
	Period TimeRange `json:"period"`

	// Summary contains high-level statistics.
	Summary ComplianceSummary `json:"summary"`

	// PolicyCompliance summarizes policy enforcement outcomes.
	PolicyCompliance PolicyComplianceSection `json:"policy_compliance"`

	// ExecutionHealth summarizes execution outcomes.
	ExecutionHealth ExecutionHealthSection `json:"execution_health"`

	// ChainIntegrity summarizes HMAC chain verification results.
	ChainIntegrity ChainIntegritySection `json:"chain_integrity"`

	// RetentionCompliance summarizes retention policy status.
	RetentionCompliance RetentionComplianceSection `json:"retention_compliance"`

	// TopErrors lists the most frequent error patterns (max 10).
	TopErrors []ErrorPattern `json:"top_errors,omitempty"`
}

// ComplianceScope defines the scope of a compliance report.
type ComplianceScope struct {
	// TenantID is empty for global reports, or a specific tenant ID.
	TenantID string `json:"tenant_id,omitempty"`

	// UserID is empty for tenant/global scope, or a specific user.
	UserID string `json:"user_id,omitempty"`
}

// TimeRange represents a time interval.
type TimeRange struct {
	From time.Time `json:"from"`
	To   time.Time `json:"to"`
}

// ComplianceSummary contains high-level counts and rates.
type ComplianceSummary struct {
	// TotalEvents is the total number of audit events in the period.
	TotalEvents int `json:"total_events"`

	// TotalExecutions is the number of unique execution IDs.
	TotalExecutions int `json:"total_executions"`

	// ErrorRate is the percentage of events with error outcome (0-100).
	ErrorRate float64 `json:"error_rate"`

	// DenialRate is the percentage of policy events that were denied (0-100).
	DenialRate float64 `json:"denial_rate"`
}

// PolicyComplianceSection summarizes policy enforcement.
type PolicyComplianceSection struct {
	// TotalChecks is the number of policy check events.
	TotalChecks int `json:"total_checks"`

	// Allowed is the count of policy.allowed events.
	Allowed int `json:"allowed"`

	// Denied is the count of policy.denied events.
	Denied int `json:"denied"`

	// DenialRate is the percentage denied (0-100).
	DenialRate float64 `json:"denial_rate"`

	// TopDeniedActions lists the most frequently denied actions (max 10).
	TopDeniedActions []ActionCount `json:"top_denied_actions,omitempty"`
}

// ExecutionHealthSection summarizes execution outcomes.
type ExecutionHealthSection struct {
	// StepsTotal is the total step events.
	StepsTotal int `json:"steps_total"`

	// StepsCompleted is the count of successfully completed steps.
	StepsCompleted int `json:"steps_completed"`

	// StepsFailed is the count of failed steps.
	StepsFailed int `json:"steps_failed"`

	// SuccessRate is the percentage of steps that completed successfully (0-100).
	SuccessRate float64 `json:"success_rate"`

	// AvgDurationMs is the average step duration in milliseconds.
	AvgDurationMs int64 `json:"avg_duration_ms"`

	// P99DurationMs is the 99th percentile step duration in milliseconds.
	P99DurationMs int64 `json:"p99_duration_ms"`

	// FailureBreakdown groups failures by source subsystem.
	FailureBreakdown map[string]int `json:"failure_breakdown,omitempty"`
}

// ChainIntegritySection summarizes HMAC chain verification.
type ChainIntegritySection struct {
	// TotalEvents is the total number of events in the chain.
	TotalEvents int `json:"total_events"`

	// Verified is true if the chain is intact.
	Verified bool `json:"verified"`

	// FirstBreakIndex is -1 if intact, otherwise the index of the first break.
	FirstBreakIndex int `json:"first_break_index"`

	// BreakCount is the number of chain breaks detected.
	BreakCount int `json:"break_count"`
}

// RetentionComplianceSection summarizes retention policy status.
type RetentionComplianceSection struct {
	// PolicyEnabled indicates whether retention is active.
	PolicyEnabled bool `json:"policy_enabled"`

	// DefaultDays is the configured default retention period.
	DefaultDays int `json:"default_days"`

	// OldestEvent is the timestamp of the oldest audit event in the store.
	OldestEvent time.Time `json:"oldest_event,omitempty"`

	// EventsInRange is the count of events within the configured retention window.
	EventsInRange int `json:"events_in_range"`

	// EventsExpired is the count of events past the retention window.
	EventsExpired int `json:"events_expired"`

	// Compliant is true if no expired events remain (i.e., purge is up to date).
	Compliant bool `json:"compliant"`
}

// ErrorPattern represents a recurring error pattern.
type ErrorPattern struct {
	// Error is the error message or pattern.
	Error string `json:"error"`

	// Count is the number of occurrences.
	Count int `json:"count"`

	// Source is the subsystem that produced the error.
	Source Source `json:"source"`
}

// ActionCount counts occurrences of an action.
type ActionCount struct {
	// Action is the action name.
	Action string `json:"action"`

	// Count is the number of occurrences.
	Count int `json:"count"`
}
