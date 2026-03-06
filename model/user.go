package model

// User represents a user in the system.
// Full definition deferred to future implementation.
type User struct {
	// ID is a unique identifier for this user.
	ID string

	// TenantID links the user to their tenant.
	TenantID string

	// Name is the display name of the user.
	Name string
}
