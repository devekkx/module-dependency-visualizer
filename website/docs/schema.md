# JSON Schema Reference

`mdv analyze` and `mdv export --format=json` emit a versioned JSON document.

Current version: **1.1.0**

---

## Top-level Structure

```json
{
  "schema_version": "1.1.0",
  "generated_at":   "<RFC3339 UTC timestamp>",
  "project":        { ... },
  "nodes":          [ ... ],
  "edges":          [ ... ],
  "stats":          { ... }
}
```

| Field | Type | Description |
|---|---|---|
| `schema_version` | `string` | Schema version string |
| `generated_at` | `string` | RFC3339 UTC timestamp of analysis |
| `project` | object | Describes the analysed project |
| `nodes` | array | Sorted by `id` for determinism |
| `edges` | array | Sorted by `(from, to)` for determinism |
| `stats` | object | Aggregate graph metrics |

---

## `project`

```json
{
  "name":        "github.com/example/myapp",
  "language":    "go",
  "root_path":   "/absolute/path/to/project",
  "main_module": "github.com/example/myapp"
}
```

| Field | Type | Description |
|---|---|---|
| `name` | `string` | Project name (module path for Go, package name for Node/Python) |
| `language` | `string` | Detected language: `go`, `node`, `python` |
| `root_path` | `string` | Absolute path to the analysed directory |
| `main_module` | `string` | The root module/package identifier |

---

## `nodes[]`

Each element represents a single package or module in the graph.

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

| Field | Type | Values | Description |
|---|---|---|---|
| `id` | `string` | `"<path>@<version>"` | Canonical node identifier |
| `name` | `string` | | Module/package path without version |
| `version` | `string` | | Version string (empty for the main module) |
| `kind` | `string` | `"main"`, `"module"`, `"replace"` | Role in the graph |
| `indirect` | `bool` | | `true` for transitive-only dependencies |
| `replaced_by` | `null` \| object | | Set when a `replace` directive is in effect |
| `metadata` | `object` | | Reserved for audit enrichment |

**Main module** uses an empty version string: `"example.com/app@"`.

### `replaced_by` object

```json
{
  "path":    "github.com/fork/cobra",
  "version": "v1.8.1-fork"
}
```

---

## `edges[]`

Each element represents a directed dependency relationship.

```json
{
  "from":     "github.com/example/myapp@",
  "to":       "github.com/spf13/cobra@v1.8.0",
  "kind":     "depends_on",
  "metadata": {}
}
```

| Field | Type | Values | Description |
|---|---|---|---|
| `from` | `string` | node `id` | Dependent module |
| `to` | `string` | node `id` | Dependency module |
| `kind` | `string` | `"depends_on"`, `"replaces"` | Relationship type |
| `metadata` | `object` | | Reserved for future enrichment |

---

## `stats`

```json
{
  "node_count": 42,
  "edge_count": 87,
  "max_depth":  5,
  "has_cycles": false
}
```

| Field | Type | Description |
|---|---|---|
| `node_count` | `int` | Total number of nodes |
| `edge_count` | `int` | Total number of edges |
| `max_depth` | `int` | Longest path from root to a leaf |
| `has_cycles` | `bool` | `true` if the graph contains cycles |

---

## Versioning Policy

- **Minor additions** (new optional fields) do not bump `schema_version`.
- **Breaking changes** (removed fields, renamed fields, type changes) bump the major version.
- `Decode()` returns an error if `schema_version` does not match the supported version.

---

## Known Limitations

- `go mod graph` emits toolchain pseudo-nodes (e.g. `go@1.24`, `toolchain@go1.24.0`) as regular nodes; these appear in the graph.
- Mermaid output renders reliably up to ~500 nodes. For larger graphs, use `--format=dot`.
