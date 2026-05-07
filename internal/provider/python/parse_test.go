package python_test

import (
	"os"
	"path/filepath"
	"testing"

	pyprovider "module-dependency-visualizer/internal/provider/python"
)

// fixtureFile reads a file from testdata/fixtures/<parts>.
func fixtureFile(t *testing.T, parts ...string) []byte {
	t.Helper()
	path := filepath.Join(append([]string{"..", "..", "..", "testdata", "fixtures"}, parts...)...)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture %v: %v", parts, err)
	}
	return data
}

// ---------------------------------------------------------------------------
// ParsePoetryLock
// ---------------------------------------------------------------------------

func TestParsePoetryLock_BuildsTree(t *testing.T) {
	data := fixtureFile(t, "poetry-simple", "poetry.lock")
	root, err := pyprovider.ParsePoetryLock("poetry-app", "1.0.0", nil, data)
	if err != nil {
		t.Fatalf("ParsePoetryLock: %v", err)
	}
	if root.Name != "poetry-app" {
		t.Errorf("root.Name = %q; want poetry-app", root.Name)
	}
	if len(root.Children) == 0 {
		t.Error("expected at least one direct dependency")
	}
}

func TestParsePoetryLock_WithDirectDeps(t *testing.T) {
	data := fixtureFile(t, "poetry-simple", "poetry.lock")
	direct := map[string]string{"requests": "^2.31.0"}
	root, err := pyprovider.ParsePoetryLock("my-app", "2.0.0", direct, data)
	if err != nil {
		t.Fatalf("ParsePoetryLock: %v", err)
	}
	if root.Name != "my-app" {
		t.Errorf("root.Name = %q; want my-app", root.Name)
	}
	// requests should be a direct child
	found := false
	for _, c := range root.Children {
		if c.Name == "requests" {
			found = true
			// requests depends on certifi, charset-normalizer, idna (not urllib3 because it's optional)
			if len(c.Children) == 0 {
				t.Error("requests should have transitive dependencies")
			}
		}
	}
	if !found {
		t.Error("requests not found as direct child")
	}
}

func TestParsePoetryLock_DevCategory(t *testing.T) {
	data := fixtureFile(t, "poetry-simple", "poetry.lock")
	root, err := pyprovider.ParsePoetryLock("app", "1.0.0", nil, data)
	if err != nil {
		t.Fatalf("ParsePoetryLock: %v", err)
	}
	// pytest has category = "dev"; with nil directDeps it should be excluded.
	for _, c := range root.Children {
		if c.Name == "pytest" {
			t.Error("pytest (category=dev) should be excluded when directDeps is nil")
		}
	}
}

func TestParsePoetryLock_TransitiveDeps(t *testing.T) {
	data := []byte(`
[[package]]
name = "requests"
version = "2.31.0"
optional = false
files = []

[package.dependencies]
certifi = ">=2017.4.17"

[[package]]
name = "certifi"
version = "2024.2.2"
optional = false
files = []

[metadata]
lock-version = "1.1"
`)
	direct := map[string]string{"requests": "^2.31.0"}
	root, err := pyprovider.ParsePoetryLock("app", "1.0.0", direct, data)
	if err != nil {
		t.Fatalf("ParsePoetryLock: %v", err)
	}
	if len(root.Children) != 1 {
		t.Fatalf("expected 1 direct dep (requests), got %d", len(root.Children))
	}
	req := root.Children[0]
	if req.Name != "requests" {
		t.Errorf("child.Name = %q; want requests", req.Name)
	}
	if len(req.Children) != 1 || req.Children[0].Name != "certifi" {
		t.Errorf("requests.Children = %v; want [certifi]", req.Children)
	}
}

func TestParsePoetryLock_SkipPythonDep(t *testing.T) {
	data := []byte(`
[[package]]
name = "flask"
version = "2.3.0"
optional = false
files = []

[package.dependencies]
python = ">=3.8"
werkzeug = ">=2.0"

[[package]]
name = "werkzeug"
version = "2.3.0"
optional = false
files = []

[metadata]
lock-version = "1.1"
`)
	direct := map[string]string{"flask": "*"}
	root, err := pyprovider.ParsePoetryLock("app", "1.0.0", direct, data)
	if err != nil {
		t.Fatalf("ParsePoetryLock: %v", err)
	}
	for _, c := range root.Children[0].Children {
		if c.Name == "python" {
			t.Error("'python' should be excluded from dependency tree")
		}
	}
}

// ---------------------------------------------------------------------------
// ParseUvLock
// ---------------------------------------------------------------------------

