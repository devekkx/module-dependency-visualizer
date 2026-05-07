package npm_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"module-dependency-visualizer/internal/provider"
	npmprovider "module-dependency-visualizer/internal/provider/npm"
)

// fakeRunner returns preconfigured output for specific commands.
// The response is keyed by the last argument to Run.
type fakeRunner struct {
	responses map[string][]byte
	err       error
}

func (f *fakeRunner) Run(_ context.Context, _ string, _ string, args ...string) ([]byte, error) {
	if f.err != nil {
		return nil, f.err
	}
	key := args[len(args)-1] // last arg: e.g. "--json"
	if data, ok := f.responses[key]; ok {
		return data, nil
	}
	return nil, errors.New("fakeRunner: no response for key " + key)
}

// npmLSFixture is a minimal `npm ls --all --json` output used by fallback tests.
var npmLSFixture = []byte(`{
	"name": "my-app",
	"version": "1.0.0",
	"dependencies": {
		"chalk": {
			"version": "5.3.0"
		}
	}
}`)

// TestNPMProvider_Name verifies the provider's ecosystem name.
func TestNPMProvider_Name(t *testing.T) {
	p := npmprovider.New()
	if p.Name() != "npm" {
		t.Errorf("Name() = %q; want %q", p.Name(), "npm")
	}
}

// TestNPMProvider_Detect_WithPackageJSON verifies detection when package.json exists.
func TestNPMProvider_Detect_WithPackageJSON(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"name":"x","version":"1.0.0"}`), 0644); err != nil {
		t.Fatal(err)
	}

	p := npmprovider.New()
	ok, err := p.Detect(context.Background(), dir)
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	if !ok {
		t.Error("Detect() = false; want true when package.json is present")
	}
}

// TestNPMProvider_Detect_NoPackageJSON verifies that an empty directory returns false.
func TestNPMProvider_Detect_NoPackageJSON(t *testing.T) {
	p := npmprovider.New()
	ok, err := p.Detect(context.Background(), t.TempDir())
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	if ok {
		t.Error("Detect() = true; want false when package.json is absent")
	}
}

// TestNPMProvider_Detect_StatError verifies that a permission error on the
// directory is propagated (skipped when running as root).
func TestNPMProvider_Detect_StatError(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("root ignores permission bits")
	}

	dir := t.TempDir()
	// Make the directory unreadable so os.Stat on package.json returns a
	// permission error (not IsNotExist).
	if err := os.Chmod(dir, 0000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(dir, 0755) })

	p := npmprovider.New()
	_, err := p.Detect(context.Background(), dir)
	if err == nil {
		t.Error("expected error for unreadable directory, got nil")
	}
}

// TestNPMProvider_Parse_WithLockfile verifies end-to-end parsing when both
// package.json and package-lock.json are present.
func TestNPMProvider_Parse_WithLockfile(t *testing.T) {
	dir := t.TempDir()

	pkgJSON := readNpmFixture(t, "package.json")
	lockJSON := readNpmFixture(t, "package-lock.json")

	if err := os.WriteFile(filepath.Join(dir, "package.json"), pkgJSON, 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "package-lock.json"), lockJSON, 0644); err != nil {
		t.Fatal(err)
	}

	p := npmprovider.New()
	g, proj, err := p.Parse(context.Background(), dir, provider.ParseOptions{MaxDepth: -1})
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	if proj.Language != "npm" {
		t.Errorf("Project.Language = %q; want %q", proj.Language, "npm")
	}
	if proj.Name != "my-app" {
		t.Errorf("Project.Name = %q; want %q", proj.Name, "my-app")
	}

	// Fixture has 1 root + 2 packages (chalk, typescript) = 3 nodes.
	if g.NodeCount() != 3 {
		t.Errorf("NodeCount = %d; want 3", g.NodeCount())
	}
	// Root → chalk, Root → typescript = 2 edges.
	if g.EdgeCount() != 2 {
		t.Errorf("EdgeCount = %d; want 2", g.EdgeCount())
	}
}

// TestNPMProvider_Parse_FallbackToNpmLS verifies that when no lockfile is
// present the provider invokes the runner and parses its output.
func TestNPMProvider_Parse_FallbackToNpmLS(t *testing.T) {
	dir := t.TempDir()

	pkgJSON := []byte(`{"name":"my-app","version":"1.0.0"}`)
	if err := os.WriteFile(filepath.Join(dir, "package.json"), pkgJSON, 0644); err != nil {
		t.Fatal(err)
	}
	// No package-lock.json → triggers npm ls fallback.

	runner := &fakeRunner{
		responses: map[string][]byte{
			"--json": npmLSFixture,
		},
	}
	p := npmprovider.NewWithRunner(runner)

	g, proj, err := p.Parse(context.Background(), dir, provider.ParseOptions{})
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	if proj.Language != "npm" {
		t.Errorf("Project.Language = %q; want %q", proj.Language, "npm")
	}
	// 1 root + 1 dep = 2 nodes.
	if g.NodeCount() != 2 {
		t.Errorf("NodeCount = %d; want 2", g.NodeCount())
	}
	if g.EdgeCount() != 1 {
		t.Errorf("EdgeCount = %d; want 1", g.EdgeCount())
	}
}

// TestNPMProvider_Parse_NpmLSError verifies that a runner failure is propagated
// as an error when no lockfile is available.
func TestNPMProvider_Parse_NpmLSError(t *testing.T) {
	dir := t.TempDir()

	pkgJSON := []byte(`{"name":"my-app","version":"1.0.0"}`)
	if err := os.WriteFile(filepath.Join(dir, "package.json"), pkgJSON, 0644); err != nil {
		t.Fatal(err)
	}

	runner := &fakeRunner{err: errors.New("npm: command not found")}
	p := npmprovider.NewWithRunner(runner)

	_, _, err := p.Parse(context.Background(), dir, provider.ParseOptions{})
	if err == nil {
		t.Error("expected error when npm ls fails, got nil")
	}
}
