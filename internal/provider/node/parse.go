package node

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/devekkx/module-dependency-visualizer/internal/graph"
)

// PackageJSON holds the minimal fields we need from package.json.
type PackageJSON struct {
	Name             string            `json:"name"`
	Version          string            `json:"version"`
	Dependencies     map[string]string `json:"dependencies"`
	DevDependencies  map[string]string `json:"devDependencies"`
	PeerDependencies map[string]string `json:"peerDependencies"`
}

// ParsePackageJSON parses the minimal fields from package.json bytes.
func ParsePackageJSON(data []byte) (PackageJSON, error) {
	var pkg PackageJSON
	if err := json.Unmarshal(data, &pkg); err != nil {
		return PackageJSON{}, fmt.Errorf("node: parse package.json: %w", err)
	}
	return pkg, nil
}

// ParseNpmLS parses the JSON output of `npm ls --all --json` into a DepNode tree.
func ParseNpmLS(data []byte) (DepNode, error) {
	type entry struct {
		Name         string           `json:"name"`
		Version      string           `json:"version"`
		Dev          bool             `json:"dev"`
		Dependencies map[string]entry `json:"dependencies"`
	}
	var root entry
	if err := json.Unmarshal(data, &root); err != nil {
		return DepNode{}, fmt.Errorf("node: npm ls: %w", err)
	}
	var convert func(name, version string, deps map[string]entry, dev bool) DepNode
	convert = func(name, version string, deps map[string]entry, dev bool) DepNode {
		node := DepNode{Name: name, Version: version, Dev: dev}
		for depName, dep := range deps {
			node.Children = append(node.Children, convert(depName, dep.Version, dep.Dependencies, dep.Dev))
		}
		return node
	}
	return convert(root.Name, root.Version, root.Dependencies, false), nil
}

// pnpmDep is one entry in pnpm ls --depth=Infinity --json output.
type pnpmDep struct {
	From         string              `json:"from"`
	Version      string              `json:"version"`
	Dependencies map[string]pnpmDep `json:"dependencies"`
}

type pnpmListEntry struct {
	Name             string              `json:"name"`
	Version          string              `json:"version"`
	Dependencies     map[string]pnpmDep  `json:"dependencies"`
	DevDependencies  map[string]pnpmDep  `json:"devDependencies"`
	PeerDependencies map[string]pnpmDep  `json:"peerDependencies"`
}

// ParsePnpmLS parses the JSON output of `pnpm ls --depth=Infinity --json`.
func ParsePnpmLS(data []byte) (DepNode, error) {
	var entries []pnpmListEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return DepNode{}, fmt.Errorf("node: pnpm ls: %w", err)
	}
	if len(entries) == 0 {
		return DepNode{}, fmt.Errorf("node: pnpm ls: empty output")
	}
	root := entries[0]

	var convertDeps func(deps map[string]pnpmDep, dev bool) []DepNode
	convertDeps = func(deps map[string]pnpmDep, dev bool) []DepNode {
		nodes := make([]DepNode, 0, len(deps))
		for mapKey, dep := range deps {
			name := dep.From
			if name == "" {
				name = mapKey
			}
			nodes = append(nodes, DepNode{
				Name:     name,
				Version:  dep.Version,
				Dev:      dev,
				Children: convertDeps(dep.Dependencies, false),
			})
		}
		return nodes
	}

	rootNode := DepNode{Name: root.Name, Version: root.Version}
	rootNode.Children = append(convertDeps(root.Dependencies, false), convertDeps(root.DevDependencies, true)...)
	rootNode.Children = append(rootNode.Children, convertDeps(root.PeerDependencies, false)...)
	return rootNode, nil
}

// yarnTree is one node in the yarn list --json tree output.
type yarnTree struct {
	Name     string     `json:"name"` // "pkg@version"
	Children []yarnTree `json:"children"`
}

type yarnListOutput struct {
	Type string `json:"type"`
	Data struct {
		Type  string     `json:"type"`
		Trees []yarnTree `json:"trees"`
	} `json:"data"`
}

