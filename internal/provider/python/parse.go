package python

import (
	"encoding/json"
	"fmt"
	"strings"

	"module-dependency-visualizer/internal/graph"
)

// DepNode is an intermediate tree of a package and its transitive dependencies.
type DepNode struct {
	Name     string
	Version  string
	Dev      bool
	Children []DepNode
}

// buildGraph converts a DepNode tree into an immutable graph.Graph.
// It deduplicates nodes by ID and edges by (From, To), and skips recursing
// into nodes whose children have already been processed (cycle guard).
func buildGraph(root DepNode) (*graph.Graph, error) {
	b := graph.NewBuilder()
	added := make(map[graph.NodeID]bool)
	processed := make(map[graph.NodeID]bool)
	edgeSet := make(map[[2]graph.NodeID]bool)

	var walk func(n DepNode, parentID graph.NodeID, isMain bool) error
	walk = func(n DepNode, parentID graph.NodeID, isMain bool) error {
		id := graph.NewNodeID(n.Name, n.Version)

		if !added[id] {
			kind := graph.NodeKindModule
			if isMain {
				kind = graph.NodeKindMain
			}
			if err := b.AddNode(graph.Node{
				ID:       id,
				Name:     n.Name,
				Version:  n.Version,
				Kind:     kind,
				Indirect: n.Dev,
			}); err != nil {
				return fmt.Errorf("python: add node %s: %w", id, err)
			}
			added[id] = true
		}

		if parentID != "" {
			ek := [2]graph.NodeID{parentID, id}
			if !edgeSet[ek] {
				_ = b.AddEdge(graph.Edge{From: parentID, To: id, Kind: graph.EdgeKindDependsOn})
				edgeSet[ek] = true
			}
		}

		if processed[id] {
			return nil
		}
		processed[id] = true

		for _, child := range n.Children {
			if err := walk(child, id, false); err != nil {
				return err
			}
		}
		return nil
	}

	if err := walk(root, "", true); err != nil {
		return nil, err
	}

	g, err := b.Build()
	if err != nil {
		return nil, fmt.Errorf("python: build graph: %w", err)
	}
	return g, nil
}

// -------------------------------------------------------------------
// Poetry lockfile parsing (TOML line-scanner, no external dependency)
// -------------------------------------------------------------------

type poetryPkg struct {
	Name         string
	Version      string
	Optional     bool
	Category     string            // "main" or "dev" (Poetry < 1.2)
	Dependencies map[string]string // dep name → raw version spec
}

// ParsePoetryLock parses a poetry.lock file (Poetry 1.x format) and builds
// a dependency tree rooted at the given project name/version.
// Direct deps are inferred from pkgDeps (from pyproject.toml); if pkgDeps is
// nil, all non-dev packages are treated as direct.
func ParsePoetryLock(rootName, rootVersion string, pkgDeps map[string]string, data []byte) (DepNode, error) {
	pkgs, err := parsePoetryLockPkgs(data)
	if err != nil {
		return DepNode{}, err
	}

	// Build an index: lowercase name → *poetryPkg.
	// Poetry package names are case-insensitive and normalised to lowercase.
	idx := make(map[string]*poetryPkg, len(pkgs))
	for i := range pkgs {
		idx[normalise(pkgs[i].Name)] = &pkgs[i]
	}

	// Determine direct deps: use pkgDeps if provided, else all non-dev/optional.
	directNames := pkgDeps
	if directNames == nil {
		directNames = make(map[string]string)
		for _, p := range pkgs {
			if p.Category != "dev" && !p.Optional {
				directNames[normalise(p.Name)] = p.Version
			}
		}
	}

	root := DepNode{Name: rootName, Version: rootVersion}

	visited := make(map[string]bool)
	var buildChildren func(pkg *poetryPkg) []DepNode
	buildChildren = func(pkg *poetryPkg) []DepNode {
		var children []DepNode
		for depName := range pkg.Dependencies {
			if depName == "python" {
				continue
			}
			key := normalise(depName)
			dep, ok := idx[key]
			if !ok {
				continue
			}
			child := DepNode{Name: dep.Name, Version: dep.Version, Dev: dep.Category == "dev" || dep.Optional}
			if !visited[key] {
				visited[key] = true
				child.Children = buildChildren(dep)
			}
			children = append(children, child)
		}
		return children
	}

	for depName := range directNames {
		if depName == "python" {
			continue
		}
		key := normalise(depName)
		pkg, ok := idx[key]
		if !ok {
			continue
		}
		child := DepNode{Name: pkg.Name, Version: pkg.Version, Dev: pkg.Category == "dev" || pkg.Optional}
		if !visited[key] {
			visited[key] = true
			child.Children = buildChildren(pkg)
		}
		root.Children = append(root.Children, child)
	}

	return root, nil
}

