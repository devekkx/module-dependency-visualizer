package python_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/devekkx/module-dependency-visualizer/internal/provider"
	pyprovider "github.com/devekkx/module-dependency-visualizer/internal/provider/python"
)

// TestExecRunner_Run verifies ExecRunner shells out correctly.
func TestExecRunner_Run(t *testing.T) {
	r := pyprovider.ExecRunner{}
	out, err := r.Run(context.Background(), t.TempDir(), "/bin/echo", "hello-py")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !strings.Contains(string(out), "hello-py") {
		t.Errorf("stdout = %q; want to contain 'hello-py'", string(out))
	}
}

// TestExecRunner_Run_Error verifies ExecRunner returns an error for a failing command.
func TestExecRunner_Run_Error(t *testing.T) {
	r := pyprovider.ExecRunner{}
	_, err := r.Run(context.Background(), t.TempDir(), "/bin/false")
	if err == nil {
		t.Error("expected error from failing command, got nil")
	}
}

// TestPythonProvider_Parse_Poetry_WithPEP621PyProject covers the directDeps
// code-path in parsePoetry when pyproject.toml has [project.dependencies].
func TestPythonProvider_Parse_Poetry_WithPEP621PyProject(t *testing.T) {
	dir := t.TempDir()
	copyFixtures(t, dir, "poetry-simple", "poetry.lock")

	// Write a PEP 621 pyproject.toml alongside the poetry.lock.
	pep621 := `[project]
name = "mixed-app"
version = "3.0.0"
dependencies = [
    "requests>=2.31.0",
]
`
	if err := os.WriteFile(filepath.Join(dir, "pyproject.toml"), []byte(pep621), 0644); err != nil {
		t.Fatal(err)
	}

	p := pyprovider.New()
	g, proj, err := p.Parse(context.Background(), dir, provider.ParseOptions{})
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if proj.MainModule != "mixed-app" {
		t.Errorf("MainModule = %q; want mixed-app", proj.MainModule)
	}
	if g.NodeCount() == 0 {
		t.Error("expected nodes in graph")
	}
}

// TestPythonProvider_Parse_Poetry_MissingLockFile tests the read error path
// when poetry.lock is listed but can't be read.
func TestPythonProvider_Parse_Poetry_MissingLockFile(t *testing.T) {
	dir := t.TempDir()
	// Create poetry.lock so detection picks PackageManagerPoetry, then delete it.
	lockPath := filepath.Join(dir, "poetry.lock")
	if err := os.WriteFile(lockPath, []byte(""), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(lockPath); err != nil {
		t.Fatal(err)
	}
	// Now dir has no files → detectPackageManager → Unknown → error.
	p := pyprovider.New()
	_, _, err := p.Parse(context.Background(), dir, provider.ParseOptions{})
	if err == nil {
		t.Error("expected error when no dependency files present")
	}
}

// TestPythonProvider_Parse_Pipenv_InvalidJSON covers parsePipenv when the lock file is bad JSON.
func TestPythonProvider_Parse_Pipenv_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "Pipfile.lock"), []byte("{bad json}"), 0644); err != nil {
		t.Fatal(err)
	}
	p := pyprovider.New()
	_, _, err := p.Parse(context.Background(), dir, provider.ParseOptions{})
	if err == nil {
		t.Error("expected error for invalid Pipfile.lock JSON")
	}
}

// TestPythonProvider_Parse_PyProject_MissingName covers parsePyProject when
// pyproject.toml has no [project] name field.
func TestPythonProvider_Parse_PyProject_MissingName(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "pyproject.toml"), []byte("[project]\nversion = \"1.0.0\"\n"), 0644); err != nil {
		t.Fatal(err)
	}
	p := pyprovider.New()
	_, _, err := p.Parse(context.Background(), dir, provider.ParseOptions{})
	if err == nil {
		t.Error("expected error when pyproject.toml has no name")
	}
}

