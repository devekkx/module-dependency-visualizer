package node

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/devekkx/module-dependency-visualizer/internal/graph"
)

// lockfilePackage represents one entry in the package-lock.json "packages" map (v2/v3).
type lockfilePackage struct {
	Name             string            `json:"name"`
	Version          string            `json:"version"`
	Dev              bool              `json:"dev"`
	Optional         bool              `json:"optional"`
	Dependencies     map[string]string `json:"dependencies"`
	DevDependencies  map[string]string `json:"devDependencies"`
	PeerDependencies map[string]string `json:"peerDependencies"`
}

// v1Dep is a node in the v1 lock file recursive dependencies tree.
type v1Dep struct {
	Version      string            `json:"version"`
	Dev          bool              `json:"dev"`
	Requires     map[string]string `json:"requires"`
	Dependencies map[string]v1Dep  `json:"dependencies"`
}

// PackageLock is the parsed representation of package-lock.json.
type PackageLock struct {
	Name            string                     `json:"name"`
	Version         string                     `json:"version"`
	LockfileVersion int                        `json:"lockfileVersion"`
	Packages        map[string]lockfilePackage `json:"packages"`    // v2/v3
	Dependencies    map[string]v1Dep           `json:"dependencies"` // v1
}

// ParsePackageLock parses package-lock.json content (supports v1, v2, v3).
func ParsePackageLock(data []byte) (*PackageLock, error) {
	var lock PackageLock
	if err := json.Unmarshal(data, &lock); err != nil {
		return nil, fmt.Errorf("node: parse package-lock.json: %w", err)
	}
	return &lock, nil
}

// PackageNameFromKey extracts the npm package name from a package-lock.json
// "packages" key such as "node_modules/chalk" or "node_modules/@scope/name".
// For nested installs like "node_modules/foo/node_modules/bar" it returns "bar".
func PackageNameFromKey(key string) string {
	const sep = "node_modules/"
	idx := strings.LastIndex(key, sep)
	if idx < 0 {
		return key
	}
	return key[idx+len(sep):]
}

// LockToGraph converts a PackageLock into an immutable graph.
// It dispatches to v2/v3 or v1 logic based on LockfileVersion.
func LockToGraph(lock *PackageLock) (*graph.Graph, string, error) {
	if lock.LockfileVersion >= 2 && lock.Packages != nil {
		return lockV2ToGraph(lock)
	}
	return lockV1ToGraph(lock)
}

// lockV2ToGraph handles package-lock.json v2 and v3 (flat "packages" map).
func lockV2ToGraph(lock *PackageLock) (*graph.Graph, string, error) {
	root, ok := lock.Packages[""]
	if !ok {
		return nil, "", fmt.Errorf("node: package-lock.json: missing root entry")
	}

	rootName := coalesce(root.Name, lock.Name)
	rootVersion := coalesce(root.Version, lock.Version)

	b := graph.NewBuilder()
	edgeSet := make(map[[2]graph.NodeID]bool)

	rootID := graph.NewNodeID(rootName, rootVersion)
	if err := b.AddNode(graph.Node{
		ID:      rootID,
		Name:    rootName,
		Version: rootVersion,
		Kind:    graph.NodeKindMain,
	}); err != nil {
		return nil, "", fmt.Errorf("node: add root: %w", err)
	}

	// nameToID maps package name → its canonical NodeID (top-level entries only).
	nameToID := make(map[string]graph.NodeID, len(lock.Packages))

	for key, pkg := range lock.Packages {
		if key == "" || strings.Count(key, "node_modules/") != 1 {
			continue
		}
		name := PackageNameFromKey(key)
		id := graph.NewNodeID(name, pkg.Version)
		if err := b.AddNode(graph.Node{
			ID:       id,
			Name:     name,
			Version:  pkg.Version,
			Kind: graph.NodeKindModule,
			Dev:  pkg.Dev,
		}); err != nil {
			continue // skip duplicate (same name@version from workspaces)
		}
		nameToID[name] = id
	}

	addEdges := func(fromID graph.NodeID, deps map[string]string) {
		for depName := range deps {
			toID, ok := nameToID[depName]
			if !ok {
				continue
			}
			ek := [2]graph.NodeID{fromID, toID}
			if edgeSet[ek] {
				continue
			}
			_ = b.AddEdge(graph.Edge{From: fromID, To: toID, Kind: graph.EdgeKindDependsOn})
			edgeSet[ek] = true
		}
	}

	addEdges(rootID, root.Dependencies)
	addEdges(rootID, root.DevDependencies)
	addEdges(rootID, root.PeerDependencies)

	for key, pkg := range lock.Packages {
		if key == "" || strings.Count(key, "node_modules/") != 1 {
			continue
		}
		name := PackageNameFromKey(key)
		fromID, ok := nameToID[name]
		if !ok {
			continue
		}
		addEdges(fromID, pkg.Dependencies)
	}

	g, err := b.Build()
	if err != nil {
		return nil, "", fmt.Errorf("node: build graph: %w", err)
	}
	return g, rootName, nil
}

