# Versioning

`mdv` uses [Semantic Versioning](https://semver.org). The version string, git commit hash, and build timestamp are baked into the binary at compile time — there is no separate version file. The git tag is the single source of truth.

## How the version gets into the binary

Three package-level variables in `cmd/mdv/main.go` hold the defaults for local builds:

```go
var (
    version   = "dev"
    commit    = "none"
    buildDate = "unknown"
)
```

Go's `-ldflags` linker flag overwrites them at compile time without changing source code:

```bash
go build -ldflags "\
  -X '...internal/config.version=v1.2.3' \
  -X '...internal/config.commit=abc1234' \
  -X '...internal/config.buildDate=2024-11-01T10:00:00Z'" \
  ./cmd/mdv
```

The `Makefile` handles this automatically. `COMMIT` and `BUILD_DATE` are derived from git and the system clock; `VERSION` defaults to `dev`:

```bash
make build                  # version=dev  commit=<sha>  built=<now>
make build VERSION=v1.2.3   # version=v1.2.3 commit=<sha>  built=<now>
```

Verify a binary at any time:

```bash
mdv version
# version=v1.2.3 commit=abc1234 built=2024-11-01T10:00:00Z
```

## Cutting a release

The release pipeline reads the git tag and passes it as `VERSION` automatically — you only need to push a tag:

```bash
git checkout production
git tag v1.2.3
git push origin v1.2.3
```

The [`release.yml`](https://github.com/devekkx/module-dependency-visualizer/blob/production/.github/workflows/release.yml) workflow fires, builds static binaries for all platforms, and publishes a GitHub Release with the tagged version baked in.

::: tip
Always tag from the `production` branch. The release workflow only triggers on tags matching `v*.*.*`.
:::

## Version format

Tags follow `vMAJOR.MINOR.PATCH`:

| Change type | Example | When to bump |
|---|---|---|
| `PATCH` | `v1.2.3` → `v1.2.4` | Bug fixes, internal refactors with no user-visible changes |
| `MINOR` | `v1.2.3` → `v1.3.0` | New commands, new flags, new export formats (backwards-compatible) |
| `MAJOR` | `v1.2.3` → `v2.0.0` | Breaking CLI flag changes, JSON schema major version bumps |

## Release artifacts

Each GitHub Release contains:

```
mdv_v1.2.3_linux_amd64.tar.gz
mdv_v1.2.3_linux_arm64.tar.gz
mdv_v1.2.3_darwin_amd64.tar.gz
mdv_v1.2.3_darwin_arm64.tar.gz
mdv_v1.2.3_windows_amd64.zip
checksums.txt
```

All binaries are built with `CGO_ENABLED=0` (fully static, no libc dependency) and stripped of debug symbols (`-s -w`) for a smaller binary size. Verify any archive against `checksums.txt`:

```bash
sha256sum --check checksums.txt
```
