package graph_test

import (
	"regexp"
	"testing"

	"module-dependency-visualizer/internal/graph"
)

func buildFilterFixture(t *testing.T) *graph.Graph {
	t.Helper()
	// root -> cobra -> pflag
	//      -> indirect (indirect=true)
	b := graph.NewBuilder()

	root := graph.Node{ID: "app@", Name: "app", Kind: graph.NodeKindMain}
	cobra := graph.Node{ID: "cobra@v1.8.0", Name: "github.com/spf13/cobra", Version: "v1.8.0", Kind: graph.NodeKindModule}
	pflag := graph.Node{ID: "pflag@v1.0.9", Name: "github.com/spf13/pflag", Version: "v1.0.9", Kind: graph.NodeKindModule}
	indirect := graph.Node{ID: "indirect@v1.0.0", Name: "github.com/some/indirect", Version: "v1.0.0", Kind: graph.NodeKindModule, Indirect: true}

	for _, n := range []graph.Node{root, cobra, pflag, indirect} {
		if err := b.AddNode(n); err != nil {
			t.Fatal(err)
		}
	}
	for _, e := range []graph.Edge{
		{From: "app@", To: "cobra@v1.8.0", Kind: graph.EdgeKindDependsOn},
		{From: "cobra@v1.8.0", To: "pflag@v1.0.9", Kind: graph.EdgeKindDependsOn},
		{From: "app@", To: "indirect@v1.0.0", Kind: graph.EdgeKindDependsOn},
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

func nodeNames(g *graph.Graph) map[string]bool {
	names := make(map[string]bool)
	for _, n := range g.Nodes() {
		names[n.Name] = true
	}
	return names
}

func TestFilter_MaxDepth(t *testing.T) {
	g := buildFilterFixture(t)

	filtered, err := graph.Filter(g, graph.FilterOptions{MaxDepth: 1})
	if err != nil {
		t.Fatalf("Filter: %v", err)
	}

	names := nodeNames(filtered)
	if names["github.com/spf13/pflag"] {
		t.Error("depth=1 filter should exclude pflag (depth 2)")
	}
	if !names["github.com/spf13/cobra"] {
		t.Error("depth=1 filter should include cobra (depth 1)")
	}
}

func TestFilter_NoIndirect(t *testing.T) {
	g := buildFilterFixture(t)

	filtered, err := graph.Filter(g, graph.FilterOptions{MaxDepth: -1, NoIndirect: true})
	if err != nil {
		t.Fatalf("Filter: %v", err)
	}

	names := nodeNames(filtered)
	if names["github.com/some/indirect"] {
		t.Error("NoIndirect filter should exclude indirect dependency")
	}
	if !names["github.com/spf13/cobra"] {
		t.Error("NoIndirect filter should keep direct dependency cobra")
	}
}

func TestFilter_IncludePattern(t *testing.T) {
	g := buildFilterFixture(t)

	filtered, err := graph.Filter(g, graph.FilterOptions{
		MaxDepth:        -1,
		IncludePatterns: []*regexp.Regexp{regexp.MustCompile(`spf13`)},
	})
	if err != nil {
		t.Fatalf("Filter: %v", err)
	}

	names := nodeNames(filtered)
	if names["github.com/some/indirect"] {
		t.Error("include spf13 pattern should exclude indirect module")
	}
	if !names["github.com/spf13/cobra"] {
		t.Error("include spf13 pattern should keep cobra")
	}
}

func TestFilter_ExcludePattern(t *testing.T) {
	g := buildFilterFixture(t)

	filtered, err := graph.Filter(g, graph.FilterOptions{
		MaxDepth:        -1,
		ExcludePatterns: []*regexp.Regexp{regexp.MustCompile(`pflag`)},
	})
	if err != nil {
		t.Fatalf("Filter: %v", err)
	}

	names := nodeNames(filtered)
	if names["github.com/spf13/pflag"] {
		t.Error("exclude pflag pattern should remove pflag")
	}
	if !names["github.com/spf13/cobra"] {
		t.Error("exclude pflag pattern should keep cobra")
	}
}
