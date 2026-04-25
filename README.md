# openbotstack-core

Control Plane for OpenBotStack — interfaces, state machines, skill registry, policy engine, and shared types.

## Role

Core defines the **contracts and orchestration logic** that all other planes depend on. It contains no executable entrypoints, no network calls, and no side effects. Runtime and Apps implement the interfaces defined here.

## Architecture

```
core/
├── ai/               Model provider abstraction, routing, safety, RAG
├── assistant/        Assistant profiles and registry
├── audit/            Audit logging contracts
├── context/          Context assembly for planning
├── control/          Agent control plane
│   ├── assistants/   State machine (Idle → Planning → Executing → ... → Completed)
│   ├── policy/       Policy engine with pattern matching and time-based rules
│   ├── skills/       Skill type definitions
│   └── execution/    Execution state tracking
├── execution/        Execution engine contracts (plans, steps, results)
├── memory/
│   └── abstraction/  Memory manager interface (short-term + long-term)
├── planner/          Execution planner (LLM-based plan generation)
├── registry/skills/  Skill registry, WASM module interface, manifests
└── validation/       Input validation utilities
```

## Key Interfaces

### Skill (registry/skills)

```go
type Skill interface {
    ID() string
    Name() string
    Description() string
    Capabilities() []string
    Execute(ctx context.Context, input []byte) ([]byte, error)
}
```

### Model Provider (ai/providers)

```go
type ModelProvider interface {
    ID() string
    Capabilities() []CapabilityType
    Generate(ctx context.Context, req GenerateRequest) (*GenerateResponse, error)
}

type ModelRouter interface {
    Route(requirements []CapabilityType, constraints ModelConstraints) (ModelProvider, error)
}
```

### Execution Planner (planner)

```go
type ExecutionPlanner interface {
    Plan(ctx context.Context, pCtx *PlannerContext) (*ExecutionPlan, error)
}
```

### Memory Manager (memory/abstraction)

```go
type MemoryManager interface {
    StoreShortTerm(ctx context.Context, key, content string) error
    StoreLongTerm(ctx context.Context, content string, tags []string) error
    RetrieveSimilar(ctx context.Context, query string, limit int) ([]MemoryEntry, error)
    Clear(ctx context.Context, key string) error
}
```

### Agent State Machine (control/assistants)

```
Idle → Planning → Executing → Reflecting → Finalizing → Completed
                   ↑              │
                   └──────────────┘ (bounded by maxReflections)
Any state → Error (on failure)
```

## Dependencies

- `gopkg.in/yaml.v3` — YAML parsing for manifests

Zero external service dependencies. Pure Go, no CGO.

## Build & Test

```bash
make all    # lint + test + build (verification only)
make test   # go test -v -race ./...
make lint   # go vet + staticcheck
make tidy   # go mod tidy
```

## Contract

See [AI_CONTRACT.md](./AI_CONTRACT.md) for architectural boundaries.

**Core MUST NOT contain:** executable entrypoints, tool execution, network calls, side effects, infrastructure-specific code.

## Dependency Chain

```
core ← runtime (Go replace directive)
core ← apps    (Go replace directive)
```
