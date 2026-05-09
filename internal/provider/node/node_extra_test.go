package node_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/devekkx/module-dependency-visualizer/internal/provider"
	nodeprovider "github.com/devekkx/module-dependency-visualizer/internal/provider/node"
)

// TestNodeProvider_Parse_NPM_InvalidLockfile ensures the lockfile parse error
// path is covered (valid JSON but v3 with no root "packages" entry).
func TestNodeProvider_Parse_NPM_InvalidLockfile(t *testing.T) {
	dir := t.TempDir()
	// Write a package.json so Detect passes.
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"name":"app","version":"1.0.0"}`), 0644); err != nil {
		t.Fatal(err)
	}
	// Write a lockfile that is valid JSON but missing the root "" entry.
	bad := []byte(`{"lockfileVersion":3,"packages":{"node_modules/foo":{"version":"1.0.0"}}}`)
	if err := os.WriteFile(filepath.Join(dir, "package-lock.json"), bad, 0644); err != nil {
		t.Fatal(err)
	}

	p := nodeprovider.NewWithRunnerAndLookPath(&fakeRunner{}, fakeLookPath)
	_, _, err := p.Parse(context.Background(), dir, provider.ParseOptions{})
	if err == nil {
		t.Error("expected error for lockfile missing root entry, got nil")
	}
}

// TestNodeProvider_Parse_NPM_FallbackParseError covers the path where npm ls
// returns output that ParseNpmLS cannot decode.
func TestNodeProvider_Parse_NPM_FallbackParseError(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"name":"app","version":"1.0.0"}`), 0644); err != nil {
		t.Fatal(err)
	}
	// No lock file → falls back to npm ls.
	runner := &fakeRunner{responses: map[string][]byte{"--json": []byte("{bad json}")}}
	p := nodeprovider.NewWithRunnerAndLookPath(runner, fakeLookPath)
	_, _, err := p.Parse(context.Background(), dir, provider.ParseOptions{})
	if err == nil {
		t.Error("expected error when npm ls output is invalid JSON")
	}
}

// TestNodeProvider_Parse_NPM_NpmLSRunnerError covers the runner failure branch
// in the npm fallback path.
func TestNodeProvider_Parse_NPM_NpmLSRunnerError(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"name":"app","version":"1.0.0"}`), 0644); err != nil {
		t.Fatal(err)
	}
	runner := &fakeRunner{err: errors.New("npm ls failed")}
	p := nodeprovider.NewWithRunnerAndLookPath(runner, fakeLookPath)
	_, _, err := p.Parse(context.Background(), dir, provider.ParseOptions{})
	if err == nil {
		t.Error("expected error when npm ls runner fails")
	}
}

// TestNodeProvider_Parse_NPM_FallbackNoName covers coalesce(root.Name, pkg.Name)
// when the npm ls output has no "name" field (root.Name == "").
func TestNodeProvider_Parse_NPM_FallbackNoName(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"name":"from-pkgjson","version":"1.0.0"}`), 0644); err != nil {
		t.Fatal(err)
	}
	out := []byte(`{"version":"1.0.0","dependencies":{"chalk":{"version":"5.3.0"}}}`)
	runner := &fakeRunner{responses: map[string][]byte{"--json": out}}
	p := nodeprovider.NewWithRunnerAndLookPath(runner, fakeLookPath)
	_, proj, err := p.Parse(context.Background(), dir, provider.ParseOptions{})
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if proj.MainModule != "from-pkgjson" {
		t.Errorf("MainModule = %q; want from-pkgjson (from package.json)", proj.MainModule)
	}
}

// TestNodeProvider_Parse_PNPM_ParseError covers pnpm ls output parse failure.
func TestNodeProvider_Parse_PNPM_ParseError(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"name":"app","version":"1.0.0"}`), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "pnpm-lock.yaml"), []byte("lockfileVersion: '6.0'"), 0644); err != nil {
		t.Fatal(err)
	}
	runner := &fakeRunner{responses: map[string][]byte{"--json": []byte("not json")}}
	p := nodeprovider.NewWithRunnerAndLookPath(runner, fakeLookPath)
	_, _, err := p.Parse(context.Background(), dir, provider.ParseOptions{})
	if err == nil {
		t.Error("expected error when pnpm ls output is invalid JSON")
	}
}

// TestNodeProvider_Parse_YarnV1_ParseError covers yarn list output parse failure.
func TestNodeProvider_Parse_YarnV1_ParseError(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"name":"app","version":"1.0.0"}`), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "yarn.lock"), []byte("# yarn v1"), 0644); err != nil {
		t.Fatal(err)
	}
	runner := &fakeRunner{responses: map[string][]byte{"--json": []byte("{bad}")}}
	p := nodeprovider.NewWithRunnerAndLookPath(runner, fakeLookPath)
	_, _, err := p.Parse(context.Background(), dir, provider.ParseOptions{})
	if err == nil {
		t.Error("expected error when yarn list output is invalid JSON")
	}
}

