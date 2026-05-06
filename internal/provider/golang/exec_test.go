package golang_test

import (
	"context"
	"testing"

	goprovider "module-dependency-visualizer/internal/provider/golang"
)

func TestExecRunner_Run_ValidCommand(t *testing.T) {
	r := goprovider.ExecRunner{}
	ctx := context.Background()

	// `go version` is always present in the test environment.
	out, err := r.Run(ctx, ".", "go", "version")
	if err != nil {
		t.Fatalf("ExecRunner.Run: %v", err)
	}
	if len(out) == 0 {
		t.Error("ExecRunner.Run returned empty output for 'go version'")
	}
}

func TestExecRunner_Run_CommandNotFound(t *testing.T) {
	r := goprovider.ExecRunner{}
	_, err := r.Run(context.Background(), ".", "this-binary-does-not-exist-mdv-test")
	if err == nil {
		t.Error("expected error for non-existent binary, got nil")
	}
}

func TestExecRunner_Run_NonZeroExit(t *testing.T) {
	r := goprovider.ExecRunner{}
	// `go mod verify` on a directory without go.mod exits non-zero.
	_, err := r.Run(context.Background(), t.TempDir(), "go", "mod", "verify")
	if err == nil {
		t.Error("expected error for non-zero exit code, got nil")
	}
}

func TestExecRunner_Run_ContextTimeout(t *testing.T) {
	r := goprovider.ExecRunner{}
	ctx, cancel := context.WithTimeout(context.Background(), 0) // already timed out
	defer cancel()

	_, err := r.Run(ctx, ".", "go", "version")
	if err == nil {
		t.Error("expected error for expired context, got nil")
	}
}
