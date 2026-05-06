package graph

import "fmt"

// Builder accumulates nodes and edges, then produces an immutable Graph.
// Builder is not safe for concurrent use.
type Builder struct {
	nodes map[NodeID]Node
	edges []Edge
}

// NewBuilder returns an empty Builder.
func NewBuilder() *Builder {
	return &Builder{
		nodes: make(map[NodeID]Node),
	}
}

// AddNode registers a node. Returns ErrDuplicateNode if the ID already exists.
func (b *Builder) AddNode(n Node) error {
	if _, exists := b.nodes[n.ID]; exists {
		return fmt.Errorf("%w: %s", ErrDuplicateNode, n.ID)
	}
	b.nodes[n.ID] = n.Copy()
	return nil
}

// AddEdge registers a directed edge. Returns ErrUnknownNode if either endpoint
// is not already registered. Returns ErrSelfLoop if From == To.
func (b *Builder) AddEdge(e Edge) error {
	if e.From == e.To {
		return fmt.Errorf("%w: %s", ErrSelfLoop, e.From)
	}
	if _, ok := b.nodes[e.From]; !ok {
		return fmt.Errorf("%w: from=%s", ErrUnknownNode, e.From)
	}
	if _, ok := b.nodes[e.To]; !ok {
		return fmt.Errorf("%w: to=%s", ErrUnknownNode, e.To)
	}
	b.edges = append(b.edges, e.Copy())
	return nil
}

// Build constructs an immutable Graph from the accumulated nodes and edges.
// It computes adjacency lists, reverse adjacency, max depth, and cycle detection.
func (b *Builder) Build() (*Graph, error) {
	adjacency := make(map[NodeID][]NodeID, len(b.nodes))
	reverse := make(map[NodeID][]NodeID, len(b.nodes))

	for id := range b.nodes {
		adjacency[id] = nil
		reverse[id] = nil
	}

	edges := make([]Edge, len(b.edges))
	for i, e := range b.edges {
		edges[i] = e.Copy()
		adjacency[e.From] = append(adjacency[e.From], e.To)
		reverse[e.To] = append(reverse[e.To], e.From)
	}

	for id := range adjacency {
		sortNodeIDs(adjacency[id])
		sortNodeIDs(reverse[id])
	}

	g := &Graph{
		nodes:     b.nodes,
		edges:     edges,
		adjacency: adjacency,
		reverse:   reverse,
	}

	g.stats = Stats{
		NodeCount: len(b.nodes),
		EdgeCount: len(b.edges),
		MaxDepth:  computeMaxDepth(g),
		HasCycles: detectCycles(g),
	}

	return g, nil
}
