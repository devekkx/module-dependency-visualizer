package graph_test

import (
	"testing"

	"github.com/devekkx/module-dependency-visualizer/internal/graph"
)

// buildTwoNodeGraph builds: A -> B
func buildTwoNodeGraph(t *testing.T) (*graph.Graph, graph.NodeID, graph.NodeID) {
	t.Helper()
	b := graph.NewBuilder()
	a := graph.Node{ID: "a@v1", Name: "a", Version: "v1", Kind: graph.NodeKindMain}
	bNode := graph.Node{ID: "b@v2", Name: "b", Version: "v2", Kind: graph.NodeKindModule}
	_ = b.AddNode(a)
	_ = b.AddNode(bNode)
	_ = b.AddEdge(graph.Edge{From: "a@v1", To: "b@v2", Kind: graph.EdgeKindDependsOn})
	g, err := b.Build()
	if err != nil {
		t.Fatal(err)
	}
	return g, a.ID, bNode.ID
}

func TestGraph_Edges_ReturnsCopy(t *testing.T) {
	g, _, _ := buildTwoNodeGraph(t)

	edges := g.Edges()
	if len(edges) != 1 {
		t.Fatalf("Edges() len = %d; want 1", len(edges))
	}

	// Mutating the returned slice should not affect the graph.
	orig := edges[0].Kind
	edges[0].Kind = "mutated"
	fresh := g.Edges()
	if fresh[0].Kind != orig {
		t.Error("Edges() returned a mutable reference")
	}
}

func TestGraph_Edges_SortedByFromTo(t *testing.T) {
	b := graph.NewBuilder()
	for _, n := range []graph.Node{
		{ID: "a@v1", Name: "a", Kind: graph.NodeKindMain},
		{ID: "b@v1", Name: "b", Kind: graph.NodeKindModule},
		{ID: "c@v1", Name: "c", Kind: graph.NodeKindModule},
	} {
		_ = b.AddNode(n)
	}
	_ = b.AddEdge(graph.Edge{From: "a@v1", To: "c@v1", Kind: graph.EdgeKindDependsOn})
	_ = b.AddEdge(graph.Edge{From: "a@v1", To: "b@v1", Kind: graph.EdgeKindDependsOn})
	g, _ := b.Build()

	edges := g.Edges()
	if len(edges) != 2 {
		t.Fatalf("expected 2 edges, got %d", len(edges))
	}
	if edges[0].To != "b@v1" || edges[1].To != "c@v1" {
		t.Errorf("edges not sorted: got %s, %s", edges[0].To, edges[1].To)
	}
}

func TestGraph_Node_Found(t *testing.T) {
	g, aID, _ := buildTwoNodeGraph(t)

	n, ok := g.Node(aID)
	if !ok {
		t.Fatalf("Node(%q) not found", aID)
	}
	if n.ID != aID {
		t.Errorf("Node ID = %q; want %q", n.ID, aID)
	}
}

func TestGraph_Node_NotFound(t *testing.T) {
	g, _, _ := buildTwoNodeGraph(t)

	_, ok := g.Node("nonexistent@v1")
	if ok {
		t.Error("Node() returned true for unknown ID")
	}
}

func TestGraph_Node_ReturnsCopy(t *testing.T) {
	g, aID, _ := buildTwoNodeGraph(t)

	n, _ := g.Node(aID)
	n.Name = "mutated"

	fresh, _ := g.Node(aID)
	if fresh.Name == "mutated" {
		t.Error("Node() returned a mutable reference")
	}
}

func TestGraph_Neighbors(t *testing.T) {
	g, aID, bID := buildTwoNodeGraph(t)

	neighbors := g.Neighbors(aID)
	if len(neighbors) != 1 || neighbors[0] != bID {
		t.Errorf("Neighbors(%q) = %v; want [%q]", aID, neighbors, bID)
	}
}

func TestGraph_Neighbors_ReturnsCopy(t *testing.T) {
	g, aID, _ := buildTwoNodeGraph(t)

	n1 := g.Neighbors(aID)
	n1[0] = "mutated@v1"
	n2 := g.Neighbors(aID)
	if n2[0] == "mutated@v1" {
		t.Error("Neighbors() returned a mutable reference")
	}
}

func TestGraph_Neighbors_UnknownNode(t *testing.T) {
	g, _, _ := buildTwoNodeGraph(t)
	if got := g.Neighbors("unknown@v1"); got != nil {
		t.Errorf("Neighbors() on unknown node = %v; want nil", got)
	}
}

func TestGraph_Ancestors_Method(t *testing.T) {
	g, aID, bID := buildTwoNodeGraph(t)

	anc := g.Ancestors(bID)
	if len(anc) != 1 || anc[0] != aID {
		t.Errorf("Ancestors(%q) = %v; want [%q]", bID, anc, aID)
	}
}

func TestGraph_Ancestors_Unknown(t *testing.T) {
	g, _, _ := buildTwoNodeGraph(t)
	if got := g.Ancestors("unknown@v1"); got != nil {
		t.Errorf("Ancestors() on unknown node = %v; want nil", got)
	}
}

func TestGraph_NodeCount_EdgeCount(t *testing.T) {
	g, _, _ := buildTwoNodeGraph(t)

	if g.NodeCount() != 2 {
		t.Errorf("NodeCount() = %d; want 2", g.NodeCount())
	}
	if g.EdgeCount() != 1 {
		t.Errorf("EdgeCount() = %d; want 1", g.EdgeCount())
	}
}
