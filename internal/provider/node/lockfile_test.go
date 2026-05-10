package node_test

import (
	"os"
	"path/filepath"
	"testing"

	nodeprovider "github.com/devekkx/module-dependency-visualizer/internal/provider/node"
)

func fixtureFile(t *testing.T, parts ...string) []byte {
	t.Helper()
	path := filepath.Join(append([]string{"..", "..", "..", "testdata", "fixtures"}, parts...)...)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture %v: %v", parts, err)
	}
	return data
}

func TestParsePackageLock_V3_NodeCount(t *testing.T) {
	data := fixtureFile(t, "npm-simple", "package-lock.json")
	lock, err := nodeprovider.ParsePackageLock(data)
	if err != nil {
		t.Fatalf("ParsePackageLock: %v", err)
	}

	g, main, err := nodeprovider.LockToGraph(lock)
	if err != nil {
		t.Fatalf("LockToGraph: %v", err)
	}

	if main != "my-app" {
		t.Errorf("main = %q; want %q", main, "my-app")
	}
	if g.NodeCount() != 3 { // root + chalk + typescript
		t.Errorf("NodeCount = %d; want 3", g.NodeCount())
	}
	if g.EdgeCount() != 2 { // root→chalk, root→typescript
		t.Errorf("EdgeCount = %d; want 2", g.EdgeCount())
	}
}

func TestParsePackageLock_DevNodeMarked(t *testing.T) {
	data := fixtureFile(t, "npm-simple", "package-lock.json")
	lock, err := nodeprovider.ParsePackageLock(data)
	if err != nil {
		t.Fatalf("ParsePackageLock: %v", err)
	}
	g, _, err := nodeprovider.LockToGraph(lock)
	if err != nil {
		t.Fatalf("LockToGraph: %v", err)
	}

	for _, n := range g.Nodes() {
		if n.Name == "typescript" && !n.Dev {
			t.Error("typescript is a devDependency and should be Dev=true")
		}
		if n.Name == "chalk" && n.Dev {
			t.Error("chalk is a prod dependency and should not be Dev=true")
		}
	}
}

func TestParsePackageLock_InvalidJSON(t *testing.T) {
	_, err := nodeprovider.ParsePackageLock([]byte("{not valid}"))
	if err == nil {
		t.Error("expected error for invalid JSON, got nil")
	}
}

func TestParsePackageLock_MissingRootEntry(t *testing.T) {
	data := []byte(`{"lockfileVersion":3,"packages":{"node_modules/foo":{"version":"1.0.0"}}}`)
	lock, err := nodeprovider.ParsePackageLock(data)
	if err != nil {
		t.Fatalf("ParsePackageLock: %v", err)
	}
	_, _, err = nodeprovider.LockToGraph(lock)
	if err == nil {
		t.Error("expected error when root entry is missing in v3 lockfile")
	}
}

func TestParsePackageLock_V1_BuildsGraph(t *testing.T) {
	data := []byte(`{
		"name": "v1-app", "version": "1.0.0", "lockfileVersion": 1,
		"dependencies": {
			"chalk": {
				"version": "4.1.2",
				"requires": {"supports-color": "^7.0.0"},
				"dependencies": {
					"supports-color": {"version": "7.2.0"}
				}
			}
		}
	}`)
	lock, err := nodeprovider.ParsePackageLock(data)
	if err != nil {
		t.Fatalf("ParsePackageLock: %v", err)
	}
	g, main, err := nodeprovider.LockToGraph(lock)
	if err != nil {
		t.Fatalf("LockToGraph: %v", err)
	}
	if main != "v1-app" {
		t.Errorf("main = %q; want v1-app", main)
	}
	if g.NodeCount() != 3 {
		t.Errorf("NodeCount = %d; want 3", g.NodeCount())
	}
}

func TestPackageNameFromKey_Simple(t *testing.T) {
	cases := []struct{ key, want string }{
		{"node_modules/chalk", "chalk"},
		{"node_modules/@types/node", "@types/node"},
		{"node_modules/foo/node_modules/bar", "bar"},
		{"bare-key", "bare-key"},
	}
	for _, tc := range cases {
		got := nodeprovider.PackageNameFromKey(tc.key)
		if got != tc.want {
			t.Errorf("PackageNameFromKey(%q) = %q; want %q", tc.key, got, tc.want)
		}
	}
}