// parsePoetryLockPkgs parses [[package]] blocks from a poetry.lock byte slice.
func parsePoetryLockPkgs(data []byte) ([]poetryPkg, error) {
	var pkgs []poetryPkg
	var cur *poetryPkg
	inDeps := false
	arrayDepth := 0

	for _, rawLine := range strings.Split(string(data), "\n") {
		line := strings.TrimRight(rawLine, "\r")

		// Track multi-line array depth (skip content until balanced ]).
		if arrayDepth > 0 {
			for _, ch := range line {
				switch ch {
				case '[':
					arrayDepth++
				case ']':
					arrayDepth--
				}
			}
			continue
		}

		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		// New top-level package array-of-tables entry.
		if trimmed == "[[package]]" {
			if cur != nil && cur.Name != "" {
				pkgs = append(pkgs, *cur)
			}
			cur = &poetryPkg{Dependencies: make(map[string]string)}
			inDeps = false
			continue
		}

		if cur == nil {
			continue
		}

		// Nested array-of-tables that is NOT [[package]] — skip section.
		if strings.HasPrefix(trimmed, "[[") {
			inDeps = false
			continue
		}

		// Sub-table header inside current package.
		if strings.HasPrefix(trimmed, "[package.dependencies]") {
			inDeps = true
			continue
		}
		if strings.HasPrefix(trimmed, "[package.") || strings.HasPrefix(trimmed, "[metadata") {
			inDeps = false
			continue
		}

		// Key = value parsing.
		eqIdx := strings.IndexByte(line, '=')
		if eqIdx < 0 {
			continue
		}
		key := strings.TrimSpace(line[:eqIdx])
		val := strings.TrimSpace(line[eqIdx+1:])

		// Detect start of multi-line array (e.g. files = [).
		if val == "[" || strings.HasSuffix(strings.TrimSpace(val), "[") {
			arrayDepth = 1
			continue
		}

		val = unquoteTOML(val)

		if inDeps {
			// Skip optional inline-table deps (version = "...", optional = true).
			if !strings.Contains(val, "optional = true") {
				cur.Dependencies[key] = val
			}
		} else {
			switch key {
			case "name":
				cur.Name = val
			case "version":
				cur.Version = val
			case "optional":
				cur.Optional = val == "true"
			case "category":
				cur.Category = val
			}
		}
	}

	if cur != nil && cur.Name != "" {
		pkgs = append(pkgs, *cur)
	}
	return pkgs, nil
}

// -------------------------------------------------------------------
// uv.lock parsing (TOML line-scanner)
// -------------------------------------------------------------------

type uvPkg struct {
	Name         string
	Version      string
	Dependencies []string // dep names only
}

