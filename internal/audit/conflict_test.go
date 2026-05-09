package audit_test

import (
	"testing"

	"github.com/devekkx/module-dependency-visualizer/internal/audit"
	"github.com/devekkx/module-dependency-visualizer/internal/graph"
)

func buildConflictGraph(t *testing.T, nodes []graph.Node) *graph.Graph {
	t.Helper()
	b := graph.NewBuilder()
	for _, n := range nodes {
		if err := b.AddNode(n); err != nil {
			t.Fatalf("AddNode %s: %v", n.ID, err)
		}
	}
	g, err := b.Build()
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	return g
}

func TestDetectConflicts_NoConflicts(t *testing.T) {
	g := buildConflictGraph(t, []graph.Node{
		{ID: "example.com/app@", Name: "app", Kind: graph.NodeKindMain},
		{ID: "github.com/foo@v1.0.0", Name: "github.com/foo", Version: "v1.0.0", Kind: graph.NodeKindModule},
		{ID: "github.com/bar@v2.0.0", Name: "github.com/bar", Version: "v2.0.0", Kind: graph.NodeKindModule},
	})

	conflicts := audit.DetectConflicts(g)
	if len(conflicts) != 0 {
		t.Fatalf("expected no conflicts, got %d", len(conflicts))
	}
}

func TestDetectConflicts_WithConflict(t *testing.T) {
	g := buildConflictGraph(t, []graph.Node{
		{ID: "example.com/app@", Name: "app", Kind: graph.NodeKindMain},
		{ID: "github.com/foo@v1.0.0", Name: "github.com/foo", Version: "v1.0.0", Kind: graph.NodeKindModule},
		{ID: "github.com/foo@v1.2.0", Name: "github.com/foo", Version: "v1.2.0", Kind: graph.NodeKindModule},
		{ID: "github.com/bar@v2.0.0", Name: "github.com/bar", Version: "v2.0.0", Kind: graph.NodeKindModule},
	})

	conflicts := audit.DetectConflicts(g)
	if len(conflicts) != 1 {
		t.Fatalf("expected 1 conflict, got %d", len(conflicts))
	}
	c := conflicts[0]
	if c.Module != "github.com/foo" {
		t.Errorf("module = %q, want %q", c.Module, "github.com/foo")
	}
	if len(c.Versions) != 2 || c.Versions[0] != "v1.0.0" || c.Versions[1] != "v1.2.0" {
		t.Errorf("versions = %v, want [v1.0.0 v1.2.0]", c.Versions)
	}
}

func TestDetectConflicts_MainExcluded(t *testing.T) {
	g := buildConflictGraph(t, []graph.Node{
		{ID: "example.com/app@", Name: "app", Kind: graph.NodeKindMain},
	})

	conflicts := audit.DetectConflicts(g)
	if len(conflicts) != 0 {
		t.Fatalf("main module should not trigger conflicts, got %d", len(conflicts))
	}
}

func TestDetectConflicts_SortedOutput(t *testing.T) {
	g := buildConflictGraph(t, []graph.Node{
		{ID: "example.com/app@", Name: "app", Kind: graph.NodeKindMain},
		{ID: "zzz.io/pkg@v1.0.0", Name: "zzz.io/pkg", Version: "v1.0.0", Kind: graph.NodeKindModule},
		{ID: "zzz.io/pkg@v2.0.0", Name: "zzz.io/pkg", Version: "v2.0.0", Kind: graph.NodeKindModule},
		{ID: "aaa.io/pkg@v1.0.0", Name: "aaa.io/pkg", Version: "v1.0.0", Kind: graph.NodeKindModule},
		{ID: "aaa.io/pkg@v3.0.0", Name: "aaa.io/pkg", Version: "v3.0.0", Kind: graph.NodeKindModule},
	})

	conflicts := audit.DetectConflicts(g)
	if len(conflicts) != 2 {
		t.Fatalf("expected 2 conflicts, got %d", len(conflicts))
	}
	if conflicts[0].Module != "aaa.io/pkg" {
		t.Errorf("first conflict = %q, want %q", conflicts[0].Module, "aaa.io/pkg")
	}
	if conflicts[1].Module != "zzz.io/pkg" {
		t.Errorf("second conflict = %q, want %q", conflicts[1].Module, "zzz.io/pkg")
	}
}
