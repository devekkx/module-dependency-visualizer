# Contributing to Universal Dependency Visualizer

Thank you for your interest in contributing! This document covers everything you need to get started.

---

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [Getting Started](#getting-started)
- [How to Contribute](#how-to-contribute)
- [Development Workflow](#development-workflow)
- [Commit Messages](#commit-messages)
- [Pull Request Guidelines](#pull-request-guidelines)
- [Reporting Bugs](#reporting-bugs)
- [Suggesting Features](#suggesting-features)

---

## Code of Conduct

Be respectful and constructive. Harassment or exclusionary behavior of any kind will not be tolerated.

---

## Getting Started

**Prerequisites:** Go 1.21+

```bash
git clone https://github.com/devekkx/module-dependency-visualizer.git
cd module-dependency-visualizer
go build ./...
go test ./...
```

---

## How to Contribute

1. **Fork** the repository and create your branch from `main`.
2. **Make your changes** — keep them focused and minimal.
3. **Write or update tests** for any logic you add or change.
4. **Run the test suite** to confirm nothing is broken.
5. **Open a pull request** against `main`.

---

## Development Workflow

```bash
# Run the project
go run main.go

# Run tests
go test ./...

# Format code (required before committing)
gofmt -w .

# Lint (install golangci-lint if needed)
golangci-lint run
```

All submitted code must be formatted with `gofmt` and pass `go vet`.

---

## Commit Messages

Follow the [Conventional Commits](https://www.conventionalcommits.org/) format:

```
<type>: <short summary>

[optional body]
```

Common types: `feat`, `fix`, `docs`, `refactor`, `test`, `chore`

Examples:
```
feat: add requirements.txt parser
fix: handle missing go.mod gracefully
docs: update contributing guide
```

---

## Pull Request Guidelines

- Keep PRs small and focused on a single concern.
- Reference any related issue in the PR description (e.g., `Closes #12`).
- Ensure all checks pass before requesting a review.
- Add a brief description of what changed and why.

---

## Reporting Bugs

Open an issue and include:

- A clear description of the problem
- Steps to reproduce
- Expected vs. actual behavior
- Go version (`go version`) and OS

---

## Suggesting Features

Open an issue with the `enhancement` label. Describe the use case and why it fits the project's goals. Check the [roadmap](./roadmap.pdf) first to see if it's already planned.

---

## License

By contributing, you agree that your contributions will be licensed under the [MIT License](./LICENSE).