// ParseUvLock parses a uv.lock file and builds a dependency tree.
// Direct deps are provided via directDeps (from pyproject.toml [project.dependencies]).
func ParseUvLock(rootName, rootVersion string, directDeps []string, data []byte) (DepNode, error) {
	pkgs, err := parseUvLockPkgs(data)
	if err != nil {
		return DepNode{}, err
	}

	idx := make(map[string]*uvPkg, len(pkgs))
	for i := range pkgs {
		idx[normalise(pkgs[i].Name)] = &pkgs[i]
	}

	// If no direct deps given, all packages are direct.
	directSet := make(map[string]bool, len(directDeps))
	for _, d := range directDeps {
		directSet[normalise(d)] = true
	}
	if len(directSet) == 0 {
		for k := range idx {
			directSet[k] = true
		}
	}

	root := DepNode{Name: rootName, Version: rootVersion}
	visited := make(map[string]bool)

	var buildChildren func(pkg *uvPkg) []DepNode
	buildChildren = func(pkg *uvPkg) []DepNode {
		var children []DepNode
		for _, depName := range pkg.Dependencies {
			key := normalise(depName)
			dep, ok := idx[key]
			if !ok {
				continue
			}
			child := DepNode{Name: dep.Name, Version: dep.Version}
			if !visited[key] {
				visited[key] = true
				child.Children = buildChildren(dep)
			}
			children = append(children, child)
		}
		return children
	}

	for directName := range directSet {
		pkg, ok := idx[directName]
		if !ok {
			continue
		}
		child := DepNode{Name: pkg.Name, Version: pkg.Version}
		if !visited[directName] {
			visited[directName] = true
			child.Children = buildChildren(pkg)
		}
		root.Children = append(root.Children, child)
	}

	return root, nil
}

// parseUvLockPkgs parses [[package]] blocks from a uv.lock file.
func parseUvLockPkgs(data []byte) ([]uvPkg, error) {
	var pkgs []uvPkg
	var cur *uvPkg
	inDepsArray := false

	for _, rawLine := range strings.Split(string(data), "\n") {
		line := strings.TrimRight(rawLine, "\r")
		trimmed := strings.TrimSpace(line)

		if trimmed == "[[package]]" {
			if cur != nil && cur.Name != "" {
				pkgs = append(pkgs, *cur)
			}
			cur = &uvPkg{}
			inDepsArray = false
			continue
		}

		if cur == nil {
			continue
		}

		if inDepsArray {
			if trimmed == "]" {
				inDepsArray = false
				continue
			}
			// Parse inline table: { name = "foo" } or { name = "foo", version = "..." }
			if n := extractInlineField(trimmed, "name"); n != "" {
				cur.Dependencies = append(cur.Dependencies, n)
			}
			continue
		}

		if strings.HasPrefix(trimmed, "dependencies = [") {
			// Check for empty one-liner: dependencies = []
			if strings.Contains(trimmed, "]") {
				continue
			}
			inDepsArray = true
			continue
		}

		eqIdx := strings.IndexByte(line, '=')
		if eqIdx < 0 {
			continue
		}
		key := strings.TrimSpace(line[:eqIdx])
		val := unquoteTOML(strings.TrimSpace(line[eqIdx+1:]))

		switch key {
		case "name":
			cur.Name = val
		case "version":
			cur.Version = val
		}
	}

	if cur != nil && cur.Name != "" {
		pkgs = append(pkgs, *cur)
	}
	return pkgs, nil
}

// extractInlineField extracts the value of a named field from a TOML inline
// table string such as `{ name = "foo", version = "1.0" }`.
func extractInlineField(s, field string) string {
	needle := field + " = \""
	idx := strings.Index(s, needle)
	if idx < 0 {
		return ""
	}
	rest := s[idx+len(needle):]
	end := strings.IndexByte(rest, '"')
	if end < 0 {
		return ""
	}
	return rest[:end]
}

// -------------------------------------------------------------------
// Pipfile.lock parsing (JSON)
// -------------------------------------------------------------------

type pipfileEntry struct {
	Version string `json:"version"`
}

type pipfileLockFile struct {
	Default map[string]json.RawMessage `json:"default"`
	Develop map[string]json.RawMessage `json:"develop"`
}

