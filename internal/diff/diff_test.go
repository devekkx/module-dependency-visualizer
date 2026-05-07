package diff_test

import (
	"testing"

	"module-dependency-visualizer/internal/diff"
	"module-dependency-visualizer/internal/graph"
)

func buildGraph(t *testing.T, nodes []graph.Node) *graph.Graph {
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

func TestDiff_Added(t *testing.T) {
	from := buildGraph(t, []graph.Node{
		{ID: "app@", Name: "app", Kind: graph.NodeKindMain},
		{ID: "github.com/foo@v1.0.0", Name: "github.com/foo", Version: "v1.0.0", Kind: graph.NodeKindModule},
	})
	to := buildGraph(t, []graph.Node{
		{ID: "app@", Name: "app", Kind: graph.NodeKindMain},
		{ID: "github.com/foo@v1.0.0", Name: "github.com/foo", Version: "v1.0.0", Kind: graph.NodeKindModule},
		{ID: "github.com/bar@v2.0.0", Name: "github.com/bar", Version: "v2.0.0", Kind: graph.NodeKindModule},
	})

	r := diff.Diff(from, to)
	if len(r.Added) != 1 || r.Added[0].Name != "github.com/bar" {
		t.Errorf("Added = %v, want [{github.com/bar}]", r.Added)
	}
	if len(r.Removed) != 0 {
		t.Errorf("Removed = %v, want []", r.Removed)
	}
	if len(r.Updated) != 0 {
		t.Errorf("Updated = %v, want []", r.Updated)
	}
	if r.Unchanged != 1 {
		t.Errorf("Unchanged = %d, want 1", r.Unchanged)
	}
}

func TestDiff_Removed(t *testing.T) {
	from := buildGraph(t, []graph.Node{
		{ID: "app@", Name: "app", Kind: graph.NodeKindMain},
		{ID: "github.com/foo@v1.0.0", Name: "github.com/foo", Version: "v1.0.0", Kind: graph.NodeKindModule},
		{ID: "github.com/bar@v2.0.0", Name: "github.com/bar", Version: "v2.0.0", Kind: graph.NodeKindModule},
	})
	to := buildGraph(t, []graph.Node{
		{ID: "app@", Name: "app", Kind: graph.NodeKindMain},
		{ID: "github.com/foo@v1.0.0", Name: "github.com/foo", Version: "v1.0.0", Kind: graph.NodeKindModule},
	})

	r := diff.Diff(from, to)
	if len(r.Removed) != 1 || r.Removed[0].Name != "github.com/bar" {
		t.Errorf("Removed = %v, want [{github.com/bar}]", r.Removed)
	}
	if len(r.Added) != 0 {
		t.Errorf("Added = %v, want []", r.Added)
	}
}

func TestDiff_Updated(t *testing.T) {
	from := buildGraph(t, []graph.Node{
		{ID: "app@", Name: "app", Kind: graph.NodeKindMain},
		{ID: "github.com/foo@v1.0.0", Name: "github.com/foo", Version: "v1.0.0", Kind: graph.NodeKindModule},
	})
	to := buildGraph(t, []graph.Node{
		{ID: "app@", Name: "app", Kind: graph.NodeKindMain},
		{ID: "github.com/foo@v1.2.0", Name: "github.com/foo", Version: "v1.2.0", Kind: graph.NodeKindModule},
	})

	r := diff.Diff(from, to)
	if len(r.Updated) != 1 {
		t.Fatalf("Updated = %v, want 1 entry", r.Updated)
	}
	c := r.Updated[0]
	if c.Name != "github.com/foo" || c.FromVersion != "v1.0.0" || c.ToVersion != "v1.2.0" {
		t.Errorf("Updated[0] = %+v", c)
	}
}

func TestDiff_Unchanged(t *testing.T) {
	nodes := []graph.Node{
		{ID: "app@", Name: "app", Kind: graph.NodeKindMain},
		{ID: "github.com/foo@v1.0.0", Name: "github.com/foo", Version: "v1.0.0", Kind: graph.NodeKindModule},
		{ID: "github.com/bar@v2.0.0", Name: "github.com/bar", Version: "v2.0.0", Kind: graph.NodeKindModule},
	}
	from := buildGraph(t, nodes)
	to := buildGraph(t, nodes)

	r := diff.Diff(from, to)
	if len(r.Added) != 0 || len(r.Removed) != 0 || len(r.Updated) != 0 {
		t.Errorf("expected no changes, got added=%d removed=%d updated=%d",
			len(r.Added), len(r.Removed), len(r.Updated))
	}
	if r.Unchanged != 2 {
		t.Errorf("Unchanged = %d, want 2", r.Unchanged)
	}
}

func TestDiff_MainExcluded(t *testing.T) {
	from := buildGraph(t, []graph.Node{
		{ID: "app@v1.0.0", Name: "app", Version: "v1.0.0", Kind: graph.NodeKindMain},
	})
	to := buildGraph(t, []graph.Node{
		{ID: "app@v2.0.0", Name: "app", Version: "v2.0.0", Kind: graph.NodeKindMain},
	})

	r := diff.Diff(from, to)
	if len(r.Added)+len(r.Removed)+len(r.Updated) != 0 {
		t.Error("main module version change should not appear in diff")
	}
}

func TestDiff_SortedOutput(t *testing.T) {
	from := buildGraph(t, []graph.Node{
		{ID: "app@", Name: "app", Kind: graph.NodeKindMain},
	})
	to := buildGraph(t, []graph.Node{
		{ID: "app@", Name: "app", Kind: graph.NodeKindMain},
		{ID: "zzz.io/pkg@v1.0.0", Name: "zzz.io/pkg", Version: "v1.0.0", Kind: graph.NodeKindModule},
		{ID: "aaa.io/pkg@v1.0.0", Name: "aaa.io/pkg", Version: "v1.0.0", Kind: graph.NodeKindModule},
	})

	r := diff.Diff(from, to)
	if len(r.Added) != 2 {
		t.Fatalf("expected 2 added, got %d", len(r.Added))
	}
	if r.Added[0].Name != "aaa.io/pkg" {
		t.Errorf("first added = %q, want aaa.io/pkg", r.Added[0].Name)
	}
}
