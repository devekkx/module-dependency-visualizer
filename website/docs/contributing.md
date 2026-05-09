# Contributing

Thank you for your interest in contributing to `mdv`!

## Getting Started

**Prerequisites:** Go 1.21+

```bash
git clone https://github.com/devekkx/module-dependency-visualizer.git
cd module-dependency-visualizer
go build ./...
go test ./...
```

## Development Workflow

```bash
make build        # build → dist/mdv
make test         # run test suite
make test-race    # run with race detector
make cover        # generate HTML coverage report
make lint         # run golangci-lint
```

All submitted code must pass `gofmt` and `go vet` - CI enforces both.

## How to Contribute

1. **Fork** the repository and create your branch from `main`.
2. **Make focused changes** - one concern per PR.
3. **Write or update tests** for any logic you add or change.
4. **Run `make test`** to confirm nothing is broken.
5. **Open a pull request** against `main`.

## Commit Messages

Follow [Conventional Commits](https://www.conventionalcommits.org/):

```
<type>: <short summary>

[optional body]
```

Common types: `feat`, `fix`, `docs`, `refactor`, `test`, `chore`

**Examples:**
```
feat: add requirements.txt parser
fix: handle missing go.mod gracefully
docs: update contributing guide
```

## Pull Request Guidelines

- Keep PRs small and focused on a single concern.
- Reference any related issue (`Closes #12`).
- Ensure all CI checks pass before requesting review.
- Add a brief description of what changed and why.

## Reporting Bugs

Open an [issue](https://github.com/devekkx/module-dependency-visualizer/issues) and include:

- A clear description of the problem
- Steps to reproduce
- Expected vs. actual behavior
- Go version (`go version`) and OS

## Suggesting Features

Open an issue with the `enhancement` label. Describe the use case and why it fits the project's goals.

## License

By contributing, you agree that your contributions will be licensed under the [MIT License](https://github.com/devekkx/module-dependency-visualizer/blob/main/LICENSE).
