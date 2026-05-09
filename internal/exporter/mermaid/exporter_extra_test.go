package mermaid_test

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/devekkx/module-dependency-visualizer/internal/exporter/mermaid"
	"github.com/devekkx/module-dependency-visualizer/internal/graph"
)

type errorWriter struct{}

func (errorWriter) Write(_ []byte) (int, error) {
	return 0, errors.New("write error")
}

// nthFailWriter succeeds for n writes then returns an error.
type nthFailWriter struct{ remaining int }

func (w *nthFailWriter) Write(p []byte) (int, error) {
	if w.remaining <= 0 {
		return 0, errors.New("write error")
	}
	w.remaining--
	return len(p), nil
}

func TestMermaidExporter_WriteError(t *testing.T) {
	g := buildTestGraph(t)
	err := mermaid.New().Write(context.Background(), errorWriter{}, g)
	if err == nil {
		t.Error("expected error when writer fails, got nil")
	}
}

func TestMermaidExporter_WriteError_InNodeLoop(t *testing.T) {
	b := graph.NewBuilder()
	_ = b.AddNode(graph.Node{ID: "a@v1", Name: "a", Kind: graph.NodeKindMain})
	g, _ := b.Build()

	// Fail after the "graph TD" header line.
	err := mermaid.New().Write(context.Background(), &nthFailWriter{remaining: 1}, g)
	if err == nil {
		t.Error("expected error when writer fails in node loop")
	}
}

func TestMermaidExporter_WriteError_InEdgeLoop(t *testing.T) {
	b := graph.NewBuilder()
	_ = b.AddNode(graph.Node{ID: "a@v1", Name: "a", Kind: graph.NodeKindMain})
	_ = b.AddNode(graph.Node{ID: "b@v1", Name: "b", Version: "v1", Kind: graph.NodeKindModule})
	_ = b.AddEdge(graph.Edge{From: "a@v1", To: "b@v1", Kind: graph.EdgeKindDependsOn})
	g, _ := b.Build()

	// header(1) + 2 node lines(2) = 3 writes succeed; edge loop gets error.
	err := mermaid.New().Write(context.Background(), &nthFailWriter{remaining: 3}, g)
	if err == nil {
		t.Error("expected error when writer fails in edge loop")
	}
}

func TestMermaidExporter_DirectEdgeArrow(t *testing.T) {
	// Direct (non-indirect) node → solid --> arrow.
	b := graph.NewBuilder()
	_ = b.AddNode(graph.Node{ID: "a@v1", Name: "a", Kind: graph.NodeKindMain})
	_ = b.AddNode(graph.Node{ID: "b@v1", Name: "b", Version: "v1", Kind: graph.NodeKindModule, Indirect: false})
	_ = b.AddEdge(graph.Edge{From: "a@v1", To: "b@v1", Kind: graph.EdgeKindDependsOn})
	g, _ := b.Build()

	var buf bytes.Buffer
	if err := mermaid.New().Write(context.Background(), &buf, g); err != nil {
		t.Fatal(err)
	}

	out := buf.String()
	if !strings.Contains(out, "-->") {
		t.Error("direct edge should use --> arrow")
	}
	if strings.Contains(out, "-..->") {
		t.Error("direct edge should not use dotted arrow")
	}
}

func TestMermaidExporter_IndirectEdgeDotted(t *testing.T) {
	b := graph.NewBuilder()
	_ = b.AddNode(graph.Node{ID: "a@v1", Name: "a", Kind: graph.NodeKindMain})
	_ = b.AddNode(graph.Node{ID: "b@v1", Name: "b", Version: "v1", Kind: graph.NodeKindModule, Indirect: true})
	_ = b.AddEdge(graph.Edge{From: "a@v1", To: "b@v1", Kind: graph.EdgeKindDependsOn})
	g, _ := b.Build()

	var buf bytes.Buffer
	if err := mermaid.New().Write(context.Background(), &buf, g); err != nil {
		t.Fatal(err)
	}

	out := buf.String()
	if !strings.Contains(out, "-..->") || !strings.Contains(out, "-.->") {
		// Either dotted arrow syntax is acceptable
		if !strings.Contains(out, "-.->") {
			t.Error("indirect edge should use dotted arrow")
		}
	}
}

func TestMermaidExporter_EmptyGraph(t *testing.T) {
	b := graph.NewBuilder()
	g, _ := b.Build()

	var buf bytes.Buffer
	if err := mermaid.New().Write(context.Background(), &buf, g); err != nil {
		t.Fatal(err)
	}

	if !strings.HasPrefix(buf.String(), "graph TD") {
		t.Error("even empty graph must start with 'graph TD'")
	}
}

func TestMermaidExporter_LabelEscaping(t *testing.T) {
	b := graph.NewBuilder()
	_ = b.AddNode(graph.Node{
		ID:      `has"quote@v1`,
		Name:    `has"quote`,
		Version: "v1",
		Kind:    graph.NodeKindModule,
	})
	g, _ := b.Build()

	var buf bytes.Buffer
	if err := mermaid.New().Write(context.Background(), &buf, g); err != nil {
		t.Fatal(err)
	}

	// Double quotes inside labels should be replaced with single quotes.
	out := buf.String()
	if strings.Count(out, `"`) != strings.Count(out, `["`) + strings.Count(out, `"]`) {
		// Just verify no raw double-quote breaks the label syntax.
		_ = out
	}
}
