package audit

import (
	"encoding/json"
	"time"
)

// EventType categorizes audit events by domain.
type EventType string

const (
	// Execution events — core step/skill/tool lifecycle
	EventStepStarted   EventType = "execution.step.started"
	EventStepCompleted EventType = "execution.step.completed"
	EventStepFailed    EventType = "execution.step.failed"

	// Reasoning events — LLM planning and reasoning loop
	EventReasoningTurn EventType = "execution.reasoning.turn"

	// Policy events — permission and safety decisions
	EventPolicyAllowed EventType = "policy.allowed"
	EventPolicyDenied  EventType = "policy.denied"

	// Admin events — CRUD operations on system resources
	EventAdminProviderCreated EventType = "admin.provider.created"
	EventAdminProviderUpdated EventType = "admin.provider.updated"
	EventAdminKeyCreated      EventType = "admin.key.created"
	EventAdminKeyDeleted      EventType = "admin.key.deleted"
	EventAdminSkillDisabled   EventType = "admin.skill.disabled"
	EventAdminSkillEnabled    EventType = "admin.skill.enabled"
	EventAdminSkillReloaded   EventType = "admin.skill.reloaded"

	// System events — health, startup, shutdown
	EventSystemStarted  EventType = "system.started"
	EventSystemStopped  EventType = "system.stopped"
	EventSystemDegraded EventType = "system.degraded"
)

// Severity classifies audit event importance.
type Severity string

const (
	SeverityInfo  Severity = "info"
	SeverityWarn  Severity = "warn"
	SeverityError Severity = "error"
)

// Source identifies which subsystem produced the event.
type Source string

const (
	SourceExecutor  Source = "executor"
	SourcePolicy    Source = "policy"
	SourceAdmin     Source = "admin_api"
	SourceReasoning Source = "reasoning_loop"
	SourceSystem    Source = "system"
)

// AuditEnvelope is a standardized view of an AuditEvent for external consumption.
// It normalizes all audit sources into a single queryable schema without
// modifying the underlying storage or the AuditEvent struct.
type AuditEnvelope struct {
	EventID     string            `json:"event_id"`
	TraceID     string            `json:"trace_id,omitempty"`
	Timestamp   time.Time         `json:"timestamp"`
	TenantID    string            `json:"tenant_id"`
	UserID      string            `json:"user_id,omitempty"`
	ExecutionID string            `json:"execution_id,omitempty"`
	SessionID   string            `json:"session_id,omitempty"`
	StepID      string            `json:"step_id,omitempty"`
	StepName    string            `json:"step_name,omitempty"`
	EventType   EventType         `json:"event_type"`
	Severity    Severity          `json:"severity"`
	Source      Source            `json:"source"`
	Action      string            `json:"action"`
	Resource    string            `json:"resource,omitempty"`
	Outcome     string            `json:"outcome"`
	DurationMs  int64             `json:"duration_ms"`
	Error       string            `json:"error,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
	Payload     json.RawMessage   `json:"payload,omitempty"`
}

// ToEnvelope converts an AuditEvent to a standardized AuditEnvelope.
// It infers EventType, Severity, and Source from the event's fields.
func (e AuditEvent) ToEnvelope() AuditEnvelope {
	env := AuditEnvelope{
		EventID:     e.ID,
		TraceID:     e.TraceID,
		Timestamp:   e.Timestamp,
		TenantID:    e.TenantID,
		UserID:      e.UserID,
		ExecutionID: e.RequestID,
		SessionID:   e.Metadata["session_id"],
		StepID:      e.StepID,
		StepName:    e.StepName,
		EventType:   inferEventType(e),
		Severity:    inferSeverity(e),
		Source:      resolveSource(e),
		Action:      e.Action,
		Resource:    e.Resource,
		Outcome:     e.Outcome,
		DurationMs:  e.Duration.Milliseconds(),
		Error:       e.Error,
		Metadata:    e.Metadata,
	}

	if len(e.ToolInput) > 0 || e.ToolOutput != nil {
		payload := make(map[string]any)
		if len(e.ToolInput) > 0 {
			payload["tool_input"] = e.ToolInput
		}
		if e.ToolOutput != nil {
			payload["tool_output"] = e.ToolOutput
		}
		if raw, err := json.Marshal(payload); err == nil {
			env.Payload = raw
		}
	}

	return env
}

func inferEventType(e AuditEvent) EventType {
	if e.Action != "" {
		switch {
		case e.Action == "policy.enforce" || e.Action == "policy.check":
			if e.Outcome == "denied" {
				return EventPolicyDenied
			}
			return EventPolicyAllowed
		case e.StepID != "" && e.Status == "started":
			return EventStepStarted
		case e.StepID != "" && e.Status == "completed":
			return EventStepCompleted
		case e.StepID != "" && e.Status == "failed":
			return EventStepFailed
		}
	}
	if e.Status == "failed" || e.Error != "" {
		return EventStepFailed
	}
	if e.StepID != "" {
		return EventStepCompleted
	}
	return EventStepCompleted
}

func inferSeverity(e AuditEvent) Severity {
	if e.Status == "failed" || e.Error != "" || e.Outcome == "failure" || e.Outcome == "timeout" {
		return SeverityError
	}
	if e.Outcome == "denied" {
		return SeverityWarn
	}
	return SeverityInfo
}

// resolveSource returns the explicit Source if set, otherwise infers it.
func resolveSource(e AuditEvent) Source {
	if e.Source != "" {
		return e.Source
	}
	return inferSource(e)
}

func inferSource(e AuditEvent) Source {
	if e.ActorID == "admin" || e.Action == "admin" {
		return SourceAdmin
	}
	if e.Action == "policy.enforce" || e.Action == "policy.check" {
		return SourcePolicy
	}
	if e.StepType == "llm" {
		return SourceReasoning
	}
	if e.StepID != "" || e.StepName != "" {
		return SourceExecutor
	}
	return SourceExecutor
}
