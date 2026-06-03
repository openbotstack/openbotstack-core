// Package ratelimit defines the RateLimiter and QuotaStore interfaces and
// supporting types for hierarchical rate limiting.
//
// This is a contract package: it defines interfaces that are implemented by
// the runtime layer. The primary implementations live in runtime/ratelimit/
// (SQLiteLimiter, SQLiteQuotaStore) and are wired into the API middleware
// (runtime/api/middleware/). No types in this package are consumed within
// core itself.
//
// Rate limiting is applied at multiple levels:
//   - Tenant: Hard limit for billing/cost control
//   - User: Soft limit for fairness
//   - Skill: Fine-grained control (V2)
package ratelimit
