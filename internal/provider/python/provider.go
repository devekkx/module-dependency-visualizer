package python

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"module-dependency-visualizer/internal/graph"
	"module-dependency-visualizer/internal/provider"
)

// PythonProvider implements provider.Provider for Python projects.
// It auto-detects Poetry, Pipenv, uv, pip (requirements.txt), and
// PEP 621 pyproject.toml projects from their lock/config files.
// All parsing is done directly from lock/config files with no subprocess calls.
type PythonProvider struct{}

// New returns a PythonProvider.
func New() *PythonProvider { return &PythonProvider{} }

// NewWithRunner returns a PythonProvider (Runner is accepted for API symmetry
// with other providers but is not used; Python parsing is file-based only).
func NewWithRunner(_ Runner) *PythonProvider { return &PythonProvider{} }

// NewWithRunnerAndLookPath returns a PythonProvider. The runner and lookPath
// arguments are accepted for API symmetry but not used.
func NewWithRunnerAndLookPath(_ Runner, _ func(string) (string, error)) *PythonProvider {
	return &PythonProvider{}
}

// Name returns "python".
func (p *PythonProvider) Name() string { return "python" }

// Detect reports true when rootPath contains a recognised Python project file.
func (p *PythonProvider) Detect(_ context.Context, rootPath string) (bool, error) {
	markers := []string{
		"pyproject.toml", "setup.py", "setup.cfg",
		"requirements.txt", "Pipfile", "Pipfile.lock",
		"poetry.lock", "uv.lock",
	}
	for _, m := range markers {
		if fileExists(filepath.Join(rootPath, m)) {
			return true, nil
		}
	}
	return false, nil
}

// Parse reads the dependency manifest at rootPath and builds an immutable graph.
// It auto-detects the package manager from lock / config files.
func (p *PythonProvider) Parse(_ context.Context, rootPath string, _ provider.ParseOptions) (*graph.Graph, provider.Project, error) {
	pm := detectPackageManager(rootPath)
	if pm == PackageManagerUnknown {
		return nil, provider.Project{}, fmt.Errorf("python: no dependency file found in %s", rootPath)
	}

	var root DepNode
	var err error

	switch pm {
	case PackageManagerPoetry:
		root, err = p.parsePoetry(rootPath)
	case PackageManagerPipenv:
		root, err = p.parsePipenv(rootPath)
	case PackageManagerUV:
		root, err = p.parseUV(rootPath)
	case PackageManagerPip:
		root, err = p.parsePip(rootPath)
	default: // PackageManagerPyProject
		root, err = p.parsePyProject(rootPath)
	}
	if err != nil {
		return nil, provider.Project{}, err
	}

	g, err := buildGraph(root)
	if err != nil {
		return nil, provider.Project{}, err
	}

	proj := provider.Project{
		Name:       root.Name,
		Language:   "python",
		RootPath:   rootPath,
		MainModule: root.Name,
	}
	return g, proj, nil
}

// parsePoetry reads poetry.lock (+ pyproject.toml for project meta and direct
// deps) and builds a full transitive dependency tree.
func (p *PythonProvider) parsePoetry(rootPath string) (DepNode, error) {
	meta, _ := readPyProjectMeta(rootPath)

	lockData, err := os.ReadFile(filepath.Join(rootPath, "poetry.lock"))
	if err != nil {
		return DepNode{}, fmt.Errorf("python: read poetry.lock: %w", err)
	}

	// Build direct deps map from pyproject.toml [tool.poetry.dependencies].
	var directDeps map[string]string
	if len(meta.DirectDeps) > 0 {
		directDeps = make(map[string]string, len(meta.DirectDeps))
		for _, dep := range meta.DirectDeps {
			name, _ := parseRequirementSpec(dep)
			if name != "" && name != "python" {
				directDeps[normalise(name)] = ""
			}
		}
	}

	rootName := coalesce(meta.Name, "project")
	return ParsePoetryLock(rootName, meta.Version, directDeps, lockData)
}

// parsePipenv reads Pipfile.lock for a flat dependency tree.
func (p *PythonProvider) parsePipenv(rootPath string) (DepNode, error) {
	meta, _ := readPyProjectMeta(rootPath)

	lockData, err := os.ReadFile(filepath.Join(rootPath, "Pipfile.lock"))
	if err != nil {
		return DepNode{}, fmt.Errorf("python: read Pipfile.lock: %w", err)
	}

	rootName := coalesce(meta.Name, "project")
	return ParsePipfileLock(rootName, meta.Version, lockData)
}

// parseUV reads uv.lock (+ pyproject.toml for project meta and direct deps).
func (p *PythonProvider) parseUV(rootPath string) (DepNode, error) {
	meta, _ := readPyProjectMeta(rootPath)

	lockData, err := os.ReadFile(filepath.Join(rootPath, "uv.lock"))
	if err != nil {
		return DepNode{}, fmt.Errorf("python: read uv.lock: %w", err)
	}

	var directDeps []string
	for _, dep := range meta.DirectDeps {
		name, _ := parseRequirementSpec(dep)
		if name != "" {
			directDeps = append(directDeps, name)
		}
	}

	rootName := coalesce(meta.Name, "project")
	return ParseUvLock(rootName, meta.Version, directDeps, lockData)
}

// parsePip reads requirements.txt for a flat dependency tree.
func (p *PythonProvider) parsePip(rootPath string) (DepNode, error) {
	meta, _ := readPyProjectMeta(rootPath)

	reqData, err := os.ReadFile(filepath.Join(rootPath, "requirements.txt"))
	if err != nil {
		return DepNode{}, fmt.Errorf("python: read requirements.txt: %w", err)
	}

	rootName := coalesce(meta.Name, "project")
	return ParseRequirementsTxt(rootName, meta.Version, reqData)
}

// parsePyProject reads pyproject.toml [project.dependencies] for a flat tree.
func (p *PythonProvider) parsePyProject(rootPath string) (DepNode, error) {
	data, err := os.ReadFile(filepath.Join(rootPath, "pyproject.toml"))
	if err != nil {
		return DepNode{}, fmt.Errorf("python: read pyproject.toml: %w", err)
	}
	return ParsePyProjectDeps(data)
}

// readPyProjectMeta reads name, version, and dependencies from pyproject.toml.
// Errors are silently swallowed; callers use the partial result.
func readPyProjectMeta(rootPath string) (PyProjectMeta, error) {
	data, err := os.ReadFile(filepath.Join(rootPath, "pyproject.toml"))
	if err != nil {
		return PyProjectMeta{}, err
	}
	return ParsePyProjectTOML(data)
}

func coalesce(a, b string) string {
	if a != "" {
		return a
	}
	return b
}
