package node

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"module-dependency-visualizer/internal/graph"
	"module-dependency-visualizer/internal/provider"
)

// NodeProvider implements provider.Provider for Node.js projects.
// It auto-detects npm, pnpm, yarn (v1 and berry), and bun from lock files.
type NodeProvider struct {
	runner     Runner
	lookPathFn func(string) (string, error) // nil → uses package-level lookPath
}

// New returns a NodeProvider using the real os/exec Runner.
func New() *NodeProvider {
	return &NodeProvider{runner: ExecRunner{}}
}

// NewWithRunner returns a NodeProvider using the given Runner (for testing).
func NewWithRunner(r Runner) *NodeProvider {
	return &NodeProvider{runner: r}
}

// NewWithRunnerAndLookPath returns a NodeProvider with an injectable binary
// lookup function, allowing tests to avoid depending on system-installed
// package managers.
func NewWithRunnerAndLookPath(r Runner, lp func(string) (string, error)) *NodeProvider {
	return &NodeProvider{runner: r, lookPathFn: lp}
}

func (p *NodeProvider) findBinary(name string) (string, error) {
	if p.lookPathFn != nil {
		return p.lookPathFn(name)
	}
	return lookPath(name)
}

// Name returns "node".
func (p *NodeProvider) Name() string { return "node" }

// Detect reports true when rootPath contains a package.json file.
func (p *NodeProvider) Detect(_ context.Context, rootPath string) (bool, error) {
	_, err := os.Stat(filepath.Join(rootPath, "package.json"))
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("node: detect: %w", err)
	}
	return true, nil
}

// Parse reads the dependency manifest at rootPath and builds an immutable graph.
// It auto-detects the package manager from lock files and selects the appropriate
// parse strategy.
func (p *NodeProvider) Parse(ctx context.Context, rootPath string, _ provider.ParseOptions) (*graph.Graph, provider.Project, error) {
	pkgJSON, err := readPackageJSON(rootPath)
	if err != nil {
		return nil, provider.Project{}, err
	}

	pm := detectPackageManager(rootPath)

	var g *graph.Graph
	var mainModule string

	switch pm {
	case PackageManagerNPM, PackageManagerUnknown:
		g, mainModule, err = p.parseNPM(ctx, rootPath, pkgJSON)
	case PackageManagerPNPM:
		g, mainModule, err = p.parsePNPM(ctx, rootPath, pkgJSON)
	case PackageManagerYarnV1:
		g, mainModule, err = p.parseYarnV1(ctx, rootPath, pkgJSON)
	case PackageManagerYarnBerry:
		g, mainModule, err = p.parseYarnBerry(ctx, rootPath, pkgJSON)
	case PackageManagerBun:
		g, mainModule, err = p.parseBun(ctx, rootPath, pkgJSON)
	}
	if err != nil {
		return nil, provider.Project{}, err
	}

	proj := provider.Project{
		Name:       mainModule,
		Language:   "node",
		RootPath:   rootPath,
		MainModule: mainModule,
	}
	return g, proj, nil
}

// parseNPM tries package-lock.json first, then falls back to npm ls.
func (p *NodeProvider) parseNPM(ctx context.Context, rootPath string, pkg PackageJSON) (*graph.Graph, string, error) {
	lockPath := filepath.Join(rootPath, "package-lock.json")
	if data, err := os.ReadFile(lockPath); err == nil {
		lock, err := ParsePackageLock(data)
		if err != nil {
			return nil, "", err
		}
		return LockToGraph(lock)
	}

	npmPath, err := p.findBinary("npm")
	if err != nil {
		return nil, "", fmt.Errorf("node: npm not found in PATH: %w", err)
	}
	out, err := p.runner.Run(ctx, rootPath, npmPath, "ls", "--all", "--json")
	if err != nil {
		return nil, "", fmt.Errorf("node: npm ls: %w", err)
	}
	root, err := ParseNpmLS(out)
	if err != nil {
		return nil, "", err
	}
	g, err := buildGraph(root)
	return g, coalesce(root.Name, pkg.Name), err
}

// parsePNPM runs pnpm ls --depth=Infinity --json.
func (p *NodeProvider) parsePNPM(ctx context.Context, rootPath string, pkg PackageJSON) (*graph.Graph, string, error) {
	pnpmPath, err := p.findBinary("pnpm")
	if err != nil {
		return nil, "", fmt.Errorf("node: pnpm not found in PATH: %w", err)
	}
	out, err := p.runner.Run(ctx, rootPath, pnpmPath, "ls", "--depth=Infinity", "--json")
	if err != nil {
		return nil, "", fmt.Errorf("node: pnpm ls: %w", err)
	}
	root, err := ParsePnpmLS(out)
	if err != nil {
		return nil, "", err
	}
	g, err := buildGraph(root)
	return g, coalesce(root.Name, pkg.Name), err
}

// parseYarnV1 runs yarn list --json (yarn v1).
func (p *NodeProvider) parseYarnV1(ctx context.Context, rootPath string, pkg PackageJSON) (*graph.Graph, string, error) {
	yarnPath, err := p.findBinary("yarn")
	if err != nil {
		return nil, "", fmt.Errorf("node: yarn not found in PATH: %w", err)
	}
	out, err := p.runner.Run(ctx, rootPath, yarnPath, "list", "--json")
	if err != nil {
		return nil, "", fmt.Errorf("node: yarn list: %w", err)
	}
	root, err := ParseYarnList(pkg, out)
	if err != nil {
		return nil, "", err
	}
	g, err := buildGraph(root)
	return g, pkg.Name, err
}

// parseYarnBerry runs yarn info --all --json (yarn v2+).
func (p *NodeProvider) parseYarnBerry(ctx context.Context, rootPath string, pkg PackageJSON) (*graph.Graph, string, error) {
	yarnPath, err := p.findBinary("yarn")
	if err != nil {
		return nil, "", fmt.Errorf("node: yarn not found in PATH: %w", err)
	}
	out, err := p.runner.Run(ctx, rootPath, yarnPath, "info", "--all", "--json")
	if err != nil {
		return nil, "", fmt.Errorf("node: yarn info: %w", err)
	}
	return ParseYarnInfo(pkg, out)
}

// parseBun runs bun pm ls (bun).
func (p *NodeProvider) parseBun(ctx context.Context, rootPath string, pkg PackageJSON) (*graph.Graph, string, error) {
	bunPath, err := p.findBinary("bun")
	if err != nil {
		return nil, "", fmt.Errorf("node: bun not found in PATH: %w", err)
	}
	out, err := p.runner.Run(ctx, rootPath, bunPath, "pm", "ls")
	if err != nil {
		return nil, "", fmt.Errorf("node: bun pm ls: %w", err)
	}
	return ParseBunLS(pkg, out)
}

func readPackageJSON(rootPath string) (PackageJSON, error) {
	data, err := os.ReadFile(filepath.Join(rootPath, "package.json"))
	if err != nil {
		return PackageJSON{}, fmt.Errorf("node: read package.json: %w", err)
	}
	return ParsePackageJSON(data)
}
