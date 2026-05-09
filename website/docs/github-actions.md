# GitHub Actions

`mdv` ships a composite Action that can analyze dependencies, run audits, diff across refs, and generate `DEPENDENCIES.md` - all within a single workflow step.

## Quick Setup

Add the following job to any existing workflow:

```yaml
# .github/workflows/dependency-check.yml
name: Dependency Check

on:
  push:
    branches: [main]
  pull_request:

jobs:
  deps:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0          # required for --diff

      - uses: devekkx/module-dependency-visualizer/.github/actions/mdv@main
        with:
          audit: "true"
          diff: ${{ github.event.pull_request.base.sha || 'HEAD~1' }}
          generate-docs: "true"
```

---

## Inputs

| Input | Default | Description |
|---|---|---|
| `path` | `.` | Path to the project root (relative to repo root) |
| `format` | `table` | Output format for the analyze step: `table`, `json`, `markdown` |
| `audit` | `false` | Run vulnerability and license audit |
| `diff` | `""` | Show dependency changes since this ref (e.g. `HEAD~1`, `main`) |
| `generate-docs` | `false` | Write `DEPENDENCIES.md` to `<path>/DEPENDENCIES.md` |
| `version` | `latest` | mdv release tag to install (e.g. `v0.3.0`) |

## Outputs

| Output | Description |
|---|---|
| `summary-file` | Path to the written `DEPENDENCIES.md` (only set when `generate-docs: true`) |

---

## Example Workflows

### Analyze with audit on every PR

```yaml
- uses: devekkx/module-dependency-visualizer/.github/actions/mdv@main
  with:
    audit: "true"
```

### Diff against base branch

```yaml
- uses: devekkx/module-dependency-visualizer/.github/actions/mdv@main
  with:
    diff: ${{ github.base_ref }}
```

### Generate and commit DEPENDENCIES.md

```yaml
- uses: devekkx/module-dependency-visualizer/.github/actions/mdv@main
  with:
    generate-docs: "true"
    audit: "true"

- name: Commit DEPENDENCIES.md
  run: |
    git config user.name  "github-actions[bot]"
    git config user.email "github-actions[bot]@users.noreply.github.com"
    git add DEPENDENCIES.md
    git diff --staged --quiet || git commit -m "chore: update DEPENDENCIES.md"
    git push
```

### Pin to a specific mdv version

```yaml
- uses: devekkx/module-dependency-visualizer/.github/actions/mdv@main
  with:
    version: v0.3.0
    audit: "true"
```

---

## Permissions

The Action only reads from the repository - no write permissions needed unless you also push `DEPENDENCIES.md` (see the commit example above).

## Pre-requisites

`fetch-depth: 0` is required on the `actions/checkout` step when using the `diff` input, so that git history is available to compare refs.
