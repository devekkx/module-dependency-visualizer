package cli_test

import (
	"bytes"
	"context"
	"os"
	"strings"
	"testing"

	"github.com/devekkx/module-dependency-visualizer/internal/cli"
	"github.com/devekkx/module-dependency-visualizer/internal/config"
	"github.com/devekkx/module-dependency-visualizer/internal/exporter"
	"github.com/devekkx/module-dependency-visualizer/internal/exporter/dot"
	"github.com/devekkx/module-dependency-visualizer/internal/exporter/jsonexp"
	"github.com/devekkx/module-dependency-visualizer/internal/exporter/mermaid"
	"github.com/devekkx/module-dependency-visualizer/internal/graph"
	"github.com/devekkx/module-dependency-visualizer/internal/logging"
	"github.com/devekkx/module-dependency-visualizer/internal/provider"
	"github.com/devekkx/module-dependency-visualizer/internal/schema"
)

// stubProvider always detects and returns a fixed graph.
type stubProvider struct {
	name string
	g    *graph.Graph
	proj provider.Project
	err  error
}

func (s *stubProvider) Name() string { return s.name }

func (s *stubProvider) Detect(_ context.Context, _ string) (bool, error) {
	return s.err == nil, s.err
}

func (s *stubProvider) Parse(_ context.Context, _ string, _ provider.ParseOptions) (*graph.Graph, provider.Project, error) {
	return s.g, s.proj, s.err
}

func buildStubGraph(t *testing.T) *graph.Graph {
	t.Helper()
	b := graph.NewBuilder()
	n := graph.Node{ID: "test@v1", Name: "test", Version: "v1", Kind: graph.NodeKindMain}
	if err := b.AddNode(n); err != nil {
		t.Fatal(err)
	}
	g, err := b.Build()
	if err != nil {
		t.Fatal(err)
	}
	return g
}

func buildTestDeps(t *testing.T) *cli.Deps {
	t.Helper()
	g := buildStubGraph(t)
	proj := provider.Project{Name: "test", Language: "go", RootPath: "/test", MainModule: "test"}

	provReg := provider.NewRegistry()
	if err := provReg.Register(&stubProvider{name: "go", g: g, proj: proj}); err != nil {
		t.Fatal(err)
	}

	expReg := exporter.NewRegistry()
	_ = expReg.Register(jsonexp.New(schema.EncodeOptions{}))
	_ = expReg.Register(dot.New())
	_ = expReg.Register(mermaid.New())

	return &cli.Deps{
		Log:       logging.Discard(),
		BuildInfo: config.Default,
		Providers: provReg,
		Exporters: expReg,
	}
}

func TestVersion_PrintsBuildInfo(t *testing.T) {
	deps := buildTestDeps(t)
	deps.BuildInfo = config.BuildInfo{Version: "v1.2.3", Commit: "abc", BuildDate: "2026-05-06"}

	code := cli.ExecuteWithDeps(deps, []string{"version"})
	if code != 0 {
		t.Errorf("exit code = %d; want 0", code)
	}
}

func TestAnalyze_WritesJSON(t *testing.T) {
	deps := buildTestDeps(t)

	tmpFile, err := os.CreateTemp(t.TempDir(), "out*.json")
	if err != nil {
		t.Fatal(err)
	}
	tmpFile.Close()

	code := cli.ExecuteWithDeps(deps, []string{"analyze", ".", "--output", tmpFile.Name()})
	if code != 0 {
		t.Errorf("exit code = %d; want 0", code)
	}

	data, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "schema_version") {
		t.Error("analyze output missing schema_version")
	}
}

func TestExport_DOT(t *testing.T) {
	deps := buildTestDeps(t)

	tmpFile, err := os.CreateTemp(t.TempDir(), "out*.dot")
	if err != nil {
		t.Fatal(err)
	}
	tmpFile.Close()

	code := cli.ExecuteWithDeps(deps, []string{"export", ".", "--format", "dot", "--output", tmpFile.Name()})
	if code != 0 {
		t.Errorf("exit code = %d; want 0", code)
	}

	data, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "digraph deps") {
		t.Error("export dot output missing 'digraph deps'")
	}
}

func TestExport_UnknownFormat_Fails(t *testing.T) {
	deps := buildTestDeps(t)
	code := cli.ExecuteWithDeps(deps, []string{"export", ".", "--format", "csv"})
	if code == 0 {
		t.Error("expected non-zero exit for unknown format, got 0")
	}
}

func TestVersion_OutputContainsVersion(t *testing.T) {
	deps := buildTestDeps(t)
	deps.BuildInfo = config.BuildInfo{Version: "v9.9.9", Commit: "xyz", BuildDate: "2026-01-01"}

	var buf bytes.Buffer
	code := cli.ExecuteWithDepsOutput(deps, []string{"version"}, &buf)
	if code != 0 {
		t.Errorf("exit code = %d; want 0", code)
	}
	if !strings.Contains(buf.String(), "v9.9.9") {
		t.Errorf("version output %q does not contain v9.9.9", buf.String())
	}
}

func TestUnknownCommand_Fails(t *testing.T) {
	deps := buildTestDeps(t)
	code := cli.ExecuteWithDeps(deps, []string{"unknowncmd"})
	if code == 0 {
		t.Error("expected non-zero exit for unknown command, got 0")
	}
}
