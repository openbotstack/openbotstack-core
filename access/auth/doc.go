// Package auth defines the User type and related authentication abstractions.
//
// This is a contract package: it defines shared types that are consumed by
// the runtime layer (runtime/api/middleware/ for JWT and API key auth, and
// other runtime packages). The User type is used in auth middleware chains
// but has no consumers within core itself.
package auth