// ParseYarnList parses the JSON output of `yarn list --json` (yarn v1).
// pkg provides the root package name/version and the dev-dependency set.
func ParseYarnList(pkg PackageJSON, data []byte) (DepNode, error) {
	var out yarnListOutput
	if err := json.Unmarshal(data, &out); err != nil {
		return DepNode{}, fmt.Errorf("node: yarn list: %w", err)
	}

	var convertTree func(tree yarnTree) DepNode
	convertTree = func(tree yarnTree) DepNode {
		name, version := splitAtVersion(tree.Name)
		_, isDev := pkg.DevDependencies[name]
		node := DepNode{Name: name, Version: version, Dev: isDev}
		for _, child := range tree.Children {
			node.Children = append(node.Children, convertTree(child))
		}
		return node
	}

	root := DepNode{Name: pkg.Name, Version: pkg.Version}
	for _, tree := range out.Data.Trees {
		root.Children = append(root.Children, convertTree(tree))
	}
	return root, nil
}

// berryInfoEntry is one line of `yarn info --all --json` NDJSON output.
type berryInfoEntry struct {
	Value    string `json:"value"` // e.g. "chalk@npm:5.3.0"
	Children struct {
		Version      string `json:"Version"`
		Dependencies []struct {
			Descriptor string `json:"descriptor"`
			Locator    string `json:"locator"`
		} `json:"Dependencies"`
	} `json:"children"`
}

// ParseYarnInfo parses the NDJSON output of `yarn info --all --json` (yarn berry).
// It builds and returns a full graph directly (each package's dep locators give
// the exact resolved name@version pairs).
func ParseYarnInfo(pkg PackageJSON, data []byte) (*graph.Graph, string, error) {
	type pkgInfo struct {
		name    string
		version string
		deps    [][2]string // [name, version] pairs
	}
	byKey := make(map[string]*pkgInfo) // "name@version" → info

	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var entry berryInfoEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue
		}
		name, version := parseLocator(entry.Value)
		if name == "" || version == "" {
			continue
		}
		info := &pkgInfo{name: name, version: version}
		for _, dep := range entry.Children.Dependencies {
			dn, dv := parseLocator(dep.Locator)
			if dn != "" && dv != "" {
				info.deps = append(info.deps, [2]string{dn, dv})
			}
		}
		byKey[name+"@"+version] = info
	}

	b := graph.NewBuilder()
	edgeSet := make(map[[2]graph.NodeID]bool)
	added := make(map[graph.NodeID]bool)

	rootID := graph.NewNodeID(pkg.Name, pkg.Version)
	if err := b.AddNode(graph.Node{
		ID: rootID, Name: pkg.Name, Version: pkg.Version,
		Kind: graph.NodeKindMain,
	}); err != nil {
		return nil, "", fmt.Errorf("node: yarn berry: add root: %w", err)
	}
	added[rootID] = true

	for _, info := range byKey {
		id := graph.NewNodeID(info.name, info.version)
		if !added[id] {
			_ = b.AddNode(graph.Node{
				ID: id, Name: info.name, Version: info.version,
				Kind: graph.NodeKindModule,
			})
			added[id] = true
		}
	}

	addEdge := func(from, to graph.NodeID) {
		ek := [2]graph.NodeID{from, to}
		if !edgeSet[ek] {
			_ = b.AddEdge(graph.Edge{From: from, To: to, Kind: graph.EdgeKindDependsOn})
			edgeSet[ek] = true
		}
	}

	for depName := range pkg.Dependencies {
		for _, info := range byKey {
			if info.name == depName {
				addEdge(rootID, graph.NewNodeID(info.name, info.version))
				break
			}
		}
	}
	for depName := range pkg.DevDependencies {
		for _, info := range byKey {
			if info.name == depName {
				addEdge(rootID, graph.NewNodeID(info.name, info.version))
				break
			}
		}
	}

	for _, info := range byKey {
		fromID := graph.NewNodeID(info.name, info.version)
		for _, dep := range info.deps {
			addEdge(fromID, graph.NewNodeID(dep[0], dep[1]))
		}
	}

	g, err := b.Build()
	if err != nil {
		return nil, "", fmt.Errorf("node: yarn berry: build graph: %w", err)
	}
	return g, pkg.Name, nil
}

