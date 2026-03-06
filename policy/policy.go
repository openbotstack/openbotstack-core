// Package policy provides policy and permission interfaces for OpenBotStack.
//
// Policy enforcement happens in the control plane BEFORE execution.
// The runtime blindly executes; the control plane governs.
package policy

// PolicyEnforcer checks whether an action is permitted.
// Full definition deferred to future implementation.
type PolicyEnforcer interface {
	// IsAllowed checks if the action is permitted for the given context.
	IsAllowed(action string, context map[string]any) (bool, error)
}
