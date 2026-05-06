package jsonexp_test

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"
	"time"

	"module-dependency-visualizer/internal/exporter/jsonexp"
	"module-dependency-visualizer/internal/graph"
	"module-dependency-visualizer/internal/schema"
)

func buildTestGraph(t *testing.T) *graph.Graph {
	t.Helper()
	b := graph.NewBuilder()
	n := graph.Node{ID: "a@v1", Name: "a", Version: "v1", Kind: graph.NodeKindModule}
	if err := b.AddNode(n); err != nil {
		t.Fatal(err)
	}
	g, err := b.Build()
	if err != nil {
		t.Fatal(err)
	}
	return g
}

func TestJSONExporter_Format(t *testing.T) {
	e := jsonexp.New(schema.EncodeOptions{})
	if e.Format() != "json" {
		t.Errorf("Format() = %q; want 'json'", e.Format())
	}
}

func TestJSONExporter_WritesValidJSON(t *testing.T) {
	g := buildTestGraph(t)
	e := jsonexp.New(schema.EncodeOptions{
		Project:     schema.Project{Name: "test", Language: "go"},
		GeneratedAt: time.Date(2026, 5, 6, 12, 0, 0, 0, time.UTC),
	})

	var buf bytes.Buffer
	if err := e.Write(context.Background(), &buf, g); err != nil {
		t.Fatalf("Write: %v", err)
	}

	var out map[string]any
	if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
		t.Fatalf("JSON exporter produced invalid JSON: %v", err)
	}
	if out["schema_version"] != schema.Version {
		t.Errorf("schema_version = %v; want %q", out["schema_version"], schema.Version)
	}
}