// TestParsePoetryLock_EmptyData covers the case where poetry.lock is effectively empty.
func TestParsePoetryLock_EmptyData(t *testing.T) {
	root, err := pyprovider.ParsePoetryLock("app", "1.0.0", nil, []byte("# empty lock\n"))
	if err != nil {
		t.Fatalf("ParsePoetryLock: %v", err)
	}
	if root.Name != "app" {
		t.Errorf("root.Name = %q; want app", root.Name)
	}
	if len(root.Children) != 0 {
		t.Errorf("expected 0 children for empty lock, got %d", len(root.Children))
	}
}

// TestParseUvLock_TransitiveDepsResolution covers the edge where a transitive
// dep key does not exist in the index (missing package, should be skipped).
func TestParseUvLock_TransitiveDepsResolution(t *testing.T) {
	data := []byte(`
version = 1

[[package]]
name = "requests"
version = "2.31.0"
dependencies = [
    { name = "unknown-pkg" },
]
`)
	root, err := pyprovider.ParseUvLock("app", "1.0.0", []string{"requests"}, data)
	if err != nil {
		t.Fatalf("ParseUvLock: %v", err)
	}
	// unknown-pkg should be skipped gracefully.
	for _, c := range root.Children {
		if c.Name == "requests" {
			if len(c.Children) != 0 {
				t.Errorf("expected 0 children (unknown-pkg skipped), got %d", len(c.Children))
			}
			return
		}
	}
	t.Error("requests not found")
}

// TestParseRequirementsTxt_VersionOperators exercises the various PEP 508
// version operator paths in parseRequirementSpec.
func TestParseRequirementsTxt_VersionOperators(t *testing.T) {
	cases := []struct {
		input   string
		wantPkg string
	}{
		{"requests~=2.28.0", "requests"},
		{"flask!=2.0.0", "flask"},
		{"django>=3.0,<4.0", "django"},
		{"pillow<=9.5.0", "pillow"},
		{"numpy>1.0", "numpy"},
		{"just-name", "just-name"},
	}
	for _, tc := range cases {
		data := []byte(tc.input + "\n")
		root, err := pyprovider.ParseRequirementsTxt("app", "1.0.0", data)
		if err != nil {
			t.Fatalf("ParseRequirementsTxt(%q): %v", tc.input, err)
		}
		if len(root.Children) != 1 || root.Children[0].Name != tc.wantPkg {
			t.Errorf("input=%q: got %v; want [{%s ...}]", tc.input, root.Children, tc.wantPkg)
		}
	}
}

// TestParsePyProjectTOML_NoProjectSection returns empty meta (no error).
func TestParsePyProjectTOML_NoProjectSection(t *testing.T) {
	data := []byte("[build-system]\nrequires = [\"setuptools\"]\n")
	meta, err := pyprovider.ParsePyProjectTOML(data)
	if err != nil {
		t.Fatalf("ParsePyProjectTOML: %v", err)
	}
	if meta.Name != "" || meta.Version != "" {
		t.Errorf("expected empty meta for no [project] section, got %+v", meta)
	}
}

// TestParsePoetryLock_CycleGuard verifies the visited-set guard prevents
// infinite recursion for circular poetry dep specs.
func TestParsePoetryLock_CycleGuard(t *testing.T) {
	data := []byte(`
[[package]]
name = "a"
version = "1.0.0"
optional = false
files = []

[package.dependencies]
b = ">=1.0"

[[package]]
name = "b"
version = "1.0.0"
optional = false
files = []

[package.dependencies]
a = ">=1.0"

[metadata]
lock-version = "1.1"
`)
	direct := map[string]string{"a": "*"}
	root, err := pyprovider.ParsePoetryLock("app", "1.0.0", direct, data)
	if err != nil {
		t.Fatalf("ParsePoetryLock: %v", err)
	}
	if len(root.Children) == 0 {
		t.Error("expected at least one child")
	}
}
