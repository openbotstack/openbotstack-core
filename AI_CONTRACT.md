# AI RULES — READ FIRST

This file defines mandatory rules for AI coding tools.
All instructions in this file are authoritative for this repository.

# REPOSITORY: openbotstack-core

## ROLE:
This repository defines the CONTROL PLANE of OpenBotStack.

## IT MAY CONTAIN:
- Tenant, User, AssistantProfile models
- Skill definitions and registry
- Policy and permission checks
- Agent state machine (plan → execute → reflect → finalize)
- Memory interfaces (Milvus abstraction only, no direct DB logic)
- Audit event schemas and decision records

## IT MUST NOT:
- Execute real tools or external side effects
- Contain infrastructure-specific code
- Depend on runtime implementations
- Perform network calls to third-party services

> **Exception (ADR-011):** Model Plane provider adapters (e.g., OpenAICompatibleProvider, ClaudeProvider)
> may perform network calls to third-party LLM services as part of their adapter role.
> This is an intentional architectural decision documented in ADR-011.

## DESIGN RULES:
- Prefer interfaces over implementations
- All state transitions must be explicit
- Reflection logic must be bounded and testable
- No business logic hidden in prompts

If functionality seems execution-related, it belongs in openbotstack-runtime.

> This repo MUST NOT contain any executable entrypoint.