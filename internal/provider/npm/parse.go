package npm

import (
	"encoding/json"
	"fmt"
	"strings"

	"module-dependency-visualizer/internal/graph"
)

// PackageLock represents a parsed package-lock.json (v1/v2/v3) or npm ls output.
// The same struct is reused for all formats; which fields are populated depends
// on the source format.
type PackageLock struct {
	Name            string                `json:"name"`
	Version         string                `json:"version"`
	LockfileVersion int                   `json:"lockfileVersion"`
	Packages        map[string]PackageV2  `json:"packages,omitempty"`  // v2/v3
	Dependencies    map[string]V1Dep      `json:"dependencies,omitempty"` // v1 / npm ls
}

// PackageV2 represents a single entry in the v2/v3 "packages" map.
type PackageV2 struct {
	Name             string            `json:"name,omitempty"`
	Version          string            `json:"version"`
	Dev              bool              `json:"dev,omitempty"`
	Dependencies     map[string]string `json:"dependencies,omitempty"`
	DevDependencies  map[string]string `json:"devDependencies,omitempty"`
}

// V1Dep represents a dependency entry in the v1 "dependencies" tree
// and the recursive tree produced by `npm ls --all --json`.
type V1Dep struct {
	Version      string            `json:"version"`
	Dev          bool              `json:"dev,omitempty"`
	Requires     map[string]string `json:"requires,omitempty"`
	Dependencies map[string]V1Dep  `json:"dependencies,omitempty"`
}

// ParsePackageLock parses the contents of a package-lock.json file.
// It supports lockfileVersion 1, 2, and 3.
func ParsePackageLock(data []byte) (*PackageLock, error) {
	var lock PackageLock
	if err := json.Unmarshal(data, &lock); err != nil {
		return nil, fmt.Errorf("npm: parse package-lock.json: %w", err)
	}
	return &lock, nil
}

// ParseNpmLS parses the JSON output of `npm ls --all --json`.
// The output shares the same tree structure as v1 package-lock.json
// dependencies, so we reuse PackageLock and treat it as v1 (lockfileVersion=0).
func ParseNpmLS(data []byte) (*PackageLock, error) {
	var lock PackageLock
	if err := json.Unmarshal(data, &lock); err != nil {
		return nil, fmt.Errorf("npm: parse npm ls output: %w", err)
	}
	// npm ls produces no lockfileVersion field; leave it as zero so buildGraph
	// dispatches to the v1/npm-ls path.
	return &lock, nil
}

// PackageNameFromKey extracts the npm package name from a node_modules key.
// Examples:
//
//	"node_modules/chalk"         → "chalk"
//	"node_modules/@types/node"   → "@types/node"
//	"node_modules/foo/node_modules/bar" → "bar"
func PackageNameFromKey(key string) string {
	const prefix = "node_modules/"
	// Find the last occurrence of the prefix to handle nested node_modules.
	idx := strings.LastIndex(key, prefix)
	if idx == -1 {
		return key
	}
	return key[idx+len(prefix):]
}

// buildGraph dispatches to the appropriate graph builder based on the lockfile
// version. Version 2 and 3 use the "packages" map; version 1 (and npm ls
// output, which has version 0) use the "dependencies" tree.
func buildGraph(lock *PackageLock) (*graph.Graph, string, error) {
	if lock.LockfileVersion >= 2 {
		return buildGraphFromPackages(lock)
	}
	return buildGraphFromDependencies(lock.Name, lock.Version, lock.Dependencies)
}

// buildGraphFromPackages builds a graph from the v2/v3 "packages" map.
func buildGraphFromPackages(lock *PackageLock) (*graph.Graph, string, error) {
	root, ok := lock.Packages[""]
	if !ok {
		return nil, "", fmt.Errorf("npm: package-lock.json missing root package entry")
	}

	rootName := lock.Name
	if rootName == "" {
		rootName = root.Name
	}
	rootVersion := lock.Version
	if rootVersion == "" {
		rootVersion = root.Version
	}

	b := graph.NewBuilder()

	rootID := graph.NewNodeID(rootName, rootVersion)
	if err := b.AddNode(graph.Node{
		ID:      rootID,
		Name:    rootName,
		Version: rootVersion,
		Kind:    graph.NodeKindMain,
	}); err != nil {
		return nil, "", fmt.Errorf("npm: add root node: %w", err)
	}

	// First pass: add all dependency nodes.
	for key, pkg := range lock.Packages {
		if key == "" {
			continue // root already added
		}
		name := PackageNameFromKey(key)
		version := pkg.Version
		id := graph.NewNodeID(name, version)

		node := graph.Node{
			ID:       id,
			Name:     name,
			Version:  version,
			Kind:     graph.NodeKindModule,
			Indirect: pkg.Dev,
		}
		if err := b.AddNode(node); err != nil {
			// Skip duplicate nodes (same package appearing under multiple paths).
			continue
		}
	}

	// Second pass: add edges from root to its direct declared dependencies.
	addedEdges := make(map[[2]graph.NodeID]bool)
	if err := addEdgesFromDeclarations(b, rootID, root.Dependencies, lock.Packages, addedEdges); err != nil {
		return nil, "", err
	}
	if err := addEdgesFromDeclarations(b, rootID, root.DevDependencies, lock.Packages, addedEdges); err != nil {
		return nil, "", err
	}

	// Third pass: add edges between non-root packages and their dependencies.
	for key, pkg := range lock.Packages {
		if key == "" {
			continue
		}
		fromName := PackageNameFromKey(key)
		fromID := graph.NewNodeID(fromName, pkg.Version)
		if err := addEdgesFromDeclarations(b, fromID, pkg.Dependencies, lock.Packages, addedEdges); err != nil {
			return nil, "", err
		}
	}

	g, err := b.Build()
	if err != nil {
		return nil, "", fmt.Errorf("npm: build graph: %w", err)
	}
	return g, rootName, nil
}