func TestParseUvLock_BuildsTree(t *testing.T) {
	data := fixtureFile(t, "uv-simple", "uv.lock")
	root, err := pyprovider.ParseUvLock("uv-app", "1.0.0", []string{"requests"}, data)
	if err != nil {
		t.Fatalf("ParseUvLock: %v", err)
	}
	if root.Name != "uv-app" {
		t.Errorf("root.Name = %q; want uv-app", root.Name)
	}
	if len(root.Children) == 0 {
		t.Error("expected at least one dep")
	}
	// requests should have transitive deps (certifi, charset-normalizer, etc.)
	for _, c := range root.Children {
		if c.Name == "requests" {
			if len(c.Children) == 0 {
				t.Error("requests should have transitive deps in uv.lock")
			}
			return
		}
	}
	t.Error("requests not found as direct dep")
}

func TestParseUvLock_NoDirectDeps_AllDirect(t *testing.T) {
	data := fixtureFile(t, "uv-simple", "uv.lock")
	// nil directDeps → all packages as direct children
	root, err := pyprovider.ParseUvLock("proj", "0.1.0", nil, data)
	if err != nil {
		t.Fatalf("ParseUvLock: %v", err)
	}
	if len(root.Children) == 0 {
		t.Error("expected direct children when directDeps is nil")
	}
}

func TestParseUvLock_EmptyDepsArray(t *testing.T) {
	data := []byte(`
version = 1

[[package]]
name = "simple"
version = "1.0.0"
dependencies = []
`)
	root, err := pyprovider.ParseUvLock("app", "1.0.0", nil, data)
	if err != nil {
		t.Fatalf("ParseUvLock: %v", err)
	}
	found := false
	for _, c := range root.Children {
		if c.Name == "simple" {
			found = true
		}
	}
	if !found {
		t.Error("simple package not found")
	}
}

// ---------------------------------------------------------------------------
// ParsePipfileLock
// ---------------------------------------------------------------------------

func TestParsePipfileLock_BuildsTree(t *testing.T) {
	data := fixtureFile(t, "pipenv-simple", "Pipfile.lock")
	root, err := pyprovider.ParsePipfileLock("pipenv-app", "1.0.0", data)
	if err != nil {
		t.Fatalf("ParsePipfileLock: %v", err)
	}
	if root.Name != "pipenv-app" {
		t.Errorf("root.Name = %q; want pipenv-app", root.Name)
	}
	// Expect requests (default), certifi (default), pytest (develop)
	if len(root.Children) != 3 {
		t.Errorf("Children len = %d; want 3 (requests + certifi + pytest)", len(root.Children))
	}
	devCount := 0
	for _, c := range root.Children {
		if c.Dev {
			devCount++
		}
	}
	if devCount != 1 {
		t.Errorf("dev children = %d; want 1 (pytest)", devCount)
	}
}

func TestParsePipfileLock_StripsVersionPrefix(t *testing.T) {
	data := []byte(`{
		"_meta": {},
		"default": {"requests": {"version": "==2.31.0"}},
		"develop": {}
	}`)
	root, err := pyprovider.ParsePipfileLock("app", "1.0.0", data)
	if err != nil {
		t.Fatalf("ParsePipfileLock: %v", err)
	}
	if len(root.Children) != 1 {
		t.Fatalf("expected 1 child, got %d", len(root.Children))
	}
	if root.Children[0].Version != "2.31.0" {
		t.Errorf("version = %q; want 2.31.0 (== prefix stripped)", root.Children[0].Version)
	}
}

