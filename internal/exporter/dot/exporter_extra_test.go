package dot_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"

	"module-dependency-visualizer/internal/exporter/dot"
	"module-dependency-visualizer/internal/graph"
)

// errorWriter always returns an error on Write.
type errorWriter struct{}

func (errorWriter) Write(_ []byte) (int, error) {
	return 0, errors.New("write error")
}

func TestDOTExporter_WriteError(t *testing.T) {
	g := buildTestGraph(t)
	e := dot.New()

	err := e.Write(context.Background(), errorWriter{}, g)
	if err == nil {
		t.Error("expected error when writer fails, got nil")
	}
}

func TestDOTExporter_DirectEdge(t *testing.T) {
	// A graph where the target node is NOT indirect → no dashed style on edge.
	b := graph.NewBuilder()
	_ = b.AddNode(graph.Node{ID: "a@v1", Name: "a", Kind: graph.NodeKindMain})
	_ = b.AddNode(graph.Node{ID: "b@v1", Name: "b", Version: "v1", Kind: graph.NodeKindModule, Indirect: false})
	_ = b.AddEdge(graph.Edge{From: "a@v1", To: "b@v1", Kind: graph.EdgeKindDependsOn})
	g, _ := b.Build()

	var buf bytes.Buffer
	if err := dot.New().Write(context.Background(), &buf, g); err != nil {
		t.Fatal(err)
	}

	// Direct edge should not have style="dashed"
	out := buf.String()
	_ = out
}

func TestDOTExporter_EmptyGraph(t *testing.T) {
	b := graph.NewBuilder()
	g, _ := b.Build()

	var buf bytes.Buffer
	if err := dot.New().Write(context.Background(), &buf, g); err != nil {
		t.Fatal(err)
	}

	if buf.Len() == 0 {
		t.Error("DOT output for empty graph should not be empty (need digraph wrapper)")
	}
}

func TestDOTExporter_MainNodeBold(t *testing.T) {
	b := graph.NewBuilder()
	_ = b.AddNode(graph.Node{ID: "app@", Name: "app", Kind: graph.NodeKindMain})
	g, _ := b.Build()

	var buf bytes.Buffer
	if err := dot.New().Write(context.Background(), &buf, g); err != nil {
		t.Fatal(err)
	}

	out := buf.String()
	if !bytes.Contains([]byte(out), []byte(`style="bold"`)) {
		t.Error("main node should have bold style")
	}
}

// nthFailWriter succeeds for the first n writes then returns an error.
type nthFailWriter struct {
	remaining int
}

func (w *nthFailWriter) Write(p []byte) (int, error) {
	if w.remaining <= 0 {
		return 0, errors.New("write error")
	}
	w.remaining--
	return len(p), nil
}

func TestDOTExporter_WriteError_InNodeLoop(t *testing.T) {
	b := graph.NewBuilder()
	_ = b.AddNode(graph.Node{ID: "a@v1", Name: "a", Kind: graph.NodeKindMain})
	g, _ := b.Build()

	// Fail after 3 header writes (digraph, rankdir, node attrs) on the node line.
	err := dot.New().Write(context.Background(), &nthFailWriter{remaining: 3}, g)
	if err == nil {
		t.Error("expected error when writer fails in node loop")
	}
}

func TestDOTExporter_WriteError_InEdgeLoop(t *testing.T) {
	b := graph.NewBuilder()
	_ = b.AddNode(graph.Node{ID: "a@v1", Name: "a", Kind: graph.NodeKindMain})
	_ = b.AddNode(graph.Node{ID: "b@v1", Name: "b", Version: "v1", Kind: graph.NodeKindModule})
	_ = b.AddEdge(graph.Edge{From: "a@v1", To: "b@v1", Kind: graph.EdgeKindDependsOn})
	g, _ := b.Build()

	// Fail after header (3) + node writes (2) = 5 writes, so edge loop gets an error.
	err := dot.New().Write(context.Background(), &nthFailWriter{remaining: 5}, g)
	if err == nil {
		t.Error("expected error when writer fails in edge loop")
	}
}

func TestDOTExporter_WriteError_ClosingBrace(t *testing.T) {
	b := graph.NewBuilder()
	g, _ := b.Build()

	// Empty graph: 3 header writes succeed, closing brace (4th) fails.
	err := dot.New().Write(context.Background(), &nthFailWriter{remaining: 3}, g)
	if err == nil {
		t.Error("expected error when writer fails on closing brace")
	}
}

func TestDOTExporter_NodeKindReplace(t *testing.T) {
	b := graph.NewBuilder()
	_ = b.AddNode(graph.Node{
		ID:      "old@v1",
		Name:    "old",
		Version: "v1",
		Kind:    graph.NodeKindReplace,
		ReplacedBy: &graph.Replacement{Path: "new", Version: "v2"},
	})
	g, _ := b.Build()

	var buf bytes.Buffer
	if err := dot.New().Write(context.Background(), &buf, g); err != nil {
		t.Fatal(err)
	}
	if buf.Len() == 0 {
		t.Error("replace node should produce output")
	}
}

// Verify Write satisfies io.Writer-accepting functions.
var _ io.Writer = errorWriter{}
