# Commands

## Global Flags

These flags are available on every command:

| Flag | Default | Description |
|---|---|---|
| `--log-level` | `warn` | Log verbosity: `debug`, `info`, `warn`, `error` |
| `--timeout` | `30s` | Timeout for external tool invocations |
| `--no-color` | `false` | Disable colour in log output |

---

## `mdv analyse`

Parse a project and emit the dependency graph as JSON. Output is always JSON — use [`mdv export`](#mdv-export) for other formats.

```bash
mdv analyse <path> [flags]
```

**Flags**

| Flag | Default | Description |
|---|---|---|
| `-o, --output` | `-` (stdout) | Write output to a file |
| `-d, --depth` | `-1` (unlimited) | Maximum dependency depth |
| `--include` | - | Keep only modules matching this regex (repeatable) |
| `--exclude` | - | Remove modules matching this regex (repeatable) |
| `--no-indirect` | `false` | Omit transitive dependencies |
| `--audit` | `false` | Embed vulnerability, licence, and conflict audit in output |

**Examples**

```bash
# JSON to stdout
mdv analyse .

# Save snapshot to file
mdv analyse . -o snapshot.json

# Direct deps only, depth 2, with audit
mdv analyse . --no-indirect --depth 2 --audit

# Filter to a specific vendor
mdv analyse . --include "^github.com/spf13"
```

---

## `mdv export`

Export the dependency graph to a renderable format.

```bash
mdv export <path> --format <fmt> [flags]
```

**Flags**

| Flag | Default | Description |
|---|---|---|
| `--format` | - | **Required.** `dot`, `mermaid`, or `json` |
| `-o, --output` | `-` (stdout) | Write output to a file |
| `-d, --depth` | `-1` | Maximum dependency depth |
| `--include` | - | Include filter (regex, repeatable) |
| `--exclude` | - | Exclude filter (regex, repeatable) |
| `--no-indirect` | `false` | Omit indirect dependencies |

**Examples**

```bash
# Render as SVG via Graphviz
mdv export . --format dot | dot -Tsvg -o deps.svg

# Mermaid diagram
mdv export . --format mermaid > deps.mmd

# JSON snapshot to file
mdv export . --format json -o deps.json

# Direct deps only
mdv export . --format dot --no-indirect
```

---

## `mdv audit`

Run a standalone security, license, and version-conflict audit.

```bash
mdv audit <path>
```

The audit checks:
- **CVEs** via the [OSV database](https://osv.dev) for each dependency version
- **License compatibility** - flags non-permissive or conflicting licenses
- **Version conflicts** - detects when the same package is required at multiple incompatible versions

**Example output**

```
AUDIT RESULTS
=============
Vulnerabilities  2 critical, 1 high
License issues   1 (GPL-3.0 in MIT project)
Conflicts        0

[CRITICAL] github.com/foo/bar@v1.2.3
  CVE-2024-12345 - Remote code execution via malformed input
  Fix: upgrade to v1.2.4
```

---

## `mdv serve`

Launch an interactive D3.js dependency graph in your browser.

```bash
mdv serve <path> [flags]
```

**Flags**

| Flag | Default | Description |
|---|---|---|
| `--port`, `-p` | `0` (auto-assign) | HTTP port to listen on |
| `--audit` | `false` | Include audit data in the UI |
| `--no-browser` | `false` | Skip opening the browser automatically |

**Example**

```bash
# Port is auto-assigned - the chosen address is printed to stdout
mdv serve . --audit

# Use a fixed port
mdv serve . --port 7777 --audit
```

The web UI supports:
- Zoom and pan
- Node search and filtering
- Depth sliders
- Live audit panel showing CVEs and license issues

---

## `mdv diff`

Show dependency changes between two git refs.

```bash
mdv diff <path> [flags]
```

**Flags**

| Flag | Default | Description |
|---|---|---|
| `--from` | - | **Required.** Base git ref (branch, tag, or commit SHA) |
| `--to` | `HEAD` | Target ref to compare against |

**Example output**

```
DEPENDENCY DIFF: main → HEAD
============================
+ github.com/new/pkg@v1.0.0        ADDED
- github.com/old/pkg@v2.3.0        REMOVED
~ github.com/spf13/cobra            v1.7.0 → v1.8.0   UPGRADED
```

**Examples**

```bash
# What changed since main?
mdv diff . --from main

# Compare two release tags
mdv diff . --from v1.0.0 --to v2.0.0

# Compare a specific commit
mdv diff . --from abc1234
```

---

## `mdv docs`

Generate a `DEPENDENCIES.md` file documenting all dependencies.

```bash
mdv docs <path> [flags]
```

**Flags**

| Flag | Default | Description |
|---|---|---|
| `-o, --output` | `DEPENDENCIES.md` | Output file path |
| `--audit` | `false` | Include vulnerability and license data |

**Example**

```bash
mdv docs . --output DEPENDENCIES.md --audit
```

The generated file includes a dependency table, stats, and optional audit findings - suitable for committing to the repository or attaching to a release.

---

## `mdv version`

Print build metadata.

```bash
mdv version
```

Output includes the version string, git commit hash, and build date when built with the release Makefile target.
