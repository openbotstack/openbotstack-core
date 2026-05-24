package execution

import (
	"context"
	"time"
)

// ApprovalStatus represents the current state of an approval request.
type ApprovalStatus string

const (
	ApprovalPending  ApprovalStatus = "pending"
	ApprovalApproved ApprovalStatus = "approved"
	ApprovalDenied   ApprovalStatus = "denied"
	ApprovalExpired  ApprovalStatus = "expired"
)

// ApprovalRequest is a request for human approval of a critical step.
type ApprovalRequest struct {
	ID          string         `json:"id"`
	StepName    string         `json:"step_name"`
	StepID      string         `json:"step_id"`
	ExecutionID string         `json:"execution_id"`
	TenantID    string         `json:"tenant_id"`
	RiskLevel   string         `json:"risk_level"`
	Reason      string         `json:"reason"`
	Status      ApprovalStatus `json:"status"`
	ApproverID  string         `json:"approver_id,omitempty"`
	DenyReason  string         `json:"deny_reason,omitempty"`
	CreatedAt   time.Time      `json:"created_at"`
	ResolvedAt  *time.Time     `json:"resolved_at,omitempty"`
	ExpiresAt   time.Time      `json:"expires_at"`
}

// ApprovalGateway manages human approval for critical steps.
type ApprovalGateway interface {
	// RequestApproval creates a new approval request and returns it.
	// The caller is responsible for polling GetApproval to wait for resolution.
	RequestApproval(ctx context.Context, req *ApprovalRequest) (*ApprovalRequest, error)
	// GetApproval retrieves an approval request by ID.
	GetApproval(id string) (*ApprovalRequest, error)
	// Approve marks an approval as approved.
	Approve(id, approverID string) error
	// Deny marks an approval as denied.
	Deny(id, approverID, reason string) error
	// ListPending returns all pending approvals, optionally filtered by tenant.
	ListPending(tenantID string) []ApprovalRequest
}
