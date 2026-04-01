package policies

// AllowAllEnforcer is a stub implementation of PolicyEnforcer that permits all actions.
type AllowAllEnforcer struct{}

// NewAllowAllEnforcer creates a new AllowAllEnforcer.
func NewAllowAllEnforcer() *AllowAllEnforcer {
	return &AllowAllEnforcer{}
}

// IsAllowed always returns true, nil.
func (e *AllowAllEnforcer) IsAllowed(action string, context map[string]any) (bool, error) {
	return true, nil
}
