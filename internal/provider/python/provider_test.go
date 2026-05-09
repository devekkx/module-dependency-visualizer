package python_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/devekkx/module-dependency-visualizer/internal/provider"
	pyprovider "github.com/devekkx/module-dependency-visualizer/internal/provider/python"
)

// fakeLookPath always succeeds, returning "/usr/bin/<name>".
var fakeLookPath = func(name string) (string, error) {
	return "/usr/bin/" + name, nil
}

func TestPythonProvider_Name(t *testing.T) {
	p := pyprovider.New()
	if p.Name() != "python" {
		t.Errorf("Name() = %q; want python", p.Name())
	}
}

func TestPythonProvider_NewWithRunner(t *testing.T) {
	p := pyprovider.NewWithRunner(pyprovider.ExecRunner{})
	if p.Name() != "python" {
		t.Errorf("Name() = %q; want python", p.Name())
	}
}

func TestPythonProvider_Detect_WithPoetryLock(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "poetry.lock"), []byte(""), 0644); err != nil {
		t.Fatal(err)
	}
	ok, err := pyprovider.New().Detect(context.Background(), dir)
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	if !ok {
		t.Error("Detect() = false; want true when poetry.lock is present")
	}
}

func TestPythonProvider_Detect_WithRequirementsTxt(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "requirements.txt"), []byte(""), 0644); err != nil {
		t.Fatal(err)
	}
	ok, err := pyprovider.New().Detect(context.Background(), dir)
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	if !ok {
		t.Error("Detect() = false; want true when requirements.txt is present")
	}
}

func TestPythonProvider_Detect_NoMarkers(t *testing.T) {
	ok, err := pyprovider.New().Detect(context.Background(), t.TempDir())
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	if ok {
		t.Error("Detect() = true; want false when no Python files present")
	}
}

func TestPythonProvider_Parse_Poetry(t *testing.T) {
	dir := t.TempDir()
	copyFixtures(t, dir, "poetry-simple", "poetry.lock", "pyproject.toml")

	p := pyprovider.NewWithRunnerAndLookPath(pyprovider.ExecRunner{}, fakeLookPath)
	g, proj, err := p.Parse(context.Background(), dir, provider.ParseOptions{})
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if proj.Language != "python" {
		t.Errorf("Language = %q; want python", proj.Language)
	}
	if proj.MainModule != "poetry-app" {
		t.Errorf("MainModule = %q; want poetry-app", proj.MainModule)
	}
	if g.NodeCount() == 0 {
		t.Error("expected nodes in poetry graph")
	}
}

func TestPythonProvider_Parse_Pipenv(t *testing.T) {
	dir := t.TempDir()
	copyFixtures(t, dir, "pipenv-simple", "Pipfile.lock")

	p := pyprovider.NewWithRunnerAndLookPath(pyprovider.ExecRunner{}, fakeLookPath)
	g, proj, err := p.Parse(context.Background(), dir, provider.ParseOptions{})
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if proj.Language != "python" {
		t.Errorf("Language = %q; want python", proj.Language)
	}
	if g.NodeCount() == 0 {
		t.Error("expected nodes in pipenv graph")
	}
}

func TestPythonProvider_Parse_UV(t *testing.T) {
	dir := t.TempDir()
	copyFixtures(t, dir, "uv-simple", "uv.lock", "pyproject.toml")

	p := pyprovider.NewWithRunnerAndLookPath(pyprovider.ExecRunner{}, fakeLookPath)
	g, proj, err := p.Parse(context.Background(), dir, provider.ParseOptions{})
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if proj.Language != "python" {
		t.Errorf("Language = %q; want python", proj.Language)
	}
	if proj.MainModule != "uv-app" {
		t.Errorf("MainModule = %q; want uv-app", proj.MainModule)
	}
	if g.NodeCount() == 0 {
		t.Error("expected nodes in uv graph")
	}
}

func TestPythonProvider_Parse_Pip(t *testing.T) {
	dir := t.TempDir()
	copyFixtures(t, dir, "pip-simple", "requirements.txt")

	p := pyprovider.NewWithRunnerAndLookPath(pyprovider.ExecRunner{}, fakeLookPath)
	g, proj, err := p.Parse(context.Background(), dir, provider.ParseOptions{})
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if proj.Language != "python" {
		t.Errorf("Language = %q; want python", proj.Language)
	}
	if g.NodeCount() == 0 {
		t.Error("expected nodes in pip graph")
	}
}

func TestPythonProvider_Parse_PyProject(t *testing.T) {
	dir := t.TempDir()
	copyFixtures(t, dir, "pyproject-simple", "pyproject.toml")

	p := pyprovider.NewWithRunnerAndLookPath(pyprovider.ExecRunner{}, fakeLookPath)
	g, proj, err := p.Parse(context.Background(), dir, provider.ParseOptions{})
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if proj.MainModule != "pyproject-app" {
		t.Errorf("MainModule = %q; want pyproject-app", proj.MainModule)
	}
	if g.NodeCount() == 0 {
		t.Error("expected nodes in pyproject graph")
	}
}

func TestPythonProvider_Parse_NoFiles(t *testing.T) {
	p := pyprovider.NewWithRunnerAndLookPath(pyprovider.ExecRunner{}, fakeLookPath)
	_, _, err := p.Parse(context.Background(), t.TempDir(), provider.ParseOptions{})
	if err == nil {
		t.Error("expected error when no Python project files are present")
	}
}

func TestPythonProvider_Parse_Poetry_MissingLock(t *testing.T) {
	dir := t.TempDir()
	// poetry.lock not present but pyproject.toml is - falls to pyproject path.
	copyFixtures(t, dir, "poetry-simple", "pyproject.toml")

	p := pyprovider.NewWithRunnerAndLookPath(pyprovider.ExecRunner{}, fakeLookPath)
	// Should succeed via pyproject fallback.
	_, proj, err := p.Parse(context.Background(), dir, provider.ParseOptions{})
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if proj.MainModule != "poetry-app" {
		t.Errorf("MainModule = %q; want poetry-app", proj.MainModule)
	}
}

func TestPythonProvider_Parse_Pipenv_MissingLock(t *testing.T) {
	dir := t.TempDir()
	// Write an empty requirements.txt as only file → pip path.
	if err := os.WriteFile(filepath.Join(dir, "requirements.txt"), []byte("flask==2.3.0\n"), 0644); err != nil {
		t.Fatal(err)
	}
	p := pyprovider.NewWithRunnerAndLookPath(pyprovider.ExecRunner{}, fakeLookPath)
	g, _, err := p.Parse(context.Background(), dir, provider.ParseOptions{})
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if g.NodeCount() == 0 {
		t.Error("expected at least root node")
	}
}

func copyFixtures(t *testing.T, dir, fixture string, names ...string) {
	t.Helper()
	for _, name := range names {
		data := fixtureFile(t, fixture, name)
		if err := os.WriteFile(filepath.Join(dir, name), data, 0644); err != nil {
			t.Fatalf("copyFixtures: %v", err)
		}
	}
}
