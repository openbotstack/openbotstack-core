package model

// Tenant represents a tenant in the multi-tenant system.
// Full definition deferred to future implementation.
type Tenant struct {
	// ID is a unique identifier for this tenant.
	ID string

	// Name is the display name of the tenant.
	Name string
}
