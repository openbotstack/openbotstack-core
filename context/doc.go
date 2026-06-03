// Package context defines the ContextAssembler interface and supporting
// types for assembling LLM conversation context from assistant profiles,
// memory, and user requests.
//
// This is a contract package: it defines interfaces that are implemented by
// the runtime layer. The primary implementation lives in runtime/context/
// (RuntimeContextAssembler). No types in this package are consumed within
// core itself.
//
// Multiple assembly strategies exist or are planned:
//   - RuntimeContextAssembler (keyword-based, current)
//   - RAG-enhanced assembler (embedding retrieval, ADR-017)
package context
