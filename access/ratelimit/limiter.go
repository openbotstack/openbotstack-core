// Package ratelimit provides hierarchical rate limiting for OpenBotStack.
//
// Rate limiting is applied at multiple levels:
//   - Tenant: Hard limit for billing/cost control
//   - User: Soft limit for fairness
//   - Skill: Fine-grained control (V2)
package ratelimit

import (
	"context"
	"time"
)

// RateLimitKey identifies the scope for rate limiting.
type RateLimitKey struct {
	TenantID string
	UserID   string // optional, empty = tenant-level only
	SkillID  string // optional, V2
}

// QuotaConfig defines rate limit thresholds.
type QuotaConfig struct {
	// TenantTokensPerMinute is the hard limit for tenant billing.
	TenantTokensPerMinute int64

	// TenantRequestsPerMinute limits request count per tenant.
	TenantRequestsPerMinute int64

	// UserRequestsPerMinute limits request count per user (fairness).
	UserRequestsPerMinute int64

	// UserTokensPerMinute limits tokens per user.
	UserTokensPerMinute int64
}

// RateLimitResult contains the outcome of a rate limit check.
type RateLimitResult struct {
	Allowed    bool
	Remaining  int64
	ResetAt    time.Time
	RetryAfter time.Duration // if not allowed
}

// RateLimiter provides rate limiting operations.
type RateLimiter interface {
	// Allow checks if the request is allowed without consuming quota.
	Allow(ctx context.Context, key RateLimitKey) (*RateLimitResult, error)

	// Consume deducts tokens from the quota.
	// Call this after successful execution.
	Consume(ctx context.Context, key RateLimitKey, tokens int64) error

	// Remaining returns the remaining quota.
	Remaining(ctx context.Context, key RateLimitKey) (int64, error)

	// Reset resets the quota for a key (admin operation).
	Reset(ctx context.Context, key RateLimitKey) error
}

// QuotaStore persists and retrieves quota configurations.
type QuotaStore interface {
	// GetQuota retrieves quota config for a tenant.
	GetQuota(ctx context.Context, tenantID string) (*QuotaConfig, error)

	// SetQuota updates quota config for a tenant.
	SetQuota(ctx context.Context, tenantID string, config *QuotaConfig) error
}
