package golang

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
)

// Runner executes an external command and returns its stdout.
// It is an interface so tests can inject a fake without shelling out.
type Runner interface {
	Run(ctx context.Context, dir string, name string, args ...string) ([]byte, error)
}

// ExecRunner is the production Runner that shells out via os/exec.
type ExecRunner struct{}

// Run executes name with args in dir, returns stdout on success.
// Stderr is captured and included in the error message on failure.
func (ExecRunner) Run(ctx context.Context, dir, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("exec %s %v: %w\nstderr: %s", name, args, err, stderr.String())
	}
	return stdout.Bytes(), nil
}