// lockV1ToGraph handles package-lock.json v1 (recursive "dependencies" tree).
// It performs two passes: first collecting all nodes, then building edges via
// each package's `requires` field so that hoisted packages (the common case)
// are correctly linked to their dependants.
func lockV1ToGraph(lock *PackageLock) (*graph.Graph, string, error) {
	b := graph.NewBuilder()
	added := make(map[graph.NodeID]bool)
	edgeSet := make(map[[2]graph.NodeID]bool)

	// nameToID maps package name → its NodeID.
	// Top-level entries take priority; nested entries only fill gaps.
	nameToID := make(map[string]graph.NodeID)

	rootID := graph.NewNodeID(lock.Name, lock.Version)
	if err := b.AddNode(graph.Node{
		ID:      rootID,
		Name:    lock.Name,
		Version: lock.Version,
		Kind:    graph.NodeKindMain,
	}); err != nil {
		return nil, "", fmt.Errorf("node: add root: %w", err)
	}
	added[rootID] = true

	// Pass 1: collect all nodes at all nesting levels.
	// Top-level (depth 0) entries overwrite any previously registered name,
	// ensuring that the flat registry reflects the hoisted (canonical) versions.
	var collectNodes func(deps map[string]v1Dep, depth int)
	collectNodes = func(deps map[string]v1Dep, depth int) {
		for name, dep := range deps {
			id := graph.NewNodeID(name, dep.Version)
			if !added[id] {
				_ = b.AddNode(graph.Node{
					ID:      id,
					Name:    name,
					Version: dep.Version,
					Kind:    graph.NodeKindModule,
					Dev:     dep.Dev,
				})
				added[id] = true
			}
			if _, exists := nameToID[name]; !exists || depth == 0 {
				nameToID[name] = id
			}
			collectNodes(dep.Dependencies, depth+1)
		}
	}
	collectNodes(lock.Dependencies, 0)

	addEdge := func(from, to graph.NodeID) {
		ek := [2]graph.NodeID{from, to}
		if !edgeSet[ek] {
			_ = b.AddEdge(graph.Edge{From: from, To: to, Kind: graph.EdgeKindDependsOn})
			edgeSet[ek] = true
		}
	}

	// Pass 2: build edges.
	// For each package, use `requires` to add edges to the resolved versions.
	// When a required dep is nested directly under the package, use that
	// version; otherwise fall back to the hoisted nameToID registry.
	var walkEdges func(fromID graph.NodeID, dep v1Dep)
	walkEdges = func(fromID graph.NodeID, dep v1Dep) {
		for reqName := range dep.Requires {
			if nested, ok := dep.Dependencies[reqName]; ok {
				addEdge(fromID, graph.NewNodeID(reqName, nested.Version))
			} else if toID, ok := nameToID[reqName]; ok {
				addEdge(fromID, toID)
			}
		}
		for name, nested := range dep.Dependencies {
			nestedID := graph.NewNodeID(name, nested.Version)
			addEdge(fromID, nestedID)
			walkEdges(nestedID, nested)
		}
	}

	for name, dep := range lock.Dependencies {
		toID := graph.NewNodeID(name, dep.Version)
		addEdge(rootID, toID)
		walkEdges(toID, dep)
	}

	g, err := b.Build()
	if err != nil {
		return nil, "", fmt.Errorf("node: build graph: %w", err)
	}
	return g, lock.Name, nil
}

func coalesce(a, b string) string {
	if a != "" {
		return a
	}
	return b
}
