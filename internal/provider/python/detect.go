package python

import (
	"os"
	"path/filepath"
)

// PackageManager identifies the Python package manager in use.
type PackageManager int

const (
	PackageManagerUnknown   PackageManager = iota
	PackageManagerPoetry                   // poetry.lock present
	PackageManagerPipenv                   // Pipfile.lock present
	PackageManagerUV                       // uv.lock present
	PackageManagerPip                      // requirements.txt present
	PackageManagerPyProject                // only pyproject.toml present
)

// detectPackageManager inspects rootPath and returns the most specific
// PackageManager that can be inferred from the files present.
// Priority: poetry > pipenv > uv > pip > pyproject.
func detectPackageManager(rootPath string) PackageManager {
	switch {
	case fileExists(filepath.Join(rootPath, "poetry.lock")):
		return PackageManagerPoetry
	case fileExists(filepath.Join(rootPath, "Pipfile.lock")):
		return PackageManagerPipenv
	case fileExists(filepath.Join(rootPath, "uv.lock")):
		return PackageManagerUV
	case fileExists(filepath.Join(rootPath, "requirements.txt")):
		return PackageManagerPip
	case fileExists(filepath.Join(rootPath, "pyproject.toml")):
		return PackageManagerPyProject
	default:
		return PackageManagerUnknown
	}
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
