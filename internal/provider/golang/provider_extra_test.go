package golang_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"module-dependency-visualizer/internal/provider"
	goprovider "module-dependency-visualizer/internal/provider/golang"
)

func TestGoProvider_Detect_StatError(t *testing.T) {
	// Create a directory where go.mod exists as a subdirectory (not a file),
	// then make it unreadable to trigger a stat error other than IsNotExist.
	dir := t.TempDir()
	goModDir := filepath.Join(dir, "go.mod")
	if err := os.Mkdir(goModDir, 0755); err != nil {
		t.Fatal(err)
	}
	// Make the inner directory non-readable so stat returns a permission error.
	if err := os.Chmod(dir, 0000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(dir, 0755) })

	if os.Getuid() == 0 {
		t.Skip("root ignores permission bits")
	}

	p := goprovider.New()
	_, err := p.Detect(context.Background(), dir)
	if err == nil {
		t.Error("expected error for unreadable directory, got nil")
	}
}

func TestGoProvider_Parse_ModGraphError(t *testing.T) {
	// Runner returns error specifically for "graph" sub-command.
	runner := &errorAfterNRunner{failOn: "graph"}
	p := goprovider.NewWithRunner(runner)
	_, _, err := p.Parse(context.Background(), "/fake", provider.ParseOptions{})
	if err == nil {
		t.Error("expected error when go mod graph fails")
	}
}

func TestGoProvider_Parse_ModListError(t *testing.T) {
	// Runner succeeds for "graph" but fails for "all" (go list).
	runner := &errorAfterNRunner{failOn: "all"}
	runner.responses = map[string][]byte{"graph": []byte("example.com/app github.com/x@v1\n")}
	p := goprovider.NewWithRunner(runner)
	_, _, err := p.Parse(context.Background(), "/fake", provider.ParseOptions{})
	if err == nil {
		t.Error("expected error when go list fails")
	}
}

// errorAfterNRunner fails when the last arg equals failOn.
type errorAfterNRunner struct {
	responses map[string][]byte
	failOn    string
}

func (r *errorAfterNRunner) Run(_ context.Context, _ string, _ string, args ...string) ([]byte, error) {
	last := args[len(args)-1]
	if last == r.failOn {
		return nil, errRunFailed
	}
	if r.responses != nil {
		if data, ok := r.responses[last]; ok {
			return data, nil
		}
	}
	return nil, errRunFailed
}

var errRunFailed = &runError{}

type runError struct{}

func (e *runError) Error() string { return "runner: command failed" }
