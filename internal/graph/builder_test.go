package graph_test

import (
	"errors"
	"testing"

	"github.com/devekkx/module-dependency-visualizer/internal/graph"
)

func makeNode(path, version string, kind graph.NodeKind) graph.Node {
	return graph.Node{
		ID:      graph.NewNodeID(path, version),
		Name:    path,
		Version: version,
		Kind:    kind,
	}
}

func makeEdge(from, to graph.NodeID) graph.Edge {
	return graph.Edge{From: from, To: to, Kind: graph.EdgeKindDependsOn}
}

func TestBuilder_AddNode_RejectsDuplicate(t *testing.T) {
	b := graph.NewBuilder()
	n := makeNode("example.com/foo", "v1.0.0", graph.NodeKindModule)

	if err := b.AddNode(n); err != nil {
		t.Fatalf("first AddNode failed: %v", err)
	}
	if err := b.AddNode(n); !errors.Is(err, graph.ErrDuplicateNode) {
		t.Errorf("expected ErrDuplicateNode, got %v", err)
	}
}

func TestBuilder_AddEdge_RejectsUnknownNodes(t *testing.T) {
	b := graph.NewBuilder()
	_ = b.AddNode(makeNode("a.com/a", "v1", graph.NodeKindMain))

	cases := []struct {
		name string
		from graph.NodeID
		to   graph.NodeID
	}{
		{
			name: "unknown from",
			from: graph.NewNodeID("unknown.com/x", "v1"),
			to:   graph.NewNodeID("a.com/a", "v1"),
		},
		{
			name: "unknown to",
			from: graph.NewNodeID("a.com/a", "v1"),
			to:   graph.NewNodeID("unknown.com/y", "v1"),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			e := graph.Edge{From: tc.from, To: tc.to, Kind: graph.EdgeKindDependsOn}
			if err := b.AddEdge(e); !errors.Is(err, graph.ErrUnknownNode) {
				t.Errorf("expected ErrUnknownNode, got %v", err)
			}
		})
	}
}

func TestBuilder_AddEdge_RejectsSelfLoop(t *testing.T) {
	b := graph.NewBuilder()
	n := makeNode("example.com/a", "v1", graph.NodeKindModule)
	_ = b.AddNode(n)

	e := graph.Edge{From: n.ID, To: n.ID, Kind: graph.EdgeKindDependsOn}
	if err := b.AddEdge(e); !errors.Is(err, graph.ErrSelfLoop) {
		t.Errorf("expected ErrSelfLoop, got %v", err)
	}
}

func TestBuilder_Build_EmptyGraph(t *testing.T) {
	b := graph.NewBuilder()
	g, err := b.Build()
	if err != nil {
		t.Fatalf("Build() error: %v", err)
	}
	s := g.Stats()
	if s.NodeCount != 0 || s.EdgeCount != 0 || s.MaxDepth != 0 || s.HasCycles {
		t.Errorf("empty graph stats unexpected: %+v", s)
	}
}

func TestBuilder_Build_ImmutableNodes(t *testing.T) {
	b := graph.NewBuilder()
	n := makeNode("example.com/x", "v1.0.0", graph.NodeKindModule)
	n.Metadata = map[string]any{"k": "v"}
	_ = b.AddNode(n)

	g, _ := b.Build()
	nodes := g.Nodes()
	if len(nodes) != 1 {
		t.Fatalf("expected 1 node, got %d", len(nodes))
	}

	// Mutating the returned slice/metadata should not affect the graph.
	nodes[0].Metadata["k"] = "mutated"
	fresh := g.Nodes()
	if fresh[0].Metadata["k"] != "v" {
		t.Error("Nodes() returned a mutable reference — immutability violated")
	}
}

func TestBuilder_Build_Stats(t *testing.T) {
	g := buildLinearChain(t, 4) // main -> a -> b -> c

	s := g.Stats()
	if s.NodeCount != 4 {
		t.Errorf("NodeCount = %d; want 4", s.NodeCount)
	}
	if s.EdgeCount != 3 {
		t.Errorf("EdgeCount = %d; want 3", s.EdgeCount)
	}
	if s.MaxDepth != 3 {
		t.Errorf("MaxDepth = %d; want 3", s.MaxDepth)
	}
	if s.HasCycles {
		t.Error("HasCycles = true; want false for linear chain")
	}
}

func TestBuilder_Build_CycleDetection(t *testing.T) {
	b := graph.NewBuilder()
	a := makeNode("a.com/a", "v1", graph.NodeKindModule)
	c := makeNode("a.com/c", "v1", graph.NodeKindModule)
	_ = b.AddNode(a)
	_ = b.AddNode(c)
	_ = b.AddEdge(makeEdge(a.ID, c.ID))
	_ = b.AddEdge(makeEdge(c.ID, a.ID)) // cycle

	g, _ := b.Build()
	if !g.Stats().HasCycles {
		t.Error("HasCycles = false; expected true for a->c->a cycle")
	}
}

// buildLinearChain creates: root -> n0 -> n1 -> ... -> n(count-2)
func buildLinearChain(t *testing.T, count int) *graph.Graph {
	t.Helper()
	b := graph.NewBuilder()

	nodes := make([]graph.Node, count)
	nodes[0] = makeNode("root.com/main", "", graph.NodeKindMain)
	for i := 1; i < count; i++ {
		nodes[i] = makeNode("dep.com/dep", string(rune('a'-1+i)), graph.NodeKindModule)
		nodes[i].ID = graph.NewNodeID("dep.com/dep", string(rune('a'-1+i)))
	}

	for _, n := range nodes {
		if err := b.AddNode(n); err != nil {
			t.Fatalf("AddNode: %v", err)
		}
	}
	for i := 0; i < count-1; i++ {
		if err := b.AddEdge(makeEdge(nodes[i].ID, nodes[i+1].ID)); err != nil {
			t.Fatalf("AddEdge: %v", err)
		}
	}

	g, err := b.Build()
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	return g
}
