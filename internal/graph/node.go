package graph

import (
	"fmt"
	"maps"
)

// NodeID is the canonical identifier for a node: "<module-path>@<version>".
// The main module uses an empty version: "example.com/myapp@".
type NodeID string

// NewNodeID constructs a NodeID from a module path and version.
func NewNodeID(path, version string) NodeID {
	return NodeID(fmt.Sprintf("%s@%s", path, version))
}

// NodeKind classifies a node within the dependency graph.
type NodeKind string

const (
	NodeKindMain    NodeKind = "main"
	NodeKindModule  NodeKind = "module"
	NodeKindReplace NodeKind = "replace"
)

// Replacement describes a module that stands in for another.
type Replacement struct {
	Path    string
	Version string
}

// Node is an immutable value representing a single module in the graph.
type Node struct {
	ID          NodeID
	Name        string
	Version     string
	Kind        NodeKind
	Indirect    bool
	ReplacedBy  *Replacement
	Metadata    map[string]any
}

// metadataCopy returns a shallow copy of the metadata map so callers cannot
// mutate the node's internal state.
func metadataCopy(src map[string]any) map[string]any {
	if src == nil {
		return nil
	}
	return maps.Clone(src)
}

// Copy returns a deep-enough copy that callers cannot mutate this Node.
func (n Node) Copy() Node {
	n.Metadata = metadataCopy(n.Metadata)
	return n
}
