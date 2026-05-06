package graph

import (
	"cmp"
	"slices"
)

func sortNodes(nodes []Node) {
	slices.SortFunc(nodes, func(a, b Node) int {
		return cmp.Compare(string(a.ID), string(b.ID))
	})
}

func sortEdges(edges []Edge) {
	slices.SortFunc(edges, func(a, b Edge) int {
		if a.From != b.From {
			return cmp.Compare(string(a.From), string(b.From))
		}
		return cmp.Compare(string(a.To), string(b.To))
	})
}

func sortNodeIDs(ids []NodeID) {
	slices.SortFunc(ids, func(a, b NodeID) int {
		return cmp.Compare(string(a), string(b))
	})
}