// addEdgesFromDeclarations resolves declared dependency names to their resolved
// package entries in the packages map and adds edges if both nodes are registered.
func addEdgesFromDeclarations(
	b *graph.Builder,
	fromID graph.NodeID,
	decls map[string]string,
	packages map[string]PackageV2,
	added map[[2]graph.NodeID]bool,
) error {
	for depName := range decls {
		toID, ok := resolvePackageID(depName, packages)
		if !ok {
			continue
		}
		key := [2]graph.NodeID{fromID, toID}
		if added[key] {
			continue
		}
		added[key] = true
		edge := graph.Edge{
			From: fromID,
			To:   toID,
			Kind: graph.EdgeKindDependsOn,
		}
		if err := b.AddEdge(edge); err != nil {
			return fmt.Errorf("npm: add edge %s->%s: %w", fromID, toID, err)
		}
	}
	return nil
}

// resolvePackageID finds the NodeID for a dependency name by scanning the
// packages map for a matching key suffix.
func resolvePackageID(name string, packages map[string]PackageV2) (graph.NodeID, bool) {
	// Prefer the top-level entry: "node_modules/<name>"
	key := "node_modules/" + name
	if pkg, ok := packages[key]; ok {
		return graph.NewNodeID(name, pkg.Version), true
	}
	// Scan for nested node_modules entries as a fallback.
	suffix := "/node_modules/" + name
	for k, pkg := range packages {
		if strings.HasSuffix(k, suffix) {
			return graph.NewNodeID(name, pkg.Version), true
		}
	}
	return "", false
}

// buildGraphFromDependencies builds a graph from the v1 / npm ls recursive
// dependency tree. It deduplicates edges using a seen map.
func buildGraphFromDependencies(
	rootName, rootVersion string,
	deps map[string]V1Dep,
) (*graph.Graph, string, error) {
	b := graph.NewBuilder()
	rootID := graph.NewNodeID(rootName, rootVersion)

	if err := b.AddNode(graph.Node{
		ID:      rootID,
		Name:    rootName,
		Version: rootVersion,
		Kind:    graph.NodeKindMain,
	}); err != nil {
		return nil, "", fmt.Errorf("npm: add root node: %w", err)
	}

	addedEdges := make(map[[2]graph.NodeID]bool)

	if err := addV1Deps(b, rootID, deps, addedEdges); err != nil {
		return nil, "", err
	}

	g, err := b.Build()
	if err != nil {
		return nil, "", fmt.Errorf("npm: build graph: %w", err)
	}
	return g, rootName, nil
}

// addV1Deps recursively registers nodes and edges from the v1/npm-ls tree.
func addV1Deps(
	b *graph.Builder,
	parentID graph.NodeID,
	deps map[string]V1Dep,
	added map[[2]graph.NodeID]bool,
) error {
	for name, dep := range deps {
		id := graph.NewNodeID(name, dep.Version)
		// AddNode returns ErrDuplicateNode if already registered; that is fine —
		// the same package may appear in multiple sub-trees.
		_ = b.AddNode(graph.Node{
			ID:       id,
			Name:     name,
			Version:  dep.Version,
			Kind:     graph.NodeKindModule,
			Indirect: dep.Dev,
		})

		edgeKey := [2]graph.NodeID{parentID, id}
		if !added[edgeKey] {
			added[edgeKey] = true
			if err := b.AddEdge(graph.Edge{
				From: parentID,
				To:   id,
				Kind: graph.EdgeKindDependsOn,
			}); err != nil {
				return fmt.Errorf("npm: add edge %s->%s: %w", parentID, id, err)
			}
		}

		// Recurse into nested dependencies.
		if len(dep.Dependencies) > 0 {
			if err := addV1Deps(b, id, dep.Dependencies, added); err != nil {
				return err
			}
		}
	}
	return nil
}
