# ADR 0002: Provider Interface Design

**Status:** Accepted  
**Date:** 2026-05-06

## Context

Phase 1 ships a Go provider. Phases 2-5 add NPM, Python, and more. The interface must be stable enough that adding a new provider requires zero changes to `graph`, `schema`, `exporter`, or `cli`.

## Decision

The `Provider` interface has exactly three methods:

```go
type Provider interface {
    Name() string
    Detect(ctx context.Context, rootPath string) (bool, error)
    Parse(ctx context.Context, rootPath string, opts ParseOptions) (*graph.Graph, Project, error)
}
```

Key choices:
1. **`Detect` is separate from `Parse`** so the registry can probe all providers cheaply before committing to a parse.
2. **`ParseOptions` is a struct** (not variadic options) to keep the call-site readable and make it easy to add fields without breaking callers.
3. **Returns `*graph.Graph`** (not an interface) because the graph domain is stable and returning a concrete type avoids indirection overhead.
4. **Providers live in `internal/provider/<name>/`** and register via `init()`. No reflection or plugin loading.

## Consequences

- **+** Adding Phase 2 providers requires only: new directory, new `init()` call in `cmd/mdv/main.go`. Zero changes to existing packages.
- **+** `Detect` order in the registry is deterministic (registration order), enabling explicit precedence.
- **-** `init()` registration means all providers compile into the binary even if unused. Acceptable for a CLI tool; deferred until a plugin model is needed.

## Alternatives Considered

- **Variadic functional options on `Parse`**: rejected - harder to read at call sites and adds boilerplate for every caller.
- **Single `Analyze(ctx, path)` method**: rejected - separating detect from parse improves error messages (the user gets "unsupported project" instead of a parse error).
