package node

import (
	"fmt"

	"github.com/devekkx/module-dependency-visualizer/internal/graph"
)

// DepNode is an intermediate tree representation of a package and its
// transitive dependencies. It is produced by the subprocess parsers and
// converted to a graph.Graph by buildGraph.
type DepNode struct {
	Name     string
	Version  string
	Dev      bool
	Children []DepNode
}

// buildGraph converts a DepNode tree into an immutable graph.Graph.
// It deduplicates nodes by ID and edges by (From, To), and stops recursing
// into a node whose children have already been processed (cycle guard).
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
				return fmt.Errorf("node: add node %s: %w", id, err)
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
		return nil, fmt.Errorf("node: build graph: %w", err)
	}
	return g, nil
}
