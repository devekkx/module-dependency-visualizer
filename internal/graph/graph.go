package graph

// Stats holds aggregate metrics about a graph.
type Stats struct {
	NodeCount int
	EdgeCount int
	MaxDepth  int
	HasCycles bool
}

// Graph is an immutable view of a dependency graph.
// All accessor methods return copies so callers cannot mutate internal state.
type Graph struct {
	nodes     map[NodeID]Node
	edges     []Edge
	adjacency map[NodeID][]NodeID
	reverse   map[NodeID][]NodeID
	stats     Stats
}

// Nodes returns a copy of all nodes, sorted by ID for determinism.
func (g *Graph) Nodes() []Node {
	result := make([]Node, 0, len(g.nodes))
	for _, n := range g.nodes {
		result = append(result, n.Copy())
	}
	sortNodes(result)
	return result
}

// Edges returns a copy of all edges, sorted by (From, To) for determinism.
func (g *Graph) Edges() []Edge {
	result := make([]Edge, len(g.edges))
	for i, e := range g.edges {
		result[i] = e.Copy()
	}
	sortEdges(result)
	return result
}

// Node returns a copy of the node with the given ID, and whether it was found.
func (g *Graph) Node(id NodeID) (Node, bool) {
	n, ok := g.nodes[id]
	if !ok {
		return Node{}, false
	}
	return n.Copy(), true
}

// Neighbors returns a copy of the slice of node IDs that id directly depends on.
func (g *Graph) Neighbors(id NodeID) []NodeID {
	ids, ok := g.adjacency[id]
	if !ok {
		return nil
	}
	result := make([]NodeID, len(ids))
	copy(result, ids)
	return result
}

// Ancestors returns a copy of the slice of node IDs that depend on id (reverse edges).
func (g *Graph) Ancestors(id NodeID) []NodeID {
	ids, ok := g.reverse[id]
	if !ok {
		return nil
	}
	result := make([]NodeID, len(ids))
	copy(result, ids)
	return result
}

// Stats returns aggregate metrics about the graph.
func (g *Graph) Stats() Stats {
	return g.stats
}

// NodeCount returns the number of nodes in the graph.
func (g *Graph) NodeCount() int {
	return len(g.nodes)
}

// EdgeCount returns the number of edges in the graph.
func (g *Graph) EdgeCount() int {
	return len(g.edges)
}
