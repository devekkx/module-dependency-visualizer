# JSON Schema v1.1.0 Reference

`mdv analyse` and `mdv export --format=json` emit a versioned JSON document conforming to this schema.

## Top-level Structure

```json
{
  "schema_version": "1.1.0",
  "generated_at":   "<RFC3339 UTC timestamp>",
  "project":        { ... },
  "nodes":          [ ... ],
  "edges":          [ ... ],
  "stats":          { ... },
  "audit":          { ... }
}
```

| Field            | Type     | Description                                         |
|------------------|----------|-----------------------------------------------------|
| `schema_version` | `string` | Always `"1.1.0"` in this version                   |
| `generated_at`   | `string` | RFC3339 UTC timestamp of analysis                   |
| `project`        | object   | Describes the analysed project                      |
| `nodes`          | array    | Sorted by `id` for determinism                      |
| `edges`          | array    | Sorted by `(from, to)` for determinism              |
| `stats`          | object   | Aggregate graph metrics                             |
| `audit`          | object   | Optional - present only when `--audit` flag is used |

## `project`

```json
{
  "name":        "github.com/example/myapp",
  "language":    "go",
  "root_path":   "/absolute/path/to/project",
  "main_module": "github.com/example/myapp"
}
```

## `nodes[]`

```json
{
  "id":          "github.com/spf13/cobra@v1.8.0",
  "name":        "github.com/spf13/cobra",
  "version":     "v1.8.0",
  "kind":        "module",
  "indirect":    false,
  "replaced_by": null,
  "metadata":    {}
}
```

| Field        | Values                              | Description                              |
|--------------|-------------------------------------|------------------------------------------|
| `id`         | `"<path>@<version>"`               | Canonical node identifier                |
| `kind`       | `"main"`, `"module"`, `"replace"`  | Role of this module in the graph         |
| `indirect`   | `bool`                              | `true` for transitive-only dependencies  |
| `replaced_by`| `null` or `{path, version}` object  | Present when a `replace` directive exists|
| `metadata`   | `object`                            | Reserved for future enrichment           |

**Main module** uses an empty version string: `"example.com/app@"`.

## `edges[]`

```json
{
  "from":     "github.com/example/myapp@",
  "to":       "github.com/spf13/cobra@v1.8.0",
  "kind":     "depends_on",
  "metadata": {}
}
```

| Field    | Values                            | Description                    |
|----------|-----------------------------------|--------------------------------|
| `kind`   | `"depends_on"`, `"replaces"`      | Relationship type              |
| `metadata` | `object`                        | Reserved for future use        |

## `stats`

```json
{
  "node_count": 42,
  "edge_count": 87,
  "max_depth":  5,
  "has_cycles": false
}
```

## `audit` (optional)

Present only when the graph was produced with `--audit`. Omitted entirely otherwise.

```json
{
  "scanned_at": "2024-11-01T10:00:00Z",
  "vulnerabilities": [
    {
      "node_id":  "github.com/foo/bar@v1.2.3",
      "id":       "CVE-2024-12345",
      "summary":  "Remote code execution via malformed input",
      "severity": "CRITICAL",
      "fixed_in": "v1.2.4",
      "link":     "https://osv.dev/vulnerability/CVE-2024-12345"
    }
  ],
  "conflicts": [
    {
      "module":   "github.com/baz/qux",
      "versions": ["v1.0.0", "v1.2.0"]
    }
  ],
  "licenses": {
    "github.com/spf13/cobra@v1.8.0": "Apache-2.0"
  }
}
```

### `audit.vulnerabilities[]`

| Field      | Type     | Description                                      |
|------------|----------|--------------------------------------------------|
| `node_id`  | `string` | Node `id` of the affected dependency             |
| `id`       | `string` | CVE or OSV identifier                            |
| `summary`  | `string` | Short description of the vulnerability           |
| `severity` | `string` | `CRITICAL`, `HIGH`, `MEDIUM`, `LOW`, `UNKNOWN`   |
| `fixed_in` | `string` | Version that resolves the issue (omitted if none)|
| `link`     | `string` | URL to the OSV advisory (omitted if unavailable) |

### `audit.conflicts[]`

| Field      | Type       | Description                                          |
|------------|------------|------------------------------------------------------|
| `module`   | `string`   | Module path (without version)                        |
| `versions` | `string[]` | All resolved versions of this module in the graph    |

### `audit.licenses`

A map of `node_id → SPDX license identifier` for every dependency where a license was detected.

## Versioning Policy

- **Minor additions** (new optional fields) do not bump `schema_version`.
- **Breaking changes** (removed fields, type changes) bump the major version.
- `Decode()` returns an error if `schema_version` does not match the supported version.

## Known Limitations

- `go mod graph` includes toolchain pseudo-nodes (e.g. `go@1.24`, `toolchain@go1.24.0`) as regular nodes.
- Mermaid output renders reliably up to ~500 nodes. For larger graphs, use `--format=dot`.
