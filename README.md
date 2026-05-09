# mdv - Module Dependency Visualizer

**Understand your project's architecture before it becomes technical debt.**

`mdv` is a language-agnostic CLI that parses `go.mod`, `package.json`, `requirements.txt`, and more into a unified, interactive dependency graph. It ships with a security auditor, a D3.js web UI, a diff engine, and a GitHub Actions composite action - covering the full dependency lifecycle in a single binary.

[![CI](https://github.com/devekkx/module-dependency-visualizer/actions/workflows/ci.yml/badge.svg)](https://github.com/devekkx/module-dependency-visualizer/actions/workflows/ci.yml)
[![Dependency Check](https://github.com/devekkx/module-dependency-visualizer/actions/workflows/dependency-check.yml/badge.svg)](https://github.com/devekkx/module-dependency-visualizer/actions/workflows/dependency-check.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/devekkx/module-dependency-visualizer)](https://goreportcard.com/report/github.com/devekkx/module-dependency-visualizer)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

---

## Table of Contents

- [Features](#features)
- [Installation](#installation)
- [Quick Start](#quick-start)
- [Commands](#commands)
- [Output Formats](#output-formats)
- [GitHub Actions](#github-actions)
- [Build from Source](#build-from-source)
- [Project Structure](#project-structure)
- [Branch Strategy](#branch-strategy)
- [CI/CD Pipelines](#cicd-pipelines)
- [Roadmap](#roadmap)
- [Contributing](#contributing)
- [License](#license)

---

## Features

| Feature | Description |
|---|---|
| **Language support** | **Go** (`go.mod`), **Node** (npm, Yarn v1/Berry, pnpm, Bun), **Python** (pip, Poetry, Pipenv, uv, pyproject.toml) |
| **Security audit** | Queries the [OSV](https://osv.dev) database for known CVEs per dependency version |
| **License audit** | Detects non-permissive and conflicting licenses across the dependency tree |
| **Conflict detection** | Flags the same package required at multiple incompatible versions |
| **Interactive UI** | Embedded D3.js graph with zoom, pan, depth filters, and a live audit panel |
| **Multiple export formats** | JSON (schema v1.1), Graphviz DOT, Mermaid |
| **Dependency diff** | Compares snapshots across any two git refs |
| **Doc generation** | Writes a `DEPENDENCIES.md` report, suitable for committing alongside releases |
| **GitHub Actions** | Drop-in composite action - analyse, audit, diff, and generate docs on every PR |

---

## Installation

### go install (recommended)

Requires Go 1.22+.

```bash
go install github.com/devekkx/module-dependency-visualizer/cmd/mdv@latest
```

### Pre-built binaries

Download the archive for your platform from the [Releases](https://github.com/devekkx/module-dependency-visualizer/releases) page, then extract and move to `$PATH`:

```bash
# Linux (amd64)
curl -fsSL https://github.com/devekkx/module-dependency-visualizer/releases/latest/download/mdv_latest_linux_amd64.tar.gz \
  | tar -xz -C /usr/local/bin mdv

# macOS (arm64 / Apple Silicon)
curl -fsSL https://github.com/devekkx/module-dependency-visualizer/releases/latest/download/mdv_latest_darwin_arm64.tar.gz \
  | tar -xz -C /usr/local/bin mdv
```

Verify checksums with the `checksums.txt` file included in each release.

### Docker

No installation required - mount your project directory and run:

```bash
# Analyze
docker run --rm -v $(pwd):/work ghcr.io/devekkx/mdv analyze /work

# Export as DOT
docker run --rm -v $(pwd):/work ghcr.io/devekkx/mdv export /work --format dot

# Audit
docker run --rm -v $(pwd):/work ghcr.io/devekkx/mdv audit /work

# Serve (use --no-browser since there is no browser in the container)
docker run --rm -v $(pwd):/work -p 7777:7777 ghcr.io/devekkx/mdv serve /work --port 7777 --no-browser
```

Images are published to `ghcr.io/devekkx/mdv` for `linux/amd64` and `linux/arm64`.

---

## Quick Start

```bash
# Summarise dependencies in your terminal
mdv analyze .

# Launch the interactive web UI (port is auto-assigned, shown in output)
mdv serve .

# Use a fixed port
mdv serve . --port 7777

# Export a Graphviz SVG
mdv export . --format dot | dot -Tsvg -o deps.svg

# Paste a Mermaid diagram into GitHub or Notion
mdv export . --format mermaid

# Run a full security & license audit
mdv audit .

# See what changed since main
mdv diff . --from main

# Generate DEPENDENCIES.md with audit data
mdv docs . --output DEPENDENCIES.md --audit
```

---

## Commands

### `mdv analyze`

Parse a project and emit a dependency summary.

```
mdv analyze <path> [flags]
```

| Flag | Default | Description |
|---|---|---|
| `-o, --output` | `-` (stdout) | Write output to file |
| `--format` | `table` | `table`, `json`, or `markdown` |
| `-d, --depth` | `-1` (unlimited) | Maximum dependency depth |
| `--include` | - | Keep modules matching regex (repeatable) |
| `--exclude` | - | Remove modules matching regex (repeatable) |
| `--no-indirect` | `false` | Omit transitive dependencies |
| `--audit` | `false` | Embed vulnerability/license/conflict data |

### `mdv export`

Export the graph to a renderable format.

```
mdv export <path> --format <fmt> [flags]
```

`--format` is required: `dot`, `mermaid`, or `json`. Accepts the same depth/filter flags as `analyze`.

### `mdv audit`

Run a standalone security, license, and version-conflict audit.

```
mdv audit <path>
```

Checks each dependency against the OSV database, flags non-permissive licenses, and detects packages required at multiple incompatible versions.

### `mdv serve`

Launch an interactive D3.js dependency graph in your browser.

```
mdv serve <path> [--port <n>] [--audit]
```

### `mdv diff`

Show dependency changes between two git refs.

```
mdv diff <path> --from <ref> [--to <ref>]
```

`--to` defaults to `HEAD`. Outputs added, removed, and version-bumped dependencies.

### `mdv docs`

Generate a `DEPENDENCIES.md` documentation file.

```
mdv docs <path> [-o DEPENDENCIES.md] [--audit]
```

### `mdv version`

Print version, commit hash, and build date.

### Global flags

| Flag | Default | Description |
|---|---|---|
| `--log-level` | `warn` | `debug`, `info`, `warn`, `error` |
| `--timeout` | `30s` | Timeout for external tool calls |

---

## Output Formats

### JSON - Schema v1.1

The canonical, machine-readable graph. Powers all other exporters.

```json
{
  "schema_version": "1.1.0",
  "generated_at":   "2024-11-01T10:00:00Z",
  "project":  { "name": "github.com/example/app", "language": "go", ... },
  "nodes":    [{ "id": "github.com/spf13/cobra@v1.8.0", "kind": "module", "indirect": false, ... }],
  "edges":    [{ "from": "github.com/example/app@", "to": "github.com/spf13/cobra@v1.8.0", "kind": "depends_on" }],
  "stats":    { "node_count": 8, "edge_count": 12, "max_depth": 3, "has_cycles": false }
}
```

See [`docs/schema-v1.md`](docs/schema-v1.md) for the complete field reference.

### DOT - Graphviz

```bash
mdv export . --format dot | dot -Tsvg -o deps.svg
mdv export . --format dot | dot -Tpng -o deps.png
```

Best for large graphs (hundreds to thousands of nodes).

### Mermaid

```bash
mdv export . --format mermaid
```

Paste directly into GitHub Markdown, Notion, or Confluence. Renders reliably up to ~500 nodes; use DOT for larger graphs.

---

## GitHub Actions

A composite action is available at `.github/actions/mdv`.

```yaml
- uses: devekkx/module-dependency-visualizer/.github/actions/mdv@main
  with:
    audit: "true"
    diff: ${{ github.event.pull_request.base.sha }}
    generate-docs: "true"
```

**Inputs**

| Input | Default | Description |
|---|---|---|
| `path` | `.` | Path to the project root |
| `format` | `table` | Output format: `table`, `json`, `markdown` |
| `audit` | `false` | Run vulnerability and license audit |
| `diff` | `""` | Show dependency changes since this ref |
| `generate-docs` | `false` | Write `DEPENDENCIES.md` |
| `version` | `latest` | mdv release tag to pin (e.g. `v0.3.0`) |

**Outputs**

| Output | Description |
|---|---|
| `summary-file` | Path to `DEPENDENCIES.md` (when `generate-docs: true`) |

---

## Build from Source

**Requirements:** Go 1.21+

```bash
git clone https://github.com/devekkx/module-dependency-visualizer.git
cd module-dependency-visualizer
```

| Target | Description |
|---|---|
| `make build` | Compile binary → `dist/mdv` |
| `make test` | Run test suite |
| `make test-race` | Run with race detector |
| `make cover` | Tests + HTML coverage report |
| `make lint` | golangci-lint |
| `make vet` | go vet |
| `make gosec` | gosec security scanner |
| `make tidy` | go mod tidy + verify |
| `make clean` | Remove `dist/` and `coverage.out` |

---

## Project Structure

```
module-dependency-visualizer/
├── cmd/mdv/                   # Entry point
├── internal/
│   ├── audit/                 # OSV, license, and conflict checks
│   ├── cli/                   # Cobra command definitions
│   ├── config/                # Build metadata (version, commit, date)
│   ├── diff/                  # Dependency diff engine
│   ├── docgen/                # DEPENDENCIES.md generator
│   ├── exporter/
│   │   ├── dot/               # Graphviz DOT exporter
│   │   ├── jsonexp/           # JSON exporter
│   │   └── mermaid/           # Mermaid exporter
│   ├── graph/                 # Immutable graph model, builder, filters, traversal
│   ├── logging/               # Structured logger
│   ├── provider/
│   │   ├── golang/            # go mod graph provider
│   │   ├── node/              # npm/yarn/pnpm/bun provider
│   │   └── python/            # pip/poetry/pyproject provider
│   ├── schema/                # JSON schema encode/decode (v1.1)
│   └── server/                # Embedded D3.js HTTP server
├── docs/
│   ├── adr/                   # Architecture Decision Records
│   └── schema-v1.md           # JSON schema reference
├── website/                   # VitePress documentation site (→ Vercel)
│   └── docs/
│       ├── .vitepress/config.ts
│       ├── index.md
│       ├── getting-started.md
│       ├── commands.md
│       ├── formats.md
│       ├── schema.md
│       ├── github-actions.md
│       └── contributing.md
├── .github/
│   ├── actions/mdv/           # Composite GitHub Action
│   └── workflows/             # CI/CD pipelines (see below)
├── Makefile
├── go.mod
└── vercel.json                # Docs site deployment config
```

---

## Branch Strategy

The project follows a three-tier promotion model:

```
feature/*  ──PR──▶  testing     (integration branch - CI runs here)
testing    ──PR──▶  develop     (stable dev - dep checks run here)
develop    ──PR──▶  production  (release-ready - full gate runs here)
production ──tag──▶  v*.*.*    (triggers GitHub Release)
```

| Branch | Purpose | Who merges here |
|---|---|---|
| `feature/*` | Individual feature or bug-fix branches | Developers |
| `testing` | Integration point; all code must pass CI before advancing | Developers via PR |
| `develop` | Stable development branch; dependency-clean and security-scanned | `testing` via PR |
| `production` | Always release-ready; gates are strictest | `develop` via PR |

**Day-to-day flow:**

1. Branch off `testing`: `git checkout -b feature/my-thing testing`
2. Open a PR to `testing` - CI runs unit tests, lint, and build.
3. Once merged, open a PR from `testing` → `develop` - dependency audit and security scan run.
4. When ready to ship, open a PR from `develop` → `production` - the full gate runs (tests, lint, security, cross-platform build, docs).
5. Tag the merge commit on `production`: `git tag v1.2.3 && git push --tags` - the release pipeline publishes binaries to GitHub Releases.

---

## CI/CD Pipelines

All pipelines live in [`.github/workflows/`](.github/workflows/).

### `ci.yml` - CI (PR → `testing`)

Runs on every pull request targeting `testing`. Stale runs are cancelled automatically on force-push.

| Job | What it does |
|---|---|
| **Test** (matrix: Go 1.22, 1.23, stable) | `go mod tidy` drift check, `go vet`, tests with race detector, 80% coverage gate |
| **Lint** | golangci-lint with the project's `.golangci.yml` ruleset |
| **Build** | `make build` - verifies the binary compiles; uploads artifact for 3 days |

The coverage report artifact (`coverage.out`) is uploaded from the `stable` matrix leg and kept for 7 days.

### `dependency-check.yml` - Dependency Check (PR → `develop`)

Runs on every pull request targeting `develop`. Posts a single auto-updating comment on the PR.

| Job | What it does |
|---|---|
| **Dependency Audit** | `mdv analyze --audit`, dependency diff against the base SHA, `govulncheck` (informational), generates `DEPENDENCIES.md`, posts/updates PR comment |
| **Security Scan** | `gosec` in SARIF mode - results uploaded to GitHub Security tab |

### `promote.yml` - Promote to Production (PR → `production`)

The strictest gate. All jobs must pass before merging to `production`.

| Job | What it does |
|---|---|
| **Full Test Suite** | Same as CI but runs once, not matrix |
| **Lint** | golangci-lint |
| **Security Gate** | `govulncheck` (**blocking** - non-zero exit fails the PR), `gosec` SARIF |
| **Cross-platform Build** (5-way matrix) | `linux/amd64`, `linux/arm64`, `darwin/amd64`, `darwin/arm64`, `windows/amd64` - all with `CGO_ENABLED=0` |
| **Docs Build** | `npm ci && npm run build` in `website/` - verifies the docs site compiles |

### `release.yml` - Release (push tag `v*.*.*`)

Triggered by pushing a semver tag to `production`. Builds static binaries for all platforms, packages them with `README.md` and `LICENSE`, generates `checksums.txt`, and publishes a GitHub Release with auto-generated release notes.

```bash
# Cut a release from production
git checkout production
git tag v1.2.3
git push origin v1.2.3
```

Artifacts produced per release:

```
mdv_v1.2.3_linux_amd64.tar.gz
mdv_v1.2.3_linux_arm64.tar.gz
mdv_v1.2.3_darwin_amd64.tar.gz
mdv_v1.2.3_darwin_arm64.tar.gz
mdv_v1.2.3_windows_amd64.zip
checksums.txt
```

### `docs.yml` - Docs (PR touching `website/**`)

Validates the VitePress build on any PR that modifies the `website/` directory. Uploads the built site as an artifact for preview.

---

## Roadmap

| Phase | Status | Highlights |
|---|---|---|
| 1 - Foundation | ✅ Done | Go provider, JSON schema v1, DOT + Mermaid export |
| 2 - Agnostic Layer | ✅ Done | NPM/Yarn/PNPM/Bun, Python/pip/Poetry providers |
| 3 - Interactive UI | ✅ Done | Embedded D3.js web server (`mdv serve`) |
| 4 - Intelligence | ✅ Done | OSV vulnerability scan, license audit, conflict detection |
| 5 - Workflow | ✅ Done | `mdv diff`, `mdv docs`, GitHub Actions composite action |

---

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for the full guide.

Quick summary:

1. Branch off `testing`
2. Make focused changes and write tests (80% coverage required)
3. Open a PR to `testing` - CI must pass
4. Follow [Conventional Commits](https://www.conventionalcommits.org/): `feat:`, `fix:`, `docs:`, `refactor:`, `test:`, `chore:`

---

## License

[MIT](LICENSE)
