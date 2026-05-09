# Supported Languages

`mdv` currently supports three languages. Each has a dedicated provider that auto-detects the project type from manifest and lock files - no configuration required.

---

## Go

**Detected by:** `go.mod`

**Resolved via:** `go mod graph`

| Requirement | Version |
|---|---|
| Go toolchain on `PATH` | 1.21+ |

```bash
mdv analyze /path/to/go-project
```

`go mod graph` is called internally to produce the full module graph including indirect dependencies and `replace` directives.

---

## Node.js

**Detected by:** `package.json`

The package manager is auto-detected from lock files. No flag or config needed.

| Package manager | Detection file | Resolution method |
|---|---|---|
| npm | `package-lock.json` | Parsed directly from lockfile |
| Yarn v1 | `yarn.lock` (no `.yarnrc.yml`) | Parsed directly from lockfile |
| Yarn Berry (v2+) | `yarn.lock` + `.yarnrc.yml` | Parsed directly from lockfile |
| pnpm | `pnpm-lock.yaml` | Parsed directly from lockfile |
| Bun | `bun.lockb` or `bun.lock` | Parsed directly from lockfile |
| (none) | `package.json` only | Falls back to `npm ls` |

| Requirement | Version |
|---|---|
| Node.js on `PATH` | Any (only needed for `npm ls` fallback) |

```bash
mdv analyze /path/to/node-project
```

---

## Python

**Detected by:** any of the files listed below.

The tool is auto-detected from lock and config files in priority order.

| Tool | Detection file | Resolution method |
|---|---|---|
| Poetry | `poetry.lock` | Parsed directly from lockfile |
| Pipenv | `Pipfile.lock` | Parsed directly from lockfile |
| uv | `uv.lock` | Parsed directly from lockfile |
| pip | `requirements.txt` | Parsed directly from file |
| PEP 621 / setuptools | `pyproject.toml` | Parsed directly from file |
| setuptools (legacy) | `setup.py` / `setup.cfg` | Parsed directly from file |

| Requirement | Version |
|---|---|
| Python on `PATH` | 3.8+ |

```bash
mdv analyze /path/to/python-project
```

---

## Detection priority

When a project directory contains files from multiple languages (e.g. a Go CLI with a Node.js frontend), `mdv` runs all matching providers and merges the results into a single graph.

If you want to restrict analysis to a specific provider, use `--include` to filter by module path patterns:

```bash
# Only show Go modules
mdv analyze . --include "^github.com"

# Only show Python packages
mdv analyze . --include "^pypi:"
```
