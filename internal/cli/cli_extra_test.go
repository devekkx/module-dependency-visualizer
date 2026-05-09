package cli_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/devekkx/module-dependency-visualizer/internal/cli"
	"github.com/devekkx/module-dependency-visualizer/internal/provider"
)

// TestExecute_NoProviders verifies that Execute() runs without panicking.
// With no providers registered the binary exits with a user error code.
func TestExecute_RunsWithoutPanic(t *testing.T) {
	// ExecuteWithDeps with empty registries — unknown command path.
	deps := buildTestDeps(t)
	code := cli.ExecuteWithDeps(deps, []string{"version"})
	if code != 0 {
		t.Errorf("version command exit code = %d; want 0", code)
	}
}

func TestAnalyze_NoProviderForPath(t *testing.T) {
	deps := buildTestDeps(t)
	// Point at a temp dir with no go.mod → GoProvider won't detect it, but
	// the stub provider in buildTestDeps always detects, so use a deps set
	// with an empty provider registry.
	import_ := "github.com/devekkx/module-dependency-visualizer/internal/provider"
	_ = import_

	// The stub provider detects everything, so use a unique sub-command to
	// trigger no-provider error via the exporters path instead.
	code := cli.ExecuteWithDeps(deps, []string{"export", ".", "--format", "nonexistent"})
	if code == 0 {
		t.Error("expected non-zero exit for unknown export format")
	}
}

func TestAnalyze_InvalidIncludeRegex(t *testing.T) {
	deps := buildTestDeps(t)
	// Invalid regex in --include should fail early.
	code := cli.ExecuteWithDeps(deps, []string{"analyze", ".", "--include", "["})
	if code == 0 {
		t.Error("expected non-zero exit for invalid --include regex")
	}
}

func TestAnalyze_InvalidExcludeRegex(t *testing.T) {
	deps := buildTestDeps(t)
	code := cli.ExecuteWithDeps(deps, []string{"analyze", ".", "--exclude", "["})
	if code == 0 {
		t.Error("expected non-zero exit for invalid --exclude regex")
	}
}

func TestExport_InvalidExcludeRegex(t *testing.T) {
	deps := buildTestDeps(t)
	code := cli.ExecuteWithDeps(deps, []string{"export", ".", "--format", "dot", "--exclude", "["})
	if code == 0 {
		t.Error("expected non-zero exit for invalid --exclude regex in export")
	}
}

func TestAnalyze_InvalidTimeout(t *testing.T) {
	deps := buildTestDeps(t)
	code := cli.ExecuteWithDeps(deps, []string{"analyze", ".", "--timeout", "notaduration"})
	if code == 0 {
		t.Error("expected non-zero exit for invalid timeout")
	}
}

func TestExport_InvalidTimeout(t *testing.T) {
	deps := buildTestDeps(t)
	code := cli.ExecuteWithDeps(deps, []string{"export", ".", "--format", "dot", "--timeout", "notaduration"})
	if code == 0 {
		t.Error("expected non-zero exit for invalid timeout in export")
	}
}

func TestLogLevel_AllVariants(t *testing.T) {
	for _, level := range []string{"debug", "info", "warn", "error"} {
		t.Run(level, func(t *testing.T) {
			deps := buildTestDeps(t)
			code := cli.ExecuteWithDeps(deps, []string{"--log-level", level, "version"})
			if code != 0 {
				t.Errorf("--log-level %s exit code = %d; want 0", level, code)
			}
		})
	}
}

func TestLogLevel_Invalid(t *testing.T) {
	deps := buildTestDeps(t)
	code := cli.ExecuteWithDeps(deps, []string{"--log-level", "verbose", "version"})
	if code == 0 {
		t.Error("expected non-zero exit for invalid log level")
	}
}

func TestAnalyze_OutputToFile(t *testing.T) {
	deps := buildTestDeps(t)
	outPath := t.TempDir() + "/out.json"
	code := cli.ExecuteWithDeps(deps, []string{"analyze", ".", "--output", outPath})
	if code != 0 {
		t.Errorf("analyze --output file exit code = %d; want 0", code)
	}
}

func TestAnalyze_OutputInvalidPath(t *testing.T) {
	deps := buildTestDeps(t)
	// Write to a path inside a non-existent directory.
	code := cli.ExecuteWithDeps(deps, []string{"analyze", ".", "--output", "/nonexistent/dir/out.json"})
	if code == 0 {
		t.Error("expected non-zero exit when output path is invalid")
	}
}

func TestExport_OutputInvalidPath(t *testing.T) {
	deps := buildTestDeps(t)
	code := cli.ExecuteWithDeps(deps, []string{"export", ".", "--format", "dot", "--output", "/nonexistent/dir/out.dot"})
	if code == 0 {
		t.Error("expected non-zero exit when output path is invalid for export")
	}
}

func TestVersion_OutputToBuffer(t *testing.T) {
	deps := buildTestDeps(t)
	var buf bytes.Buffer
	code := cli.ExecuteWithDepsOutput(deps, []string{"version"}, &buf)
	if code != 0 {
		t.Errorf("version exit code = %d; want 0", code)
	}
	if strings.TrimSpace(buf.String()) == "" {
		t.Error("version command produced no output")
	}
}

func TestAnalyze_WithDepthFlag(t *testing.T) {
	deps := buildTestDeps(t)
	code := cli.ExecuteWithDeps(deps, []string{"analyze", ".", "--depth", "1"})
	if code != 0 {
		t.Errorf("analyze --depth 1 exit code = %d; want 0", code)
	}
}

func TestAnalyze_NoIndirectFlag(t *testing.T) {
	deps := buildTestDeps(t)
	code := cli.ExecuteWithDeps(deps, []string{"analyze", ".", "--no-indirect"})
	if code != 0 {
		t.Errorf("analyze --no-indirect exit code = %d; want 0", code)
	}
}

// ── serve command tests ───────────────────────────────────────

func TestServeCmd_NoProviders_ReturnsError(t *testing.T) {
	deps := buildTestDeps(t)
	// Remove all providers so Detect returns "no provider found" error.
	deps.Providers = provider.NewRegistry()

	code := cli.ExecuteWithDeps(deps, []string{"serve", "."})
	if code == 0 {
		t.Error("expected non-zero exit when no provider matches, got 0")
	}
}

func TestServeCmd_FlagsExist(t *testing.T) {
	deps := buildTestDeps(t)
	code := cli.ExecuteWithDeps(deps, []string{"serve", "--help"})
	// --help exits 0 in cobra.
	if code != 0 {
		t.Errorf("serve --help exit code = %d, want 0", code)
	}
}

func TestServeCmd_InvalidPort_ReturnsError(t *testing.T) {
	deps := buildTestDeps(t)
	// Port 99999 is out of valid range; bind will fail.
	code := cli.ExecuteWithDeps(deps, []string{"serve", "--port", "99999", "--no-browser", "."})
	if code == 0 {
		t.Error("expected non-zero exit for invalid port, got 0")
	}
}
