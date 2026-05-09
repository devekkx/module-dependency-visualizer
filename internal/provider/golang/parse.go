package golang

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/devekkx/module-dependency-visualizer/internal/graph"
)

// ModGraphEdge represents a single line from `go mod graph` output.
type ModGraphEdge struct {
	From string
	To   string
}

// ParseModGraph parses the output of `go mod graph`.
// Each line is "<from> <to>" where from/to are "<path>@<version>" (or just
// "<path>" for the main module).
func ParseModGraph(data []byte) ([]ModGraphEdge, error) {
	var edges []ModGraphEdge

	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) != 2 {
			return nil, fmt.Errorf("go mod graph: unexpected line format: %q", line)
		}
		edges = append(edges, ModGraphEdge{From: parts[0], To: parts[1]})
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("go mod graph: scan: %w", err)
	}
	return edges, nil
}

// ModuleInfo represents a single entry from `go list -m -json all`.
type ModuleInfo struct {
	Path     string      `json:"Path"`
	Version  string      `json:"Version"`
	Main     bool        `json:"Main"`
	Indirect bool        `json:"Indirect"`
	Replace  *ModuleInfo `json:"Replace,omitempty"`
}

// ParseModList parses the JSON stream output of `go list -m -json all`.
// The output is a stream of JSON objects (not a JSON array).
func ParseModList(data []byte) ([]ModuleInfo, error) {
	var infos []ModuleInfo

	dec := json.NewDecoder(bytes.NewReader(data))
	for dec.More() {
		var info ModuleInfo
		if err := dec.Decode(&info); err != nil {
			return nil, fmt.Errorf("go list: decode: %w", err)
		}
		infos = append(infos, info)
	}
	return infos, nil
}

// moduleRef normalizes a module reference from `go mod graph` into a consistent
// "<path>@<version>" form. The main module is emitted as "<path>" (no @), so
// we normalize it to "<path>@".
func normalizeRef(ref string) string {
	if !strings.Contains(ref, "@") {
		return ref + "@"
	}
	return ref
}

// splitRef splits a "<path>@<version>" reference into its components.
func splitRef(ref string) (path, version string) {
	ref = normalizeRef(ref)
	idx := strings.LastIndex(ref, "@")
	return ref[:idx], ref[idx+1:]
}

// buildGraph merges go mod graph edges with go list metadata to produce
// an immutable graph.
func buildGraph(edges []ModGraphEdge, infos []ModuleInfo) (*graph.Graph, string, error) {
	// Index infos by path for O(1) lookup.
	byPath := make(map[string]ModuleInfo, len(infos))
	var mainModule string

	for _, info := range infos {
		byPath[info.Path] = info
		if info.Main {
			mainModule = info.Path
		}
	}

	b := graph.NewBuilder()

	// Collect all unique node refs from edges.
	nodeRefs := make(map[string]struct{})
	for _, e := range edges {
		nodeRefs[normalizeRef(e.From)] = struct{}{}
		nodeRefs[normalizeRef(e.To)] = struct{}{}
	}

	for ref := range nodeRefs {
		path, version := splitRef(ref)
		info := byPath[path]

		kind := graph.NodeKindModule
		if info.Main {
			kind = graph.NodeKindMain
		}

		n := graph.Node{
			ID:       graph.NodeID(ref),
			Name:     path,
			Version:  version,
			Kind:     kind,
			Indirect: info.Indirect,
		}

		if info.Replace != nil {
			n.Kind = graph.NodeKindReplace
			n.ReplacedBy = &graph.Replacement{
				Path:    info.Replace.Path,
				Version: info.Replace.Version,
			}
		}

		if err := b.AddNode(n); err != nil {
			return nil, "", fmt.Errorf("add node %q: %w", ref, err)
		}
	}

	for _, e := range edges {
		from := normalizeRef(e.From)
		to := normalizeRef(e.To)
		edge := graph.Edge{
			From: graph.NodeID(from),
			To:   graph.NodeID(to),
			Kind: graph.EdgeKindDependsOn,
		}
		if err := b.AddEdge(edge); err != nil {
			return nil, "", fmt.Errorf("add edge %q->%q: %w", from, to, err)
		}
	}

	g, err := b.Build()
	if err != nil {
		return nil, "", fmt.Errorf("build graph: %w", err)
	}
	return g, mainModule, nil
}
