package node_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/devekkx/module-dependency-visualizer/internal/provider"
	nodeprovider "github.com/devekkx/module-dependency-visualizer/internal/provider/node"
)

// fakeLookPath always succeeds, returning "/usr/bin/<name>".
// This prevents tests from failing in environments without npm/pnpm/yarn/bun.
var fakeLookPath = func(name string) (string, error) {
	return "/usr/bin/" + name, nil
}

// fakeRunner returns configured output for specific commands, keyed by the
// last non-flag argument.
type fakeRunner struct {
	responses map[string][]byte
	err       error
}

func (f *fakeRunner) Run(_ context.Context, _ string, _ string, args ...string) ([]byte, error) {
	if f.err != nil {
		return nil, f.err
	}
	// Key on the last argument.
	key := args[len(args)-1]
	if data, ok := f.responses[key]; ok {
		return data, nil
	}
	return nil, errors.New("fakeRunner: no response for key " + key)
}

func TestNodeProvider_Name(t *testing.T) {
	p := nodeprovider.New()
	if p.Name() != "node" {
		t.Errorf("Name() = %q; want %q", p.Name(), "node")
	}
}

func TestNodeProvider_Detect_WithPackageJSON(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"name":"x","version":"1.0.0"}`), 0644); err != nil {
		t.Fatal(err)
	}
	ok, err := nodeprovider.New().Detect(context.Background(), dir)
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	if !ok {
		t.Error("Detect() = false; want true when package.json is present")
	}
}

func TestNodeProvider_Detect_NoPackageJSON(t *testing.T) {
	ok, err := nodeprovider.New().Detect(context.Background(), t.TempDir())
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	if ok {
		t.Error("Detect() = true; want false when package.json is absent")
	}
}

func TestNodeProvider_Detect_StatError(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("root ignores permission bits")
	}
	dir := t.TempDir()
	if err := os.Chmod(dir, 0000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(dir, 0755) })

	_, err := nodeprovider.New().Detect(context.Background(), dir)
	if err == nil {
		t.Error("expected error for unreadable directory, got nil")
	}
}

func TestNodeProvider_Parse_NPM_WithLockfile(t *testing.T) {
	dir := t.TempDir()
	copyFixtures(t, dir, "npm-simple", "package.json", "package-lock.json")

	p := nodeprovider.NewWithRunnerAndLookPath(&fakeRunner{err: errors.New("should not be called")}, fakeLookPath)
	g, proj, err := p.Parse(context.Background(), dir, provider.ParseOptions{})
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if proj.Language != "node" {
		t.Errorf("Language = %q; want node", proj.Language)
	}
	if proj.MainModule != "my-app" {
		t.Errorf("MainModule = %q; want my-app", proj.MainModule)
	}
	if g.NodeCount() != 3 {
		t.Errorf("NodeCount = %d; want 3", g.NodeCount())
	}
}

func TestNodeProvider_Parse_NPM_FallbackToNpmLS(t *testing.T) {
	dir := t.TempDir()
	// Only package.json - no lock file → must call npm ls.
	copyFixtures(t, dir, "npm-simple", "package.json")

	npmLSOut := []byte(`{
		"name":"my-app","version":"1.0.0",
		"dependencies":{"chalk":{"version":"5.3.0"}}
	}`)
	runner := &fakeRunner{responses: map[string][]byte{"--json": npmLSOut}}
	p := nodeprovider.NewWithRunnerAndLookPath(runner, fakeLookPath)
	g, proj, err := p.Parse(context.Background(), dir, provider.ParseOptions{})
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if proj.MainModule != "my-app" {
		t.Errorf("MainModule = %q; want my-app", proj.MainModule)
	}
	if g.NodeCount() != 2 {
		t.Errorf("NodeCount = %d; want 2", g.NodeCount())
	}
}

func TestNodeProvider_Parse_PNPM(t *testing.T) {
	dir := t.TempDir()
	copyFixtures(t, dir, "pnpm-simple", "package.json")
	// Simulate pnpm-lock.yaml presence.
	if err := os.WriteFile(filepath.Join(dir, "pnpm-lock.yaml"), []byte("lockfileVersion: '6.0'"), 0644); err != nil {
		t.Fatal(err)
	}

	pnpmOut := fixtureFile(t, "pnpm-simple", "pnpm-ls.json")
	runner := &fakeRunner{responses: map[string][]byte{"--json": pnpmOut}}
	p := nodeprovider.NewWithRunnerAndLookPath(runner, fakeLookPath)
	g, proj, err := p.Parse(context.Background(), dir, provider.ParseOptions{})
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if proj.Language != "node" {
		t.Errorf("Language = %q; want node", proj.Language)
	}
	if g.NodeCount() == 0 {
		t.Error("expected nodes in pnpm graph")
	}
}

func TestNodeProvider_Parse_YarnV1(t *testing.T) {
	dir := t.TempDir()
	copyFixtures(t, dir, "yarn-simple", "package.json")
	if err := os.WriteFile(filepath.Join(dir, "yarn.lock"), []byte("# yarn.lock v1"), 0644); err != nil {
		t.Fatal(err)
	}

	yarnOut := fixtureFile(t, "yarn-simple", "yarn-list.json")
	runner := &fakeRunner{responses: map[string][]byte{"--json": yarnOut}}
	p := nodeprovider.NewWithRunnerAndLookPath(runner, fakeLookPath)
	g, proj, err := p.Parse(context.Background(), dir, provider.ParseOptions{})
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if proj.MainModule != "yarn-app" {
		t.Errorf("MainModule = %q; want yarn-app", proj.MainModule)
	}
	if g.NodeCount() == 0 {
		t.Error("expected nodes in yarn v1 graph")
	}
}

func TestNodeProvider_Parse_YarnBerry(t *testing.T) {
	dir := t.TempDir()
	copyFixtures(t, dir, "yarn-simple", "package.json")
	if err := os.WriteFile(filepath.Join(dir, "yarn.lock"), []byte("__metadata:\n  version: 6"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".yarnrc.yml"), []byte("nodeLinker: node-modules"), 0644); err != nil {
		t.Fatal(err)
	}

	berryOut := []byte(`{"value":"chalk@npm:5.3.0","children":{"Version":"5.3.0","Dependencies":[]}}` + "\n")
	runner := &fakeRunner{responses: map[string][]byte{"--json": berryOut}}
	p := nodeprovider.NewWithRunnerAndLookPath(runner, fakeLookPath)
	g, _, err := p.Parse(context.Background(), dir, provider.ParseOptions{})
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if g.NodeCount() == 0 {
		t.Error("expected nodes in yarn berry graph")
	}
}

func TestNodeProvider_Parse_Bun(t *testing.T) {
	dir := t.TempDir()
	copyFixtures(t, dir, "bun-simple", "package.json")
	if err := os.WriteFile(filepath.Join(dir, "bun.lockb"), []byte{}, 0644); err != nil {
		t.Fatal(err)
	}

	bunOut := fixtureFile(t, "bun-simple", "bun-pm-ls.txt")
	runner := &fakeRunner{responses: map[string][]byte{"ls": bunOut}}
	p := nodeprovider.NewWithRunnerAndLookPath(runner, fakeLookPath)
	g, proj, err := p.Parse(context.Background(), dir, provider.ParseOptions{})
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if proj.MainModule != "bun-app" {
		t.Errorf("MainModule = %q; want bun-app", proj.MainModule)
	}
	if g.NodeCount() == 0 {
		t.Error("expected nodes in bun graph")
	}
}

func TestNodeProvider_Parse_RunnerError(t *testing.T) {
	dir := t.TempDir()
	copyFixtures(t, dir, "pnpm-simple", "package.json")
	if err := os.WriteFile(filepath.Join(dir, "pnpm-lock.yaml"), []byte("lockfileVersion: '6.0'"), 0644); err != nil {
		t.Fatal(err)
	}

	runner := &fakeRunner{err: errors.New("pnpm not found")}
	p := nodeprovider.NewWithRunnerAndLookPath(runner, fakeLookPath)
	_, _, err := p.Parse(context.Background(), dir, provider.ParseOptions{})
	if err == nil {
		t.Error("expected error when runner fails")
	}
}

func TestNodeProvider_Parse_MissingPackageJSON(t *testing.T) {
	p := nodeprovider.NewWithRunnerAndLookPath(&fakeRunner{}, fakeLookPath)
	_, _, err := p.Parse(context.Background(), t.TempDir(), provider.ParseOptions{})
	if err == nil {
		t.Error("expected error when package.json is missing")
	}
}

// copyFixtures copies named files from a fixture directory into dir.
func copyFixtures(t *testing.T, dir, fixture string, names ...string) {
	t.Helper()
	for _, name := range names {
		data := fixtureFile(t, fixture, name)
		if err := os.WriteFile(filepath.Join(dir, name), data, 0644); err != nil {
			t.Fatalf("copyFixtures: %v", err)
		}
	}
}
