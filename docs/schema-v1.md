# JSON Schema v1.0.0 Reference

`mdv analyze` and `mdv export --format=json` emit a versioned JSON document conforming to this schema.

## Top-level Structure

```json
{
  "schema_version": "1.0.0",
  "generated_at":   "<RFC3339 UTC timestamp>",
  "project":        { ... },
  "nodes":          [ ... ],
  "edges":          [ ... ],
  "stats":          { ... }
}
```

| Field            | Type     | Description                                  |
|------------------|----------|----------------------------------------------|
| `schema_version` | `string` | Always `"1.0.0"` in this version             |
| `generated_at`   | `string` | RFC3339 UTC timestamp of analysis             |
| `project`        | object   | Describes the analyzed project                |
| `nodes`          | array    | Sorted by `id` for determinism                |
| `edges`          | array    | Sorted by `(from, to)` for determinism        |
| `stats`          | object   | Aggregate graph metrics                       |

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
| `metadata`   | `object`                            | Reserved for Phase 4 enrichment          |

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

## Versioning Policy

- **Minor additions** (new optional fields) do not bump `schema_version`.
- **Breaking changes** (removed fields, type changes) bump the major version.
- `Decode()` returns an error if `schema_version` does not match the supported version.

## Known Limitations

- Mermaid output renders reliably up to ~500 nodes. For larger graphs, use `--format=dot` and render with Graphviz.
- `go mod graph` includes toolchain pseudo-nodes (e.g., `go@1.26`, `toolchain@go1.26`) as regular nodes.
