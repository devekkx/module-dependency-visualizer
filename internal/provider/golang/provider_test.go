package golang_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/devekkx/module-dependency-visualizer/internal/provider"
	goprovider "github.com/devekkx/module-dependency-visualizer/internal/provider/golang"
)

// fakeRunner returns preconfigured output for specific commands.
type fakeRunner struct {
	responses map[string][]byte
	err       error
}

func (f *fakeRunner) Run(_ context.Context, _ string, _ string, args ...string) ([]byte, error) {
	if f.err != nil {
		return nil, f.err
	}
	key := args[len(args)-1] // last arg differentiates "graph" vs "all"
	if data, ok := f.responses[key]; ok {
		return data, nil
	}
	return nil, errors.New("fakeRunner: no response for " + key)
}

func newFakeRunnerFromFixtures(t *testing.T) *fakeRunner {
	t.Helper()
	return &fakeRunner{
		responses: map[string][]byte{
			"graph": readFixture(t, "modgraph.txt"),
			"all":   readFixture(t, "modlist.json"),
		},
	}
}

func TestGoProvider_Name(t *testing.T) {
	p := goprovider.New()
	if p.Name() != "go" {
		t.Errorf("Name() = %q; want %q", p.Name(), "go")
	}
}

func TestGoProvider_Detect_GoModPresent(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module x"), 0644); err != nil {
		t.Fatal(err)
	}

	p := goprovider.New()
	ok, err := p.Detect(context.Background(), dir)
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	if !ok {
		t.Error("Detect() = false; want true when go.mod is present")
	}
}

func TestGoProvider_Detect_NoGoMod(t *testing.T) {
	p := goprovider.New()
	ok, err := p.Detect(context.Background(), t.TempDir())
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	if ok {
		t.Error("Detect() = true; want false when go.mod is absent")
	}
}

func TestGoProvider_Parse_BuildsGraph(t *testing.T) {
	runner := newFakeRunnerFromFixtures(t)
	p := goprovider.NewWithRunner(runner)

	g, proj, err := p.Parse(context.Background(), "/fake/path", provider.ParseOptions{MaxDepth: -1})
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	if proj.Language != "go" {
		t.Errorf("Project.Language = %q; want %q", proj.Language, "go")
	}
	if proj.MainModule != "example.com/app" {
		t.Errorf("Project.MainModule = %q; want %q", proj.MainModule, "example.com/app")
	}

	if g.NodeCount() != 3 {
		t.Errorf("NodeCount = %d; want 3", g.NodeCount())
	}
	if g.EdgeCount() != 2 {
		t.Errorf("EdgeCount = %d; want 2", g.EdgeCount())
	}
}

func TestGoProvider_Parse_RunnerError(t *testing.T) {
	runner := &fakeRunner{err: errors.New("command not found")}
	p := goprovider.NewWithRunner(runner)

	_, _, err := p.Parse(context.Background(), "/fake/path", provider.ParseOptions{})
	if err == nil {
		t.Error("expected error when runner fails, got nil")
	}
}

func TestGoProvider_Detect_NonExistentDir(t *testing.T) {
	p := goprovider.New()
	ok, err := p.Detect(context.Background(), "/nonexistent/path/that/does/not/exist")
	if err != nil {
		t.Fatalf("Detect on non-existent dir: %v", err)
	}
	if ok {
		t.Error("Detect() = true; want false for non-existent directory")
	}
}

func TestGoProvider_Parse_PartialRunnerError(t *testing.T) {
	// Runner succeeds for "graph" but fails for "all" (go list).
	runner := &fakeRunner{
		responses: map[string][]byte{
			"graph": readFixture(t, "modgraph.txt"),
		},
	}
	p := goprovider.NewWithRunner(runner)

	_, _, err := p.Parse(context.Background(), "/fake/path", provider.ParseOptions{})
	if err == nil {
		t.Error("expected error when go list fails, got nil")
	}
}

func TestGoProvider_Parse_EmptyModGraph(t *testing.T) {
	runner := &fakeRunner{
		responses: map[string][]byte{
			"graph": []byte(""),
			"all":   readFixture(t, "modlist.json"),
		},
	}
	p := goprovider.NewWithRunner(runner)

	// Empty mod graph means no edges - the graph should have zero nodes since
	// all nodes come from edges in go mod graph output.
	g, _, err := p.Parse(context.Background(), "/fake/path", provider.ParseOptions{})
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if g.NodeCount() != 0 {
		t.Errorf("NodeCount = %d; want 0 for empty modgraph", g.NodeCount())
	}
}

func TestGoProvider_Parse_IndirectFlag(t *testing.T) {
	runner := newFakeRunnerFromFixtures(t)
	p := goprovider.NewWithRunner(runner)

	g, _, err := p.Parse(context.Background(), "/fake/path", provider.ParseOptions{MaxDepth: -1})
	if err != nil {
		t.Fatal(err)
	}

	for _, n := range g.Nodes() {
		if n.Name == "github.com/spf13/pflag" && !n.Indirect {
			t.Error("pflag should be marked as indirect")
		}
		if n.Name == "github.com/spf13/cobra" && n.Indirect {
			t.Error("cobra should not be marked as indirect")
		}
	}
}
