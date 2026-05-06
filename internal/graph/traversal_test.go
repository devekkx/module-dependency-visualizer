package graph_test

import (
	"testing"

	"module-dependency-visualizer/internal/graph"
)

func buildDiamondGraph(t *testing.T) *graph.Graph {
	t.Helper()
	//   root
	//   / \
	//  A   B
	//   \ /
	//    C
	b := graph.NewBuilder()
	root := graph.Node{ID: "root@", Name: "root", Kind: graph.NodeKindMain}
	a := graph.Node{ID: "a@v1", Name: "a", Version: "v1", Kind: graph.NodeKindModule}
	bnode := graph.Node{ID: "b@v1", Name: "b", Version: "v1", Kind: graph.NodeKindModule}
	c := graph.Node{ID: "c@v1", Name: "c", Version: "v1", Kind: graph.NodeKindModule}

	for _, n := range []graph.Node{root, a, bnode, c} {
		if err := b.AddNode(n); err != nil {
			t.Fatal(err)
		}
	}
	for _, e := range []graph.Edge{
		{From: "root@", To: "a@v1", Kind: graph.EdgeKindDependsOn},
		{From: "root@", To: "b@v1", Kind: graph.EdgeKindDependsOn},
		{From: "a@v1", To: "c@v1", Kind: graph.EdgeKindDependsOn},
		{From: "b@v1", To: "c@v1", Kind: graph.EdgeKindDependsOn},
	} {
		if err := b.AddEdge(e); err != nil {
			t.Fatal(err)
		}
	}
	g, err := b.Build()
	if err != nil {
		t.Fatal(err)
	}
	return g
}

func TestBFS_VisitsAllNodes(t *testing.T) {
	g := buildDiamondGraph(t)

	visited := make(map[graph.NodeID]int)
	graph.BFS(g, []graph.NodeID{"root@"}, -1, func(id graph.NodeID, depth int) bool {
		visited[id] = depth
		return true
	})

	cases := []struct {
		id        graph.NodeID
		wantDepth int
	}{
		{"root@", 0},
		{"a@v1", 1},
		{"b@v1", 1},
		{"c@v1", 2},
	}

	for _, tc := range cases {
		depth, ok := visited[tc.id]
		if !ok {
			t.Errorf("BFS did not visit %s", tc.id)
		} else if depth != tc.wantDepth {
			t.Errorf("BFS depth for %s = %d; want %d", tc.id, depth, tc.wantDepth)
		}
	}
}

func TestBFS_RespectsMaxDepth(t *testing.T) {
	g := buildDiamondGraph(t)

	var visited []graph.NodeID
	graph.BFS(g, []graph.NodeID{"root@"}, 1, func(id graph.NodeID, _ int) bool {
		visited = append(visited, id)
		return true
	})

	for _, id := range visited {
		if id == "c@v1" {
			t.Error("BFS with maxDepth=1 should not reach c@v1")
		}
	}
}

func TestDescendants(t *testing.T) {
	g := buildDiamondGraph(t)

	desc := graph.Descendants(g, "root@", -1)
	wantSet := map[graph.NodeID]bool{"a@v1": true, "b@v1": true, "c@v1": true}

	if len(desc) != 3 {
		t.Fatalf("Descendants count = %d; want 3", len(desc))
	}
	for _, id := range desc {
		if !wantSet[id] {
			t.Errorf("unexpected descendant: %s", id)
		}
	}
}

func TestAncestors(t *testing.T) {
	g := buildDiamondGraph(t)

	anc := graph.Ancestors(g, "c@v1")
	wantSet := map[graph.NodeID]bool{"root@": true, "a@v1": true, "b@v1": true}

	if len(anc) != 3 {
		t.Fatalf("Ancestors count = %d; want 3: %v", len(anc), anc)
	}
	for _, id := range anc {
		if !wantSet[id] {
			t.Errorf("unexpected ancestor: %s", id)
		}
	}
}

func TestBFS_EarlyTermination(t *testing.T) {
	g := buildDiamondGraph(t)

	count := 0
	graph.BFS(g, []graph.NodeID{"root@"}, -1, func(_ graph.NodeID, _ int) bool {
		count++
		return count < 2 // stop after visiting 2 nodes
	})

	if count != 2 {
		t.Errorf("expected early stop at 2 nodes, got %d", count)
	}
}
