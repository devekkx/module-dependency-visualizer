package graph

// BFS performs a breadth-first traversal starting from each node in roots,
// following forward edges. It visits each reachable node at most once and
// calls visit(node, depth) for each. Returns early if visit returns false.
// maxDepth of -1 means unlimited.
func BFS(g *Graph, roots []NodeID, maxDepth int, visit func(NodeID, int) bool) {
	type entry struct {
		id    NodeID
		depth int
	}

	seen := make(map[NodeID]struct{})
	queue := make([]entry, 0, len(roots))

	for _, r := range roots {
		if _, ok := g.nodes[r]; ok {
			queue = append(queue, entry{r, 0})
			seen[r] = struct{}{}
		}
	}

	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]

		if !visit(cur.id, cur.depth) {
			return
		}

		if maxDepth >= 0 && cur.depth >= maxDepth {
			continue
		}

		for _, neighbor := range g.adjacency[cur.id] {
			if _, visited := seen[neighbor]; !visited {
				seen[neighbor] = struct{}{}
				queue = append(queue, entry{neighbor, cur.depth + 1})
			}
		}
	}
}

// Descendants returns all node IDs reachable from id via forward edges,
// excluding id itself. Honors maxDepth (-1 = unlimited).
func Descendants(g *Graph, id NodeID, maxDepth int) []NodeID {
	var result []NodeID
	BFS(g, []NodeID{id}, maxDepth, func(cur NodeID, depth int) bool {
		if depth > 0 {
			result = append(result, cur)
		}
		return true
	})
	sortNodeIDs(result)
	return result
}

// Ancestors returns all node IDs that can reach id via forward edges (reverse BFS).
func Ancestors(g *Graph, id NodeID) []NodeID {
	type entry struct {
		id NodeID
	}

	seen := make(map[NodeID]struct{})
	queue := []entry{{id}}
	seen[id] = struct{}{}

	var result []NodeID

	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]

		for _, parent := range g.reverse[cur.id] {
			if _, visited := seen[parent]; !visited {
				seen[parent] = struct{}{}
				queue = append(queue, entry{parent})
				result = append(result, parent)
			}
		}
	}

	sortNodeIDs(result)
	return result
}

// computeMaxDepth returns the longest shortest path from any root node
// (a node with no incoming edges) using BFS.
func computeMaxDepth(g *Graph) int {
	// Find all root nodes (no incoming edges).
	roots := make([]NodeID, 0)
	for id := range g.nodes {
		if len(g.reverse[id]) == 0 {
			roots = append(roots, id)
		}
	}

	if len(roots) == 0 {
		// All nodes are in cycles; depth is defined as 0.
		return 0
	}

	maxDepth := 0
	BFS(g, roots, -1, func(_ NodeID, depth int) bool {
		if depth > maxDepth {
			maxDepth = depth
		}
		return true
	})
	return maxDepth
}

// detectCycles reports whether the graph contains at least one cycle using
// iterative DFS with a three-color marking scheme.
func detectCycles(g *Graph) bool {
	const (
		white = 0 // unvisited
		gray  = 1 // in current DFS path
		black = 2 // fully processed
	)

	color := make(map[NodeID]int, len(g.nodes))

	type frame struct {
		id       NodeID
		childIdx int
	}

	for start := range g.nodes {
		if color[start] != white {
			continue
		}

		stack := []frame{{id: start}}
		color[start] = gray

		for len(stack) > 0 {
			top := &stack[len(stack)-1]
			neighbors := g.adjacency[top.id]

			if top.childIdx >= len(neighbors) {
				color[top.id] = black
				stack = stack[:len(stack)-1]
				continue
			}

			child := neighbors[top.childIdx]
			top.childIdx++

			if color[child] == gray {
				return true
			}
			if color[child] == white {
				color[child] = gray
				stack = append(stack, frame{id: child})
			}
		}
	}

	return false
}
