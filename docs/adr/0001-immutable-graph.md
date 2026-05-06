# ADR 0001: Immutable Graph Domain Model

**Status:** Accepted  
**Date:** 2026-05-06

## Context

The dependency graph is computed once per `mdv` invocation and then read by multiple consumers: exporters, filters, the web server (Phase 3), audit enrichment (Phase 4), and diff computation (Phase 5). Shared mutable state across these consumers would require defensive copying or locking.

## Decision

`graph.Graph` exposes only read methods that return **copies** of slices and structs. Mutation is only possible through `graph.Builder`, which is discarded after `Build()` returns. The `Node.Metadata map[string]any` field is shallow-copied on every access to prevent callers from mutating the stored value.

## Consequences

- **+** No locking needed when passing `*Graph` to concurrent goroutines (Phase 3 HTTP server).
- **+** Phase 4 enrichment creates a new `Graph` via `Builder` rather than modifying the original — the audit path cannot corrupt the analysis result.
- **+** Diffing two graphs (Phase 5) is straightforward: no snapshot required.
- **-** Each accessor call allocates a new slice. For very large graphs (>10k nodes) this may be measurable. Mitigation: benchmark-driven optimization if profiling reveals a hot path; the interface is stable so the implementation can be made copy-on-write later without breaking callers.