// ParsePipfileLock parses a Pipfile.lock JSON file into a flat DepNode tree.
// Pipfile.lock does not encode transitive dependency relationships, so all
// packages are direct children of root.
func ParsePipfileLock(rootName, rootVersion string, data []byte) (DepNode, error) {
	var lock pipfileLockFile
	if err := json.Unmarshal(data, &lock); err != nil {
		return DepNode{}, fmt.Errorf("python: parse Pipfile.lock: %w", err)
	}

	root := DepNode{Name: rootName, Version: rootVersion}

	addEntries := func(entries map[string]json.RawMessage, dev bool) {
		for name, raw := range entries {
			var entry pipfileEntry
			if err := json.Unmarshal(raw, &entry); err != nil {
				continue
			}
			version := strings.TrimPrefix(entry.Version, "==")
			root.Children = append(root.Children, DepNode{
				Name:    name,
				Version: version,
				Dev:     dev,
			})
		}
	}

	addEntries(lock.Default, false)
	addEntries(lock.Develop, true)
	return root, nil
}

// -------------------------------------------------------------------
// requirements.txt parsing
// -------------------------------------------------------------------

// ParseRequirementsTxt parses a requirements.txt file into a flat DepNode tree.
// Only direct packages are listed; transitive relationships are not recorded.
func ParseRequirementsTxt(rootName, rootVersion string, data []byte) (DepNode, error) {
	root := DepNode{Name: rootName, Version: rootVersion}

	var continuation string
	for _, rawLine := range strings.Split(string(data), "\n") {
		line := strings.TrimRight(rawLine, "\r")

		// Handle line continuation.
		if strings.HasSuffix(line, "\\") {
			continuation += strings.TrimSuffix(line, "\\")
			continue
		}
		line = continuation + line
		continuation = ""

		// Strip inline comment.
		if idx := strings.Index(line, "#"); idx >= 0 {
			line = line[:idx]
		}
		line = strings.TrimSpace(line)

		if line == "" {
			continue
		}

		// Skip options and includes.
		if strings.HasPrefix(line, "-") {
			continue
		}

		// Skip VCS / URL requirements.
		if strings.Contains(line, "://") {
			continue
		}

		name, version := parseRequirementSpec(line)
		if name == "" {
			continue
		}
		root.Children = append(root.Children, DepNode{Name: name, Version: version})
	}

	return root, nil
}

// parseRequirementSpec splits a PEP 508 requirement string into (name, version).
// Extras and environment markers are stripped.
// Examples: "requests==2.31.0" → ("requests","2.31.0"),
//
//	"flask[async]>=2.0" → ("flask",">=2.0"),
//	"black; python_version>='3.8'" → ("black","").
func parseRequirementSpec(spec string) (name, version string) {
	// Strip environment markers.
	if idx := strings.IndexByte(spec, ';'); idx >= 0 {
		spec = strings.TrimSpace(spec[:idx])
	}

	// Strip extras: "pkg[extra1,extra2]" → "pkg".
	if idx := strings.IndexByte(spec, '['); idx >= 0 {
		end := strings.IndexByte(spec[idx:], ']')
		if end >= 0 {
			spec = spec[:idx] + spec[idx+end+1:]
		}
	}

	// Split on first version operator.
	operators := []string{"===", "~=", "!=", "==", ">=", "<=", ">", "<"}
	for _, op := range operators {
		if idx := strings.Index(spec, op); idx >= 0 {
			return strings.TrimSpace(spec[:idx]), strings.TrimSpace(spec[idx:])
		}
	}

	return strings.TrimSpace(spec), ""
}

// -------------------------------------------------------------------
// pyproject.toml parsing (PEP 621 [project] table only)
// -------------------------------------------------------------------

// PyProjectMeta holds the minimal project metadata from pyproject.toml.
type PyProjectMeta struct {
	Name    string
	Version string
	// DirectDeps holds PEP 508 dependency strings from [project.dependencies]
	// or [tool.poetry.dependencies].
	DirectDeps []string
}

