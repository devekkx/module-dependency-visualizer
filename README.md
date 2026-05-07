# mdv - Module Dependency Visualizer

**Understand your project's architecture before it becomes technical debt.**

`mdv` is a language-agnostic CLI that parses `go.mod`, `package.json`, `requirements.txt`, and more into a unified, interactive dependency graph. Phase 1 ships Go support; NPM, Python, and a D3.js web UI follow in subsequent phases.

## Quick Start

```bash
# Install
go install module-dependency-visualizer/cmd/mdv@latest

# Analyze a Go project
mdv analyze .

# Export as Graphviz DOT
mdv export . --format dot > graph.dot
dot -Tsvg graph.dot -o graph.svg

# Export as Mermaid
mdv export . --format mermaid > graph.mmd

# Filter to direct deps only, depth 2
mdv export . --format dot --no-indirect --depth 2
```

## Commands

| Command | Description |
|---------|-------------|
| `mdv analyze <path>` | Parse and emit JSON (schema v1.1.0) |
| `mdv analyze <path> --audit` | Parse + embed vulnerability/license/conflict audit |
| `mdv export <path> --format=dot\|mermaid\|json` | Export to chosen format |
| `mdv audit <path>` | Run standalone security / license / conflict audit |
| `mdv serve <path>` | Launch interactive D3.js web UI with audit panel |
| `mdv version` | Print build metadata |

### Common Flags

```
Global:
  --log-level string   debug, info, warn, error (default "warn")
  --timeout string     Timeout for external commands (default "30s")

analyze / export:
  -o, --output string       Output path ("-" for stdout, default)
  -d, --depth int           Max dependency depth (-1 = unlimited, default)
      --include regex       Keep only modules matching regex (repeatable)
      --exclude regex       Remove modules matching regex (repeatable)
      --no-indirect         Omit indirect (transitive) dependencies
```

## Output Formats

### JSON (Schema v1.0.0)

```json
{
  "schema_version": "1.0.0",
  "project":        { "name": "...", "language": "go", ... },
  "nodes":          [{ "id": "github.com/spf13/cobra@v1.8.0", "kind": "module", ... }],
  "edges":          [{ "from": "...", "to": "...", "kind": "depends_on" }],
  "stats":          { "node_count": 8, "edge_count": 12, "max_depth": 3, "has_cycles": false }
}
```

See [docs/schema-v1.md](docs/schema-v1.md) for the full reference.

### DOT (Graphviz)

Render with `dot`, `neato`, or any Graphviz-compatible tool:

```bash
mdv export . --format dot | dot -Tsvg > deps.svg
```

### Mermaid

Paste the output into GitHub Markdown, Notion, or any Mermaid renderer:

```bash
mdv export . --format mermaid
```

> Mermaid renders reliably up to ~500 nodes. For larger graphs use `--format dot`.

## Build from Source

```bash
git clone https://github.com/emmanuel-kpendo/module-dependency-visualizer
cd module-dependency-visualizer
make build          # produces dist/mdv
make test           # run tests
make test-race      # run with race detector
make cover          # generate coverage report
make lint           # golangci-lint
```

**Requirements:** Go 1.21+

## Roadmap

| Phase | Status | Highlights |
|-------|--------|------------|
| 1 – Foundation | ✅ Done | Go provider, JSON schema, DOT + Mermaid export |
| 2 – Agnostic Layer | ✅ Done | NPM/Yarn/PNPM/Bun, Python/Pip/Poetry |
| 3 – Interactive UI | ✅ Done | Embedded D3.js web server (`mdv serve`) |
| 4 – Intelligence | ✅ Done | OSV vulnerability scan, license audit, conflict detection |
| 5 – Workflow | Planned | Git diff, GitHub Actions, automated docs |

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md).

## License

[MIT](LICENSE)
