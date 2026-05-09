# Contributing to mdv

Thank you for your interest in contributing! This document covers everything you need to get started.

---

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [Getting Started](#getting-started)
- [Branch Strategy](#branch-strategy)
- [Development Workflow](#development-workflow)
- [Commit Messages](#commit-messages)
- [Pull Request Guidelines](#pull-request-guidelines)
- [Versioning & Releasing](#versioning--releasing)
- [Reporting Bugs](#reporting-bugs)
- [Suggesting Features](#suggesting-features)

---

## Code of Conduct

Be respectful and constructive. Harassment or exclusionary behaviour of any kind will not be tolerated.

---

## Getting Started

**Prerequisites:** Go 1.22+

```bash
git clone https://github.com/devekkx/module-dependency-visualizer.git
cd module-dependency-visualizer
go build ./...
go test ./...
```

---

## Branch Strategy

```
feature/*  ──PR──▶  testing     (CI: unit tests, lint, build)
testing    ──PR──▶  develop     (CI: dependency audit, security scan)
develop    ──PR──▶  production  (CI: full gate, cross-platform build)
production ──tag──▶  v*.*.*    (CI: GitHub Release)
```

Always branch off `testing` for new work:

```bash
git checkout testing
git checkout -b feature/my-thing
```

---

## Development Workflow

```bash
make build        # compile → dist/mdv
make test         # run test suite
make test-race    # run with race detector
make cover        # tests + coverage report
make lint         # golangci-lint
make vet          # go vet
make tidy         # go mod tidy + verify
```

All submitted code must be formatted with `gofmt` and pass `go vet`. The CI coverage gate requires **80% minimum**.

---

## Commit Messages

Follow the [Conventional Commits](https://www.conventionalcommits.org/) format:

```
<type>: <short summary>

[optional body]
```

Common types: `feat`, `fix`, `docs`, `refactor`, `test`, `chore`

```
feat: add requirements.txt parser
fix: handle missing go.mod gracefully
docs: update contributing guide
```

---

## Pull Request Guidelines

- Keep PRs small and focused on a single concern.
- Reference any related issue (`Closes #12`).
- Ensure all CI checks pass before requesting review.
- Add a brief description of what changed and why.

---

## Versioning & Releasing

Only maintainers cut releases. This section documents the mechanism so contributors understand how it works.

### How the version gets into the binary

Three package-level variables in `cmd/mdv/main.go` hold the defaults for local builds:

```go
var (
    version   = "dev"
    commit    = "none"
    buildDate = "unknown"
)
```

The Go linker overwrites them at compile time via `-ldflags` without touching source code:

```bash
go build -ldflags "\
  -X 'github.com/devekkx/module-dependency-visualizer/internal/config.version=v1.2.3' \
  -X 'github.com/devekkx/module-dependency-visualizer/internal/config.commit=abc1234' \
  -X 'github.com/devekkx/module-dependency-visualizer/internal/config.buildDate=2024-11-01T10:00:00Z'" \
  ./cmd/mdv
```

The `Makefile` wraps this. `COMMIT` and `BUILD_DATE` are derived automatically; `VERSION` defaults to `dev`:

```bash
make build                  # version=dev  commit=<sha>  built=<now>
make build VERSION=v1.2.3   # version=v1.2.3  commit=<sha>  built=<now>
```

### Cutting a release

The git tag is the single source of truth. Push a semver tag from `production` to trigger the release pipeline:

```bash
git checkout production
git tag v1.2.3
git push origin v1.2.3
```

`release.yml` reads the tag as `VERSION`, builds static binaries for all platforms, and publishes a GitHub Release with auto-generated notes and a `checksums.txt`.

### Semver policy

| Bump | Example | When |
|---|---|---|
| `PATCH` | `v1.2.3` → `v1.2.4` | Bug fixes, internal refactors |
| `MINOR` | `v1.2.3` → `v1.3.0` | New commands, flags, or export formats (backwards-compatible) |
| `MAJOR` | `v1.2.3` → `v2.0.0` | Breaking CLI changes, JSON schema major version bumps |

---

## Reporting Bugs

Open an [issue](https://github.com/devekkx/module-dependency-visualizer/issues) and include:

- A clear description of the problem
- Steps to reproduce
- Expected vs. actual behaviour
- Go version (`go version`) and OS

---

## Suggesting Features

Open an issue with the `enhancement` label. Describe the use case and why it fits the project's goals.

---

## License

By contributing, you agree that your contributions will be licensed under the [MIT License](./LICENSE).
