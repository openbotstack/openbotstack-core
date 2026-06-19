package auth

// User represents a user in the system.
// Full definition deferred to future implementation.
type User struct {
	// ID is a unique identifier for this user.
	ID string

	// TenantID links the user to their tenant.
	TenantID string

	// Name is the display name of the user.
	Name string

	// Role is the authorization role ("admin" or "member"). Carried on the User so a
	// single context value carries identity + authorization (previously role lived in a
	// separate context key, making the two asymmetric across API-Key vs JWT paths).
	Role string
}

// IsAdmin reports whether the user holds the admin role.
func (u *User) IsAdmin() bool { return u != nil && u.Role == "admin" }
