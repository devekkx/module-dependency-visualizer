package jsonexp_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/devekkx/module-dependency-visualizer/internal/exporter/jsonexp"
	"github.com/devekkx/module-dependency-visualizer/internal/graph"
	"github.com/devekkx/module-dependency-visualizer/internal/schema"
)

type errorWriter struct{}

func (errorWriter) Write(_ []byte) (int, error) {
	return 0, errors.New("write error")
}

func TestJSONExporter_WriteError(t *testing.T) {
	g := buildTestGraph(t)
	e := jsonexp.New(schema.EncodeOptions{})

	err := e.Write(context.Background(), errorWriter{}, g)
	if err == nil {
		t.Error("expected error when writer fails, got nil")
	}
}

func TestJSONExporter_EmptyGraph(t *testing.T) {
	b := graph.NewBuilder()
	g, _ := b.Build()
	e := jsonexp.New(schema.EncodeOptions{
		Project:     schema.Project{Name: "empty", Language: "go"},
		GeneratedAt: time.Date(2026, 5, 6, 0, 0, 0, 0, time.UTC),
	})

	var buf bytes.Buffer
	if err := e.Write(context.Background(), &buf, g); err != nil {
		t.Fatalf("Write: %v", err)
	}

	var out map[string]any
	if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	nodes := out["nodes"].([]any)
	if len(nodes) != 0 {
		t.Errorf("expected 0 nodes, got %d", len(nodes))
	}
}

func TestJSONExporter_EncodeError(t *testing.T) {
	// Put an un-serializable value in Metadata to trigger schema.Encode failure.
	b := graph.NewBuilder()
	_ = b.AddNode(graph.Node{
		ID:       "a@v1",
		Name:     "a",
		Kind:     graph.NodeKindMain,
		Metadata: map[string]any{"bad": make(chan int)},
	})
	g, _ := b.Build()
	e := jsonexp.New(schema.EncodeOptions{})

	var buf bytes.Buffer
	err := e.Write(context.Background(), &buf, g)
	if err == nil {
		t.Error("expected error when Encode fails, got nil")
	}
}

func TestJSONExporter_ContextPassthrough(t *testing.T) {
	g := buildTestGraph(t)
	e := jsonexp.New(schema.EncodeOptions{})

	// Context is accepted but not used by jsonexp. Just verify no panic.
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // already cancelled

	var buf bytes.Buffer
	// Should still succeed because JSON encoding is synchronous/local.
	if err := e.Write(ctx, &buf, g); err != nil {
		t.Logf("Write with cancelled context returned error (acceptable): %v", err)
	}
}
