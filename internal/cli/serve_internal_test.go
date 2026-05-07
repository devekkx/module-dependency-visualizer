package cli

import (
	"context"
	"errors"
	"testing"

	"module-dependency-visualizer/internal/config"
	"module-dependency-visualizer/internal/exporter"
	"module-dependency-visualizer/internal/graph"
	"module-dependency-visualizer/internal/logging"
	"module-dependency-visualizer/internal/provider"
)

// stubProviderInternal is a test-only Provider that always detects and returns
// a fixed graph. Defined here in the internal test to avoid package cycles.
type stubProviderInternal struct {
	g    *graph.Graph
	proj provider.Project
}

func (s *stubProviderInternal) Name() string { return "stub-internal" }
func (s *stubProviderInternal) Detect(_ context.Context, _ string) (bool, error) {
	return true, nil
}
func (s *stubProviderInternal) Parse(_ context.Context, _ string, _ provider.ParseOptions) (*graph.Graph, provider.Project, error) {
	return s.g, s.proj, nil
}

func buildStubGraphInternal(t *testing.T) *graph.Graph {
	t.Helper()
	b := graph.NewBuilder()
	n := graph.Node{ID: "root@v1", Name: "root", Version: "v1", Kind: graph.NodeKindMain}
	if err := b.AddNode(n); err != nil {
		t.Fatal(err)
	}
	g, err := b.Build()
	if err != nil {
		t.Fatal(err)
	}
	return g
}

func TestClassifyError_Nil(t *testing.T) {
	if got := classifyError(nil); got != exitSuccess {
		t.Errorf("classifyError(nil) = %d, want %d", got, exitSuccess)
	}
}

func TestClassifyError_NonNil(t *testing.T) {
	err := errors.New("some error")
	if got := classifyError(err); got != exitUser {
		t.Errorf("classifyError(err) = %d, want %d", got, exitUser)
	}
}

func TestOpenBrowser_NoPanic(t *testing.T) {
	// openBrowser is best-effort; must not panic in any environment.
	openBrowser("http://localhost:7070")
}

func TestBuildDeps_ReturnsNonNil(t *testing.T) {
	deps := buildDeps()
	if deps == nil {
		t.Fatal("buildDeps() returned nil")
	}
	if deps.Providers == nil {
		t.Error("buildDeps().Providers is nil")
	}
	if deps.Exporters == nil {
		t.Error("buildDeps().Exporters is nil")
	}
}

func TestExecute_DoesNotPanic(t *testing.T) {
	// Execute uses os.Args which in the test binary are test flags cobra cannot
	// parse, so it will return a non-zero exit code — but it must not panic.
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Execute panicked: %v", r)
		}
	}()
	_ = Execute()
}

func TestNewServeCmd_HappyPath(t *testing.T) {
	// Override waitForSignal so the serve command returns immediately after
	// starting, without blocking on an OS signal.
	orig := waitForSignal
	waitForSignal = func(_ context.Context) {}
	defer func() { waitForSignal = orig }()

	g := buildStubGraphInternal(t)
	proj := provider.Project{Name: "test", Language: "go", RootPath: "/test", MainModule: "test"}

	provReg := provider.NewRegistry()
	if err := provReg.Register(&stubProviderInternal{g: g, proj: proj}); err != nil {
		t.Fatal(err)
	}

	deps := &Deps{
		Log:       logging.Discard(),
		BuildInfo: config.Default,
		Providers: provReg,
		Exporters: exporter.NewRegistry(),
	}

	cmd := newServeCmd(deps)
	cmd.SetArgs([]string{"--no-browser", "."})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("serve happy path: %v", err)
	}
}
