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

### Docker

No Go installation required. Mount your project directory and run:

```bash
docker pull ghcr.io/devekkx/mdv:latest

# Analyse
docker run --rm -v $(pwd):/work ghcr.io/devekkx/mdv analyse /work

# Export as DOT
docker run --rm -v $(pwd):/work ghcr.io/devekkx/mdv export /work --format dot

# Audit
docker run --rm -v $(pwd):/work ghcr.io/devekkx/mdv audit /work

# Serve - use --no-browser and map a fixed port
docker run --rm -v $(pwd):/work -p 7777:7777 \
  ghcr.io/devekkx/mdv serve /work --port 7777 --no-browser
# → open http://localhost:7777 in your browser
```

::: tip
Always use `--no-browser` when running `mdv serve` inside Docker - the container has no browser to open.
:::

Images are available for `linux/amd64` and `linux/arm64`.

## Quick Start

### Analyse a project

```bash
# Print a dependency summary table to stdout
mdv analyse .

# Emit full JSON (schema v1.1)
mdv analyse . --format json

# Run with a vulnerability and license audit
mdv analyse . --audit
```

### Visualise in your browser

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