// ParseBunLS parses the text output of `bun pm ls` and builds a graph.
// It interprets the tree-formatted output to derive parent-child dep edges.
func ParseBunLS(pkg PackageJSON, data []byte) (*graph.Graph, string, error) {
	type treeNode struct {
		name    string
		version string
		depth   int
	}

	var nodes []treeNode
	for _, line := range strings.Split(string(data), "\n") {
		depth, name, version, ok := parseBunLSLine(line)
		if !ok {
			continue
		}
		nodes = append(nodes, treeNode{name: name, version: version, depth: depth})
	}

	b := graph.NewBuilder()
	edgeSet := make(map[[2]graph.NodeID]bool)

	rootID := graph.NewNodeID(pkg.Name, pkg.Version)
	if err := b.AddNode(graph.Node{
		ID: rootID, Name: pkg.Name, Version: pkg.Version,
		Kind: graph.NodeKindMain,
	}); err != nil {
		return nil, "", fmt.Errorf("node: bun: add root: %w", err)
	}

	addEdge := func(from, to graph.NodeID) {
		ek := [2]graph.NodeID{from, to}
		if !edgeSet[ek] {
			_ = b.AddEdge(graph.Edge{From: from, To: to, Kind: graph.EdgeKindDependsOn})
			edgeSet[ek] = true
		}
	}

	// stack[i] = last node ID at tree level i-1. stack[0] = root.
	stack := []graph.NodeID{rootID}
	added := make(map[graph.NodeID]bool)
	added[rootID] = true

	for _, n := range nodes {
		id := graph.NewNodeID(n.name, n.version)
		_, isDev := pkg.DevDependencies[n.name]
		if !added[id] {
			_ = b.AddNode(graph.Node{
				ID: id, Name: n.name, Version: n.version,
				Kind: graph.NodeKindModule,
				Dev:  isDev,
			})
			added[id] = true
		}

		parentIdx := n.depth
		var parentID graph.NodeID
		if parentIdx < len(stack) {
			parentID = stack[parentIdx]
		} else {
			parentID = rootID
		}
		addEdge(parentID, id)

		childIdx := n.depth + 1
		if childIdx < len(stack) {
			stack[childIdx] = id
		} else {
			for len(stack) < childIdx {
				stack = append(stack, id)
			}
			stack = append(stack, id)
		}
	}

	g, err := b.Build()
	if err != nil {
		return nil, "", fmt.Errorf("node: bun: build graph: %w", err)
	}
	return g, pkg.Name, nil
}

// parseBunLSLine parses one line of `bun pm ls` tree output.
// Returns (depth, name, version, ok). depth=0 means direct dep of root.
func parseBunLSLine(line string) (depth int, name, version string, ok bool) {
	branches := []string{"├── ", "└── ", "├─ ", "└─ "}
	continuations := []string{"│   ", "    "}

	rest := line
	for {
		found := false
		for _, c := range continuations {
			if strings.HasPrefix(rest, c) {
				depth++
				rest = rest[len(c):]
				found = true
				break
			}
		}
		if !found {
			break
		}
	}

	for _, b := range branches {
		if strings.HasPrefix(rest, b) {
			rest = strings.TrimSpace(rest[len(b):])
			at := strings.LastIndex(rest, "@")
			if at <= 0 {
				return 0, "", "", false
			}
			return depth, rest[:at], rest[at+1:], true
		}
	}
	return 0, "", "", false
}

// splitAtVersion splits "name@version" or "@scope/name@version".
func splitAtVersion(s string) (name, version string) {
	start := 0
	if strings.HasPrefix(s, "@") {
		start = 1
	}
	idx := strings.LastIndex(s[start:], "@")
	if idx < 0 {
		return s, ""
	}
	return s[:start+idx], s[start+idx+1:]
}

// parseLocator extracts (name, version) from a yarn berry locator such as
// "chalk@npm:5.3.0" or "@types/node@npm:18.0.0".
func parseLocator(locator string) (name, version string) {
	start := 0
	if strings.HasPrefix(locator, "@") {
		start = 1
	}
	subIdx := strings.Index(locator[start:], "@npm:")
	if subIdx < 0 {
		return locator, ""
	}
	idx := start + subIdx
	name = locator[:idx]
	version = locator[idx+5:] // skip "@npm:"
	if hashIdx := strings.Index(version, "#"); hashIdx >= 0 {
		version = version[:hashIdx]
	}
	return name, version
}
