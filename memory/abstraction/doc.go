// Package abstraction defines the MemoryManager interface and supporting
// types for the agent's memory system.
//
// This is a contract package: it defines interfaces that are implemented by
// the runtime layer. The primary implementations live in runtime/memory/
// (conversation manager, manager bridge). No types in this package are
// consumed within core itself.
//
// Memory in OpenBotStack is layered:
//   - Short-term: Current conversation context
//   - Long-term: Persisted knowledge (SQLite, optional pgvector)
//   - Entity: Structured facts about known entities
package abstraction
