package graph_test

import (
	"testing"

	"module-dependency-visualizer/internal/graph"
)

func TestNewNodeID(t *testing.T) {
	cases := []struct {
		name    string
		path    string
		version string
		want    graph.NodeID
	}{
		{
			name:    "module with version",
			path:    "github.com/spf13/cobra",
			version: "v1.8.0",
			want:    "github.com/spf13/cobra@v1.8.0",
		},
		{
			name:    "main module empty version",
			path:    "example.com/myapp",
			version: "",
			want:    "example.com/myapp@",
		},
		{
			name:    "pseudo-version",
			path:    "golang.org/x/tools",
			version: "v0.0.0-20240101000000-abcdef123456",
			want:    "golang.org/x/tools@v0.0.0-20240101000000-abcdef123456",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := graph.NewNodeID(tc.path, tc.version)
			if got != tc.want {
				t.Errorf("NewNodeID(%q, %q) = %q; want %q", tc.path, tc.version, got, tc.want)
			}
		})
	}
}

func TestNode_Copy_IsolatesMetadata(t *testing.T) {
	original := graph.Node{
		ID:       graph.NewNodeID("example.com/foo", "v1.0.0"),
		Name:     "example.com/foo",
		Version:  "v1.0.0",
		Kind:     graph.NodeKindModule,
		Metadata: map[string]any{"key": "value"},
	}

	copied := original.Copy()
	copied.Metadata["key"] = "mutated"

	if original.Metadata["key"] != "value" {
		t.Error("Copy() did not isolate metadata — original was mutated")
	}
}

func TestNode_Copy_NilMetadata(t *testing.T) {
	n := graph.Node{ID: "x@v1", Metadata: nil}
	copied := n.Copy()
	if copied.Metadata != nil {
		t.Error("Copy() with nil Metadata should remain nil")
	}
}

func TestEdge_Copy_IsolatesMetadata(t *testing.T) {
	e := graph.Edge{
		From:     "a@v1",
		To:       "b@v2",
		Kind:     graph.EdgeKindDependsOn,
		Metadata: map[string]any{"direct": true},
	}
	copied := e.Copy()
	copied.Metadata["direct"] = false

	if e.Metadata["direct"] != true {
		t.Error("Edge.Copy() did not isolate metadata")
	}
}
