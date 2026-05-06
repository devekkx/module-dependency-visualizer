package golang

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"module-dependency-visualizer/internal/graph"
	"module-dependency-visualizer/internal/provider"
)

// GoProvider implements provider.Provider for Go modules.
type GoProvider struct {
	runner Runner
}

// New returns a GoProvider using the real os/exec Runner.
func New() *GoProvider {
	return &GoProvider{runner: ExecRunner{}}
}

// NewWithRunner returns a GoProvider using the given Runner (useful for testing).
func NewWithRunner(r Runner) *GoProvider {
	return &GoProvider{runner: r}
}

// Name returns "go".
func (p *GoProvider) Name() string { return "go" }

// Detect reports true when rootPath contains a go.mod file.
func (p *GoProvider) Detect(_ context.Context, rootPath string) (bool, error) {
	_, err := os.Stat(filepath.Join(rootPath, "go.mod"))
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("go: detect: %w", err)
	}
	return true, nil
}

// Parse runs `go mod graph` and `go list -m -json all` in rootPath and builds
// an immutable dependency graph.
func (p *GoProvider) Parse(ctx context.Context, rootPath string, _ provider.ParseOptions) (*graph.Graph, provider.Project, error) {
	goPath, err := findGoBinary()
	if err != nil {
		return nil, provider.Project{}, err
	}

	modGraphOut, err := p.runner.Run(ctx, rootPath, goPath, "mod", "graph")
	if err != nil {
		return nil, provider.Project{}, fmt.Errorf("go: mod graph: %w", err)
	}

	modListOut, err := p.runner.Run(ctx, rootPath, goPath, "list", "-m", "-json", "all")
	if err != nil {
		return nil, provider.Project{}, fmt.Errorf("go: list: %w", err)
	}

	edges, err := ParseModGraph(modGraphOut)
	if err != nil {
		return nil, provider.Project{}, err
	}

	infos, err := ParseModList(modListOut)
	if err != nil {
		return nil, provider.Project{}, err
	}

	g, mainModule, err := buildGraph(edges, infos)
	if err != nil {
		return nil, provider.Project{}, err
	}

	proj := provider.Project{
		Name:       mainModule,
		Language:   "go",
		RootPath:   rootPath,
		MainModule: mainModule,
	}

	return g, proj, nil
}

func findGoBinary() (string, error) {
	import_ := "go"
	path, err := lookPath(import_)
	if err != nil {
		return "", fmt.Errorf("go: binary not found in PATH — install Go from https://go.dev/dl: %w", err)
	}
	return path, nil
}