func TestParsePipfileLock_InvalidJSON(t *testing.T) {
	_, err := pyprovider.ParsePipfileLock("app", "1.0.0", []byte("{bad json}"))
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

// ---------------------------------------------------------------------------
// ParseRequirementsTxt
// ---------------------------------------------------------------------------

func TestParseRequirementsTxt_BuildsTree(t *testing.T) {
	data := fixtureFile(t, "pip-simple", "requirements.txt")
	root, err := pyprovider.ParseRequirementsTxt("pip-app", "1.0.0", data)
	if err != nil {
		t.Fatalf("ParseRequirementsTxt: %v", err)
	}
	if root.Name != "pip-app" {
		t.Errorf("root.Name = %q; want pip-app", root.Name)
	}
	// 6 packages (5 + pytest)
	if len(root.Children) != 6 {
		t.Errorf("Children len = %d; want 6", len(root.Children))
	}
}

func TestParseRequirementsTxt_SkipsOptions(t *testing.T) {
	data := []byte("-r base.txt\n-e .\nrequests==2.31.0\ngit+https://github.com/foo/bar.git\n")
	root, err := pyprovider.ParseRequirementsTxt("app", "1.0.0", data)
	if err != nil {
		t.Fatalf("ParseRequirementsTxt: %v", err)
	}
	if len(root.Children) != 1 || root.Children[0].Name != "requests" {
		t.Errorf("expected only requests, got %v", root.Children)
	}
}

func TestParseRequirementsTxt_StripExtrasAndMarkers(t *testing.T) {
	data := []byte("flask[async]>=2.0.0; python_version >= '3.8'\n")
	root, err := pyprovider.ParseRequirementsTxt("app", "1.0.0", data)
	if err != nil {
		t.Fatalf("ParseRequirementsTxt: %v", err)
	}
	if len(root.Children) != 1 {
		t.Fatalf("expected 1 child, got %d", len(root.Children))
	}
	if root.Children[0].Name != "flask" {
		t.Errorf("Name = %q; want flask", root.Children[0].Name)
	}
}

func TestParseRequirementsTxt_LineContinuation(t *testing.T) {
	data := []byte("requests\\\n==2.31.0\n")
	root, err := pyprovider.ParseRequirementsTxt("app", "1.0.0", data)
	if err != nil {
		t.Fatalf("ParseRequirementsTxt: %v", err)
	}
	if len(root.Children) != 1 {
		t.Fatalf("expected 1 child, got %d", len(root.Children))
	}
	if root.Children[0].Name != "requests" {
		t.Errorf("Name = %q; want requests", root.Children[0].Name)
	}
}

func TestParseRequirementsTxt_Empty(t *testing.T) {
	root, err := pyprovider.ParseRequirementsTxt("app", "1.0.0", []byte("# comment only\n\n"))
	if err != nil {
		t.Fatalf("ParseRequirementsTxt: %v", err)
	}
	if len(root.Children) != 0 {
		t.Errorf("expected 0 children for comment-only file, got %d", len(root.Children))
	}
}

// ---------------------------------------------------------------------------
// ParsePyProjectTOML
// ---------------------------------------------------------------------------

func TestParsePyProjectTOML_PEP621(t *testing.T) {
	data := fixtureFile(t, "pyproject-simple", "pyproject.toml")
	meta, err := pyprovider.ParsePyProjectTOML(data)
	if err != nil {
		t.Fatalf("ParsePyProjectTOML: %v", err)
	}
	if meta.Name != "pyproject-app" {
		t.Errorf("Name = %q; want pyproject-app", meta.Name)
	}
	if meta.Version != "2.0.0" {
		t.Errorf("Version = %q; want 2.0.0", meta.Version)
	}
	if len(meta.DirectDeps) != 3 {
		t.Errorf("DirectDeps len = %d; want 3", len(meta.DirectDeps))
	}
}

func TestParsePyProjectTOML_Poetry(t *testing.T) {
	data := fixtureFile(t, "poetry-simple", "pyproject.toml")
	meta, err := pyprovider.ParsePyProjectTOML(data)
	if err != nil {
		t.Fatalf("ParsePyProjectTOML: %v", err)
	}
	if meta.Name != "poetry-app" {
		t.Errorf("Name = %q; want poetry-app", meta.Name)
	}
}

func TestParsePyProjectDeps_BuildsTree(t *testing.T) {
	data := fixtureFile(t, "pyproject-simple", "pyproject.toml")
	root, err := pyprovider.ParsePyProjectDeps(data)
	if err != nil {
		t.Fatalf("ParsePyProjectDeps: %v", err)
	}
	if root.Name != "pyproject-app" {
		t.Errorf("root.Name = %q; want pyproject-app", root.Name)
	}
	if len(root.Children) != 3 {
		t.Errorf("Children = %d; want 3 (requests, flask, black)", len(root.Children))
	}
}

func TestParsePyProjectDeps_MissingName(t *testing.T) {
	data := []byte("[project]\nversion = \"1.0.0\"\n")
	_, err := pyprovider.ParsePyProjectDeps(data)
	if err == nil {
		t.Error("expected error when [project] name is missing")
	}
}

func TestParsePyProjectTOML_InlineArray(t *testing.T) {
	data := []byte("[project]\nname = \"app\"\nversion = \"1.0.0\"\ndependencies = [\"requests>=2.0\", \"flask\"]\n")
	meta, err := pyprovider.ParsePyProjectTOML(data)
	if err != nil {
		t.Fatalf("ParsePyProjectTOML: %v", err)
	}
	if len(meta.DirectDeps) != 2 {
		t.Errorf("DirectDeps = %v; want 2", meta.DirectDeps)
	}
}
