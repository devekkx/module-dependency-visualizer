package npm

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"module-dependency-visualizer/internal/graph"
	"module-dependency-visualizer/internal/provider"
)

// packageJSON holds the fields we need from package.json.
type packageJSON struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// NPMProvider implements provider.Provider for Node.js / npm projects.
type NPMProvider struct {
	runner Runner
}

// New returns an NPMProvider backed by the real os/exec Runner.
func New() *NPMProvider {
	return &NPMProvider{runner: ExecRunner{}}
}

// NewWithRunner returns an NPMProvider using the provided Runner (useful for tests).
func NewWithRunner(r Runner) *NPMProvider {
	return &NPMProvider{runner: r}
}

// Name returns "npm".
func (p *NPMProvider) Name() string { return "npm" }

// Detect reports true when rootPath contains a package.json file.
func (p *NPMProvider) Detect(_ context.Context, rootPath string) (bool, error) {
	_, err := os.Stat(filepath.Join(rootPath, "package.json"))
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("npm: detect: %w", err)
	}
	return true, nil
}

// Parse reads the npm dependency manifest(s) at rootPath and returns an
// immutable graph plus project metadata.
//
// Resolution order:
//  1. Read package.json for project name/version.
//  2. If package-lock.json exists → parse it directly (no network needed).
//  3. Otherwise → run `npm ls --all --json` and parse its output.
func (p *NPMProvider) Parse(ctx context.Context, rootPath string, _ provider.ParseOptions) (*graph.Graph, provider.Project, error) {
	pkg, err := readPackageJSON(rootPath)
	if err != nil {
		return nil, provider.Project{}, err
	}

	lock, err := p.resolveLock(ctx, rootPath)
	if err != nil {
		return nil, provider.Project{}, err
	}

	g, rootName, err := buildGraph(lock)
	if err != nil {
		return nil, provider.Project{}, err
	}

	name := pkg.Name
	if name == "" {
		name = rootName
	}

	proj := provider.Project{
		Name:       name,
		Language:   "npm",
		RootPath:   rootPath,
		MainModule: name,
	}
	return g, proj, nil
}

// resolveLock returns a PackageLock either by reading package-lock.json or by
// invoking `npm ls --all --json` as a fallback.
func (p *NPMProvider) resolveLock(ctx context.Context, rootPath string) (*PackageLock, error) {
	lockPath := filepath.Join(rootPath, "package-lock.json")
	data, err := os.ReadFile(lockPath)
	if err == nil {
		lock, parseErr := ParsePackageLock(data)
		if parseErr != nil {
			return nil, parseErr
		}
		return lock, nil
	}
	if !os.IsNotExist(err) {
		return nil, fmt.Errorf("npm: read package-lock.json: %w", err)
	}

	// No lockfile — fall back to `npm ls`.
	return p.runNpmLS(ctx, rootPath)
}

// runNpmLS finds the npm binary and runs `npm ls --all --json`.
func (p *NPMProvider) runNpmLS(ctx context.Context, rootPath string) (*PackageLock, error) {
	npmPath, err := findNpmBinary()
	if err != nil {
		return nil, err
	}

	out, err := p.runner.Run(ctx, rootPath, npmPath, "ls", "--all", "--json")
	if err != nil {
		return nil, fmt.Errorf("npm: ls: %w", err)
	}

	lock, err := ParseNpmLS(out)
	if err != nil {
		return nil, err
	}
	return lock, nil
}

// readPackageJSON reads and parses the package.json at rootPath.
func readPackageJSON(rootPath string) (*packageJSON, error) {
	data, err := os.ReadFile(filepath.Join(rootPath, "package.json"))
	if err != nil {
		return nil, fmt.Errorf("npm: read package.json: %w", err)
	}
	var pkg packageJSON
	if err := json.Unmarshal(data, &pkg); err != nil {
		return nil, fmt.Errorf("npm: parse package.json: %w", err)
	}
	return &pkg, nil
}

// findNpmBinary locates the npm executable on PATH.
func findNpmBinary() (string, error) {
	path, err := lookPath("npm")
	if err != nil {
		return "", fmt.Errorf("npm: binary not found in PATH — install Node.js from https://nodejs.org: %w", err)
	}
	return path, nil
}
