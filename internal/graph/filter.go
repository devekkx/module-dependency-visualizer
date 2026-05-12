package graph

import "regexp"

// FilterOptions controls which nodes and edges are included in a filtered graph.
type FilterOptions struct {
	// MaxDepth limits traversal depth from root nodes. -1 means unlimited.
	MaxDepth int

	// IncludePatterns are regular expressions; a node is kept only if its Name
	// matches at least one pattern. Empty means keep all.
	IncludePatterns []*regexp.Regexp

	// ExcludePatterns are regular expressions; a node is removed if its Name
	// matches any pattern.
	ExcludePatterns []*regexp.Regexp

	// NoIndirect removes nodes that are only reachable as indirect dependencies.
	NoIndirect bool
}

// Filter returns a new Graph containing only the nodes (and their connecting
// edges) that satisfy opts. The original graph is not modified.
func Filter(g *Graph, opts FilterOptions) (*Graph, error) {
	allowed := make(map[NodeID]struct{})

	if opts.MaxDepth != -1 {
		// Find roots (no incoming edges).
		var roots []NodeID
		for _, n := range g.Nodes() {
			if len(g.reverse[n.ID]) == 0 {
				roots = append(roots, n.ID)
			}
		}
		BFS(g, roots, opts.MaxDepth, func(id NodeID, _ int) bool {
			allowed[id] = struct{}{}
			return true
		})
	} else {
		for _, n := range g.Nodes() {
			allowed[n.ID] = struct{}{}
		}
	}

	b := NewBuilder()

	for id := range allowed {
		n, ok := g.nodes[id]
		if !ok {
			continue
		}

		if opts.NoIndirect && (n.Indirect || n.Dev) {
			delete(allowed, id)
			continue
		}

		if len(opts.IncludePatterns) > 0 && !matchesAny(n.Name, opts.IncludePatterns) {
			delete(allowed, id)
			continue
		}

		if matchesAny(n.Name, opts.ExcludePatterns) {
			delete(allowed, id)
			continue
		}

		if err := b.AddNode(n); err != nil {
			return nil, err
		}
	}

	for _, e := range g.edges {
		_, fromOK := allowed[e.From]
		_, toOK := allowed[e.To]
		if fromOK && toOK {
			if err := b.AddEdge(e); err != nil {
				return nil, err
			}
		}
	}

	return b.Build()
}

func matchesAny(name string, patterns []*regexp.Regexp) bool {
	for _, p := range patterns {
		if p.MatchString(name) {
			return true
		}
	}
	return false
}
