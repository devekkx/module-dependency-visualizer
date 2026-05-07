package docgen_test

import (
	"strings"
	"testing"
	"time"

	"module-dependency-visualizer/internal/docgen"
	"module-dependency-visualizer/internal/graph"
	"module-dependency-visualizer/internal/provider"
	"module-dependency-visualizer/internal/schema"
)

func buildTestGraph(t *testing.T) *graph.Graph {
	t.Helper()
	b := graph.NewBuilder()
	nodes := []graph.Node{
		{ID: "example.com/app@", Name: "example.com/app", Kind: graph.NodeKindMain},
		{ID: "github.com/foo@v1.0.0", Name: "github.com/foo", Version: "v1.0.0", Kind: graph.NodeKindModule},
		{ID: "github.com/bar@v2.0.0", Name: "github.com/bar", Version: "v2.0.0", Kind: graph.NodeKindModule, Indirect: true},
	}
	for _, n := range nodes {
		if err := b.AddNode(n); err != nil {
			t.Fatalf("AddNode: %v", err)
		}
	}
	g, err := b.Build()
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	return g
}

func generate(t *testing.T, g *graph.Graph, proj provider.Project, opts docgen.Options) string {
	t.Helper()
	var sb strings.Builder
	if err := docgen.Generate(&sb, g, proj, opts); err != nil {
		t.Fatalf("Generate: %v", err)
	}
	return sb.String()
}

func TestGenerate_ContainsTitle(t *testing.T) {
	g := buildTestGraph(t)
	proj := provider.Project{Name: "my-app", Language: "go", MainModule: "example.com/app"}
	out := generate(t, g, proj, docgen.Options{GeneratedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)})

	if !strings.Contains(out, "my-app") {
		t.Errorf("output missing project name; got:\n%s", out)
	}
	if !strings.Contains(out, "2026-01-01") {
		t.Errorf("output missing date; got:\n%s", out)
	}
}

func TestGenerate_DirectAndIndirect(t *testing.T) {
	g := buildTestGraph(t)
	proj := provider.Project{Name: "app"}
	out := generate(t, g, proj, docgen.Options{GeneratedAt: time.Now()})

	if !strings.Contains(out, "github.com/foo") {
		t.Errorf("output missing direct dep")
	}
	if !strings.Contains(out, "github.com/bar") {
		t.Errorf("output missing indirect dep")
	}
	if !strings.Contains(out, "## Direct Dependencies") {
		t.Error("missing direct section")
	}
	if !strings.Contains(out, "## Indirect Dependencies") {
		t.Error("missing indirect section")
	}
}

func TestGenerate_WithAudit(t *testing.T) {
	g := buildTestGraph(t)
	proj := provider.Project{Name: "app"}

	audit := &schema.AuditDTO{
		Vulnerabilities: []schema.VulnDTO{
			{NodeID: "github.com/foo@v1.0.0", ID: "GHSA-test", Severity: "HIGH", Summary: "A test vuln", Link: "https://osv.dev/GHSA-test"},
		},
		Licenses: map[string]string{
			"github.com/foo@v1.0.0": "MIT",
			"github.com/bar@v2.0.0": "Apache-2.0",
		},
	}

	out := generate(t, g, proj, docgen.Options{Audit: audit, GeneratedAt: time.Now()})

	if !strings.Contains(out, "GHSA-test") {
		t.Error("output missing vulnerability ID")
	}
	if !strings.Contains(out, "MIT") {
		t.Error("output missing license")
	}
	if !strings.Contains(out, "## Security") {
		t.Error("output missing Security section")
	}
}

func TestGenerate_NoVulnerabilities(t *testing.T) {
	g := buildTestGraph(t)
	proj := provider.Project{Name: "app"}
	audit := &schema.AuditDTO{
		Vulnerabilities: nil,
		Licenses:        map[string]string{},
	}

	out := generate(t, g, proj, docgen.Options{Audit: audit, GeneratedAt: time.Now()})
	if !strings.Contains(out, "No known vulnerabilities") {
		t.Error("output should report no vulnerabilities")
	}
}

func TestGenerate_FooterPresent(t *testing.T) {
	g := buildTestGraph(t)
	out := generate(t, g, provider.Project{}, docgen.Options{GeneratedAt: time.Now()})
	if !strings.Contains(out, "mdv") {
		t.Error("output missing footer")
	}
}
