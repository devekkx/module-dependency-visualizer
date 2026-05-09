package schema_test

import (
	"strings"
	"testing"
	"time"

	"github.com/devekkx/module-dependency-visualizer/internal/graph"
	"github.com/devekkx/module-dependency-visualizer/internal/schema"
)

func buildReplaceGraph(t *testing.T) *graph.Graph {
	t.Helper()
	b := graph.NewBuilder()

	main := graph.Node{
		ID:      "example.com/app@",
		Name:    "example.com/app",
		Kind:    graph.NodeKindMain,
	}
	replaced := graph.Node{
		ID:      "github.com/old/pkg@v1.0.0",
		Name:    "github.com/old/pkg",
		Version: "v1.0.0",
		Kind:    graph.NodeKindReplace,
		ReplacedBy: &graph.Replacement{
			Path:    "github.com/new/pkg",
			Version: "v2.0.0",
		},
	}

	for _, n := range []graph.Node{main, replaced} {
		if err := b.AddNode(n); err != nil {
			t.Fatal(err)
		}
	}
	_ = b.AddEdge(graph.Edge{
		From: "example.com/app@",
		To:   "github.com/old/pkg@v1.0.0",
		Kind: graph.EdgeKindDependsOn,
	})
	g, err := b.Build()
	if err != nil {
		t.Fatal(err)
	}
	return g
}

func TestEncode_ReplacedBy(t *testing.T) {
	g := buildReplaceGraph(t)
	opts := schema.EncodeOptions{
		Project:     schema.Project{Name: "example.com/app", Language: "go"},
		GeneratedAt: time.Date(2026, 5, 6, 12, 0, 0, 0, time.UTC),
	}

	data, err := schema.Encode(g, opts)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}

	if !strings.Contains(string(data), "replaced_by") {
		t.Error("Encode did not include replaced_by field")
	}
	if !strings.Contains(string(data), "github.com/new/pkg") {
		t.Error("Encode did not include replacement path")
	}
}

func TestDecode_ReplacedBy_RoundTrip(t *testing.T) {
	g := buildReplaceGraph(t)
	opts := schema.EncodeOptions{
		Project:     schema.Project{Name: "example.com/app", Language: "go"},
		GeneratedAt: time.Date(2026, 5, 6, 12, 0, 0, 0, time.UTC),
	}

	data, err := schema.Encode(g, opts)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}

	decoded, _, err := schema.Decode(data)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}

	n, ok := decoded.Node("github.com/old/pkg@v1.0.0")
	if !ok {
		t.Fatal("decoded graph missing replaced node")
	}
	if n.ReplacedBy == nil {
		t.Fatal("decoded node missing ReplacedBy")
	}
	if n.ReplacedBy.Path != "github.com/new/pkg" {
		t.Errorf("ReplacedBy.Path = %q; want %q", n.ReplacedBy.Path, "github.com/new/pkg")
	}
}

func TestEncode_DefaultGeneratedAt(t *testing.T) {
	b := graph.NewBuilder()
	g, _ := b.Build()

	// GeneratedAt zero → should be filled with time.Now()
	data, err := schema.Encode(g, schema.EncodeOptions{})
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	if !strings.Contains(string(data), "generated_at") {
		t.Error("Encode with zero GeneratedAt should still emit generated_at")
	}
}

func TestDecode_MalformedNodes(t *testing.T) {
	// Duplicate node IDs should fail.
	data := []byte(`{
		"schema_version":"1.1.0",
		"project":{},
		"nodes":[
			{"id":"a@v1","name":"a","version":"v1","kind":"module"},
			{"id":"a@v1","name":"a","version":"v1","kind":"module"}
		],
		"edges":[],
		"stats":{}
	}`)
	_, _, err := schema.Decode(data)
	if err == nil {
		t.Error("expected error for duplicate node IDs, got nil")
	}
}