// ParsePyProjectTOML reads a pyproject.toml byte slice and extracts the
// project name, version, and dependency list.
// It handles both [project] (PEP 621) and [tool.poetry] sections.
func ParsePyProjectTOML(data []byte) (PyProjectMeta, error) {
	var meta PyProjectMeta
	inProject := false   // inside [project] or [tool.poetry]
	inDepsArray := false // inside dependencies = [...]
	arrayDepth := 0

	for _, rawLine := range strings.Split(string(data), "\n") {
		line := strings.TrimRight(rawLine, "\r")

		if arrayDepth > 0 {
			for _, ch := range line {
				switch ch {
				case '[':
					arrayDepth++
				case ']':
					arrayDepth--
				}
			}
			if arrayDepth == 0 {
				inDepsArray = false
			} else {
				// Collect dependency line inside the array.
				val := unquoteTOML(strings.TrimSpace(strings.Trim(line, ", \t")))
				if val != "" && val != "python" && !strings.HasPrefix(val, "{") {
					meta.DirectDeps = append(meta.DirectDeps, val)
				}
			}
			continue
		}

		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		// Section header.
		if strings.HasPrefix(trimmed, "[") {
			inDepsArray = false
			inProject = trimmed == "[project]" || trimmed == "[tool.poetry]"
			continue
		}

		if !inProject {
			continue
		}

		if inDepsArray {
			// Inline array continuation (shouldn't reach here, handled above).
			continue
		}

		eqIdx := strings.IndexByte(line, '=')
		if eqIdx < 0 {
			continue
		}
		key := strings.TrimSpace(line[:eqIdx])
		val := strings.TrimSpace(line[eqIdx+1:])

		switch key {
		case "name":
			meta.Name = unquoteTOML(val)
		case "version":
			meta.Version = unquoteTOML(val)
		case "dependencies":
			// Could be inline array: dependencies = ["a", "b"]
			// or multi-line: dependencies = [\n  "a",\n  "b",\n]
			if strings.Contains(val, "]") {
				// Inline: parse from val.
				inner := val[strings.IndexByte(val, '[')+1 : strings.LastIndexByte(val, ']')]
				for _, item := range strings.Split(inner, ",") {
					item = unquoteTOML(strings.TrimSpace(item))
					if item != "" && item != "python" {
						meta.DirectDeps = append(meta.DirectDeps, item)
					}
				}
			} else if strings.HasSuffix(strings.TrimSpace(val), "[") || val == "[" {
				arrayDepth = 1
				inDepsArray = true
			}
		}
	}

	return meta, nil
}

// ParsePyProjectDeps builds a flat DepNode tree from a pyproject.toml.
// Dependencies in [project.dependencies] become direct children of root.
func ParsePyProjectDeps(data []byte) (DepNode, error) {
	meta, err := ParsePyProjectTOML(data)
	if err != nil {
		return DepNode{}, err
	}
	if meta.Name == "" {
		return DepNode{}, fmt.Errorf("python: pyproject.toml: missing [project] name")
	}

	root := DepNode{Name: meta.Name, Version: meta.Version}
	for _, dep := range meta.DirectDeps {
		name, version := parseRequirementSpec(dep)
		if name != "" {
			root.Children = append(root.Children, DepNode{Name: name, Version: version})
		}
	}
	return root, nil
}

// -------------------------------------------------------------------
// Shared helpers
// -------------------------------------------------------------------

// normalise lowercases and replaces underscores/dots with hyphens per PEP 503.
func normalise(name string) string {
	name = strings.ToLower(name)
	name = strings.ReplaceAll(name, "_", "-")
	name = strings.ReplaceAll(name, ".", "-")
	return name
}

// unquoteTOML strips surrounding double or single quotes from a TOML value.
func unquoteTOML(s string) string {
	if len(s) >= 2 {
		if (s[0] == '"' && s[len(s)-1] == '"') || (s[0] == '\'' && s[len(s)-1] == '\'') {
			return s[1 : len(s)-1]
		}
	}
	return s
}