// TestNodeProvider_Parse_YarnBerry_RunnerError covers yarn berry runner failure.
func TestNodeProvider_Parse_YarnBerry_RunnerError(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"name":"app","version":"1.0.0"}`), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "yarn.lock"), []byte("__metadata:\n  version: 6"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".yarnrc.yml"), []byte("nodeLinker: node-modules"), 0644); err != nil {
		t.Fatal(err)
	}
	runner := &fakeRunner{err: errors.New("yarn not found")}
	p := nodeprovider.NewWithRunnerAndLookPath(runner, fakeLookPath)
	_, _, err := p.Parse(context.Background(), dir, provider.ParseOptions{})
	if err == nil {
		t.Error("expected error when yarn berry runner fails")
	}
}

// TestNodeProvider_Parse_Bun_RunnerError covers bun pm ls runner failure.
func TestNodeProvider_Parse_Bun_RunnerError(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"name":"app","version":"1.0.0"}`), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "bun.lockb"), []byte{}, 0644); err != nil {
		t.Fatal(err)
	}
	runner := &fakeRunner{err: errors.New("bun not found")}
	p := nodeprovider.NewWithRunnerAndLookPath(runner, fakeLookPath)
	_, _, err := p.Parse(context.Background(), dir, provider.ParseOptions{})
	if err == nil {
		t.Error("expected error when bun pm ls runner fails")
	}
}

