package mermaid_test

import (
	"bytes"
	"context"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"module-dependency-visualizer/internal/exporter/mermaid"
	"module-dependency-visualizer/internal/graph"
)

var update = flag.Bool("update", false, "Update golden files")

func buildTestGraph(t *testing.T) *graph.Graph {
	t.Helper()
	b := graph.NewBuilder()
	nodes := []graph.Node{
		{ID: "example.com/app@", Name: "example.com/app", Kind: graph.NodeKindMain},
		{ID: "github.com/spf13/cobra@v1.8.0", Name: "github.com/spf13/cobra", Version: "v1.8.0", Kind: graph.NodeKindModule},
		{ID: "github.com/spf13/pflag@v1.0.9", Name: "github.com/spf13/pflag", Version: "v1.0.9", Kind: graph.NodeKindModule, Indirect: true},
	}
	for _, n := range nodes {
		if err := b.AddNode(n); err != nil {
			t.Fatal(err)
		}
	}
	for _, e := range []graph.Edge{
		{From: "example.com/app@", To: "github.com/spf13/cobra@v1.8.0", Kind: graph.EdgeKindDependsOn},
		{From: "github.com/spf13/cobra@v1.8.0", To: "github.com/spf13/pflag@v1.0.9", Kind: graph.EdgeKindDependsOn},
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

func TestMermaidExporter_Golden(t *testing.T) {
	g := buildTestGraph(t)
	e := mermaid.New()

	var buf bytes.Buffer
	if err := e.Write(context.Background(), &buf, g); err != nil {
		t.Fatalf("Write: %v", err)
	}

	goldenPath := filepath.Join("..", "..", "..", "testdata", "fixtures", "golden", "simple.mmd")

	if *update {
		if err := os.MkdirAll(filepath.Dir(goldenPath), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(goldenPath, buf.Bytes(), 0644); err != nil {
			t.Fatal(err)
		}
		t.Logf("Updated golden file: %s", goldenPath)
		return
	}

	want, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("Reading golden file: %v\nRun with -update to create it.", err)
	}

	if buf.String() != string(want) {
		t.Errorf("Mermaid output does not match golden.\nGot:\n%s\nWant:\n%s", buf.String(), want)
	}
}

func TestMermaidExporter_Deterministic(t *testing.T) {
	g := buildTestGraph(t)
	e := mermaid.New()

	var first, second bytes.Buffer
	_ = e.Write(context.Background(), &first, g)
	_ = e.Write(context.Background(), &second, g)

	if first.String() != second.String() {
		t.Error("Write() is not deterministic")
	}
}

func TestMermaidExporter_Format(t *testing.T) {
	if mermaid.New().Format() != "mermaid" {
		t.Error("Format() should return 'mermaid'")
	}
}

func TestMermaidExporter_ValidMermaidSyntax(t *testing.T) {
	g := buildTestGraph(t)
	e := mermaid.New()

	var buf bytes.Buffer
	if err := e.Write(context.Background(), &buf, g); err != nil {
		t.Fatal(err)
	}

	out := buf.String()
	if !strings.HasPrefix(out, "graph TD") {
		t.Error("Mermaid output must start with 'graph TD'")
	}
	if !strings.Contains(out, "-->") {
		t.Error("Mermaid output must contain '-->' arrows")
	}
	// Node IDs should use aliases, not raw module paths.
	if strings.Contains(out, "github.com/spf13") {
		for _, line := range strings.Split(out, "\n") {
			line = strings.TrimSpace(line)
			// Aliases like n0, n1 should appear as IDs; module paths only inside labels ["..."]
			if strings.HasPrefix(line, "github.com") {
				t.Errorf("found raw module path as node ID in: %q", line)
			}
		}
	}
}
