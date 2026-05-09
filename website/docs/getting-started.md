# Getting Started

## Installation

### From Source (Go 1.22+)

```bash
go install github.com/devekkx/module-dependency-visualizer/cmd/mdv@latest
```

### Pre-built Binaries

Download the latest release for your platform from the [GitHub Releases](https://github.com/devekkx/module-dependency-visualizer/releases) page.

```bash
# Linux / macOS (example)
curl -fsSL https://github.com/devekkx/module-dependency-visualizer/releases/latest/download/mdv_linux_amd64.tar.gz \
  | tar -xz -C /usr/local/bin mdv
```

## Quick Start

### Analyze a project

```bash
# Print a dependency summary table to stdout
mdv analyze .

# Emit full JSON (schema v1.1)
mdv analyze . --format json

# Run with a vulnerability and license audit
mdv analyze . --audit
```

### Visualize in your browser

```bash
mdv serve .
# → listening address is printed to stdout (port is auto-assigned)

# Or use a fixed port
mdv serve . --port 7777
```

The interactive D3.js UI lets you zoom, pan, and filter nodes. The audit panel shows any CVEs or license issues directly on the graph.

### Export to a file format

```bash
# Graphviz DOT → SVG
mdv export . --format dot | dot -Tsvg -o deps.svg

# Mermaid (paste into GitHub or Notion)
mdv export . --format mermaid

# JSON snapshot
mdv export . --format json -o deps.json
```

### Diff dependencies across git refs

```bash
# Compare current working tree against main
mdv diff . --from main

# Compare two specific refs
mdv diff . --from v1.0.0 --to v2.0.0
```

### Generate a DEPENDENCIES.md report

```bash
mdv docs . --output DEPENDENCIES.md --audit
```

## Requirements

| Requirement | Version |
|---|---|
| Go | 1.22+ (for `go install`) |
| Node / npm | Any (for Node.js projects) |
| Python | 3.8+ (for Python projects) |

Go, Node, and Python providers each shell out to the native toolchain (`go mod graph`, `npm ls`, `pip list`) so the relevant runtime must be on `PATH` when analyzing that project type.