// TestNodeProvider_Parse_LookPathError covers the binary-not-found error path.
func TestNodeProvider_Parse_LookPathError(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"name":"app","version":"1.0.0"}`), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "pnpm-lock.yaml"), []byte("lockfileVersion: '6.0'"), 0644); err != nil {
		t.Fatal(err)
	}
	failLookPath := func(_ string) (string, error) {
		return "", errors.New("binary not found")
	}
	p := nodeprovider.NewWithRunnerAndLookPath(&fakeRunner{}, failLookPath)
	_, _, err := p.Parse(context.Background(), dir, provider.ParseOptions{})
	if err == nil {
		t.Error("expected error when binary not found")
	}
}

// TestParseYarnInfo_WithTransitiveDeps covers the dep locator resolution path
// in ParseYarnInfo, including the git-hash stripping case.
func TestParseYarnInfo_WithTransitiveDeps(t *testing.T) {
	pkg := nodeprovider.PackageJSON{
		Name:         "app",
		Version:      "1.0.0",
		Dependencies: map[string]string{"express": "^4.18.0"},
	}
	ndjson := `{"value":"express@npm:4.18.2","children":{"Version":"4.18.2","Dependencies":[{"descriptor":"accepts@npm:~1.3.8","locator":"accepts@npm:1.3.8#abc123"}]}}` + "\n" +
		`{"value":"accepts@npm:1.3.8","children":{"Version":"1.3.8","Dependencies":[]}}` + "\n" +
		`{"value":"ws@workspace:.","children":{"Version":"0.0.0","Dependencies":[]}}` + "\n" + // non-npm locator, should be skipped
		"not valid json\n" // malformed line, should be skipped
	g, main, err := nodeprovider.ParseYarnInfo(pkg, []byte(ndjson))
	if err != nil {
		t.Fatalf("ParseYarnInfo: %v", err)
	}
	if main != "app" {
		t.Errorf("main = %q; want app", main)
	}
	if g.NodeCount() < 3 { // root + express + accepts
		t.Errorf("NodeCount = %d; want ≥ 3", g.NodeCount())
	}
}

// TestParseLockfileV3_NestedEntry verifies nested node_modules/ entries are
// correctly skipped during graph construction (not added as top-level nodes).
func TestParseLockfileV3_NestedEntry(t *testing.T) {
	data := []byte(`{
		"name":"app","version":"1.0.0","lockfileVersion":3,
		"packages":{
			"":{"name":"app","version":"1.0.0","dependencies":{"foo":"^1.0.0"}},
			"node_modules/foo":{"version":"1.0.0","dependencies":{"bar":"^1.0.0"}},
			"node_modules/bar":{"version":"1.0.0"},
			"node_modules/foo/node_modules/bar":{"version":"2.0.0"}
		}
	}`)
	lock, err := nodeprovider.ParsePackageLock(data)
	if err != nil {
		t.Fatalf("ParsePackageLock: %v", err)
	}
	g, _, err := nodeprovider.LockToGraph(lock)
	if err != nil {
		t.Fatalf("LockToGraph: %v", err)
	}
	// Nested "foo/node_modules/bar@2.0.0" should NOT appear as a top-level node.
	for _, n := range g.Nodes() {
		if n.Name == "bar" && n.Version == "2.0.0" {
			t.Error("nested node_modules entry should be skipped as a standalone node")
		}
	}
}

// TestParseLockfileV1_SharedDeps verifies the processed-set guard prevents
// infinite recursion and duplicate edges for shared dependencies.
func TestParseLockfileV1_SharedDeps(t *testing.T) {
	data := []byte(`{
		"name":"app","version":"1.0.0","lockfileVersion":1,
		"dependencies":{
			"foo":{"version":"1.0.0","dependencies":{"shared":{"version":"1.0.0"}}},
			"bar":{"version":"1.0.0","dependencies":{"shared":{"version":"1.0.0"}}}
		}
	}`)
	lock, err := nodeprovider.ParsePackageLock(data)
	if err != nil {
		t.Fatalf("ParsePackageLock: %v", err)
	}
	g, _, err := nodeprovider.LockToGraph(lock)
	if err != nil {
		t.Fatalf("LockToGraph: %v", err)
	}
	// 4 nodes: root, foo, bar, shared (shared is shared, not duplicated).
	if g.NodeCount() != 4 {
		t.Errorf("NodeCount = %d; want 4", g.NodeCount())
	}
}

// TestParseNpmLS_DiamondDep verifies buildGraph deduplicates nodes and edges
// for diamond dependency patterns (same dep from two parents).
func TestParseNpmLS_DiamondDep(t *testing.T) {
	data := []byte(`{
		"name":"app","version":"1.0.0",
		"dependencies":{
			"b":{"version":"1.0.0","dependencies":{"shared":{"version":"1.0.0"}}},
			"c":{"version":"1.0.0","dependencies":{"shared":{"version":"1.0.0"}}}
		}
	}`)
	root, err := nodeprovider.ParseNpmLS(data)
	if err != nil {
		t.Fatalf("ParseNpmLS: %v", err)
	}
	// buildGraph is exercised by going through parseNPM; test directly via
	// creating a dir with no lock file and using fakeRunner.
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"name":"app","version":"1.0.0"}`), 0644); err != nil {
		t.Fatal(err)
	}
	runner := &fakeRunner{responses: map[string][]byte{"--json": data}}
	p := nodeprovider.NewWithRunnerAndLookPath(runner, fakeLookPath)
	g, _, err := p.Parse(context.Background(), dir, provider.ParseOptions{})
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	// 4 nodes: root, b, c, shared (not duplicated).
	if g.NodeCount() != 4 {
		t.Errorf("NodeCount = %d; want 4 (shared deduplicated)", g.NodeCount())
	}
	_ = root
}

// TestParseYarnList_MalformedEntry verifies that a yarn tree entry without a
// version (no "@") is handled gracefully (empty version).
func TestParseYarnList_MalformedEntry(t *testing.T) {
	pkg := nodeprovider.PackageJSON{Name: "app", Version: "1.0.0"}
	// "noversion" has no "@" → splitAtVersion returns ("noversion", "").
	data := []byte(`{"type":"tree","data":{"type":"list","trees":[{"name":"noversion","children":[]}]}}`)
	root, err := nodeprovider.ParseYarnList(pkg, data)
	if err != nil {
		t.Fatalf("ParseYarnList: %v", err)
	}
	if len(root.Children) != 1 {
		t.Errorf("expected 1 child, got %d", len(root.Children))
	}
	if root.Children[0].Name != "noversion" {
		t.Errorf("child.Name = %q; want noversion", root.Children[0].Name)
	}
	if root.Children[0].Version != "" {
		t.Errorf("child.Version = %q; want empty string", root.Children[0].Version)
	}
}

