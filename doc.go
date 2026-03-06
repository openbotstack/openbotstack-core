// Package openbotstack provides the control plane interfaces and models for OpenBotStack.
//
// OpenBotStack is an open-source, enterprise-oriented AI execution stack that provides
// users with a "perceived persistent AI assistant" while keeping execution ephemeral,
// governed, and auditable.
//
// This module (openbotstack-core) defines:
//   - Skill definitions and registry
//   - Agent state machine and lifecycle
//   - Memory abstraction interfaces
//   - Policy and permission interfaces
//   - Audit event schemas
//   - Core domain models (Tenant, User, AssistantProfile)
//
// Key Principles:
//   - Request-scoped, ephemeral execution (agents are NOT persistent processes)
//   - Control plane only (no tool execution, no network calls, no side effects)
//   - Stateless between requests (all state is persisted externally)
//   - Bounded reflection (no infinite loops, no unbounded self-modification)
//   - Auditable and deterministic (all decisions must be traceable)
//   - Interfaces over implementations (core defines contracts, runtime implements)
//
// For execution-related functionality, see github.com/openbotstack/openbotstack-runtime.
package openbotstack
