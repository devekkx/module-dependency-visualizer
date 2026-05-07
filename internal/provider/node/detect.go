package node

import (
	"os"
	"path/filepath"
)

// PackageManager identifies which Node.js package manager a project uses.
type PackageManager int

const (
	PackageManagerUnknown    PackageManager = iota // defaults to npm on Parse
	PackageManagerNPM                              // package-lock.json present
	PackageManagerPNPM                             // pnpm-lock.yaml present
	PackageManagerYarnV1                           // yarn.lock, no .yarnrc.yml
	PackageManagerYarnBerry                        // yarn.lock + .yarnrc.yml (v2+)
	PackageManagerBun                              // bun.lockb or bun.lock present
)

// detectPackageManager examines rootPath for lock files and returns the
// package manager in use. Returns PackageManagerUnknown when only package.json
// is present (Parse defaults to npm in that case).
func detectPackageManager(rootPath string) PackageManager {
	switch {
	case fileExists(filepath.Join(rootPath, "pnpm-lock.yaml")):
		return PackageManagerPNPM
	case fileExists(filepath.Join(rootPath, "bun.lockb")) ||
		fileExists(filepath.Join(rootPath, "bun.lock")):
		return PackageManagerBun
	case fileExists(filepath.Join(rootPath, "yarn.lock")):
		if fileExists(filepath.Join(rootPath, ".yarnrc.yml")) {
			return PackageManagerYarnBerry
		}
		return PackageManagerYarnV1
	case fileExists(filepath.Join(rootPath, "package-lock.json")):
		return PackageManagerNPM
	default:
		return PackageManagerUnknown
	}
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