// TestNewWithRunner verifies NewWithRunner constructs a provider with the given runner.
func TestNewWithRunner(t *testing.T) {
	p := nodeprovider.NewWithRunner(nodeprovider.ExecRunner{})
	if p.Name() != "node" {
		t.Errorf("Name() = %q; want node", p.Name())
	}
}

// TestExecRunner_Run verifies ExecRunner shells out and returns stdout.
func TestExecRunner_Run(t *testing.T) {
	r := nodeprovider.ExecRunner{}
	out, err := r.Run(context.Background(), t.TempDir(), "/bin/echo", "hello")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !strings.Contains(string(out), "hello") {
		t.Errorf("stdout = %q; want to contain 'hello'", string(out))
	}
}

// TestExecRunner_Run_Error verifies ExecRunner returns an error for a failing command.
func TestExecRunner_Run_Error(t *testing.T) {
	r := nodeprovider.ExecRunner{}
	_, err := r.Run(context.Background(), t.TempDir(), "/bin/false")
	if err == nil {
		t.Error("expected error from failing command, got nil")
	}
}

// TestParseYarnList_ScopedPackage covers the splitAtVersion scoped-package path
// where the entry starts with "@" (e.g. "@types/node@18.0.0").
func TestParseYarnList_ScopedPackage(t *testing.T) {
	pkg := nodeprovider.PackageJSON{Name: "app", Version: "1.0.0"}
	data := []byte(`{"type":"tree","data":{"type":"list","trees":[{"name":"@types/node@18.0.0","children":[]}]}}`)
	root, err := nodeprovider.ParseYarnList(pkg, data)
	if err != nil {
		t.Fatalf("ParseYarnList: %v", err)
	}
	if len(root.Children) != 1 {
		t.Fatalf("expected 1 child, got %d", len(root.Children))
	}
	if root.Children[0].Name != "@types/node" {
		t.Errorf("child.Name = %q; want @types/node", root.Children[0].Name)
	}
	if root.Children[0].Version != "18.0.0" {
		t.Errorf("child.Version = %q; want 18.0.0", root.Children[0].Version)
	}
}

// TestParseYarnInfo_ScopedLocator covers the parseLocator scoped-package path
// where the locator starts with "@" (e.g. "@types/node@npm:18.0.0").
func TestParseYarnInfo_ScopedLocator(t *testing.T) {
	pkg := nodeprovider.PackageJSON{
		Name:         "app",
		Version:      "1.0.0",
		Dependencies: map[string]string{"@types/node": "^18.0.0"},
	}
	ndjson := `{"value":"@types/node@npm:18.0.0","children":{"Version":"18.0.0","Dependencies":[]}}` + "\n"
	g, main, err := nodeprovider.ParseYarnInfo(pkg, []byte(ndjson))
	if err != nil {
		t.Fatalf("ParseYarnInfo: %v", err)
	}
	if main != "app" {
		t.Errorf("main = %q; want app", main)
	}
	if g.NodeCount() < 2 {
		t.Errorf("NodeCount = %d; want ≥ 2 (root + @types/node)", g.NodeCount())
	}
}

// TestParseBunLS_NoVersionEntry covers parseBunLSLine where the branch entry has
// no "@" separator (i.e. a package listed without a version), which should be
// skipped gracefully.
func TestParseBunLS_NoVersionEntry(t *testing.T) {
	pkg := nodeprovider.PackageJSON{Name: "app", Version: "1.0.0"}
	// "noversion" has no "@" in the branch position → parseBunLSLine returns false → skipped.
	data := []byte("bun pm v1.0.0\n├── noversion\n└── chalk@5.3.0\n")
	g, main, err := nodeprovider.ParseBunLS(pkg, data)
	if err != nil {
		t.Fatalf("ParseBunLS: %v", err)
	}
	if main != "app" {
		t.Errorf("main = %q; want app", main)
	}
	// root + chalk; noversion skipped
	if g.NodeCount() < 2 {
		t.Errorf("NodeCount = %d; want ≥ 2", g.NodeCount())
	}
}
