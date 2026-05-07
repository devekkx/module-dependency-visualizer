package cli

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"time"

	"github.com/spf13/cobra"

	"module-dependency-visualizer/internal/config"
	"module-dependency-visualizer/internal/exporter"
	"module-dependency-visualizer/internal/logging"
	"module-dependency-visualizer/internal/provider"
)

// Deps holds the application-level dependencies injected at startup.
type Deps struct {
	Log       logging.Logger
	BuildInfo config.BuildInfo
	Providers *provider.Registry
	Exporters *exporter.Registry
}

// Execute builds the root command, wires all subcommands, and runs it.
// Returns the process exit code.
func Execute() int {
	deps := buildDeps()
	return ExecuteWithDeps(deps, nil)
}

// ExecuteWithDeps runs the CLI with the given deps and args.
// If args is nil, os.Args[1:] is used. Returns the process exit code.
func ExecuteWithDeps(deps *Deps, args []string) int {
	return ExecuteWithDepsOutput(deps, args, os.Stdout)
}

// ExecuteWithDepsOutput runs the CLI writing stdout to out. Useful in tests.
func ExecuteWithDepsOutput(deps *Deps, args []string, out io.Writer) int {
	root := newRootCmd(deps)
	root.SetOut(out)
	if args != nil {
		root.SetArgs(args)
	}
	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return exitUser
	}
	return exitSuccess
}

func buildDeps() *Deps {
	return &Deps{
		Log:       logging.Discard(),
		BuildInfo: config.Default,
		Providers: provider.NewRegistry(),
		Exporters: exporter.NewRegistry(),
	}
}

func newRootCmd(deps *Deps) *cobra.Command {
	var logLevel string
	var timeoutStr string

	cmd := &cobra.Command{
		Use:   "mdv",
		Short: "Language-agnostic module dependency visualizer",
		Long: `mdv parses dependency manifests (go.mod, package.json, requirements.txt, and more)
into a unified, interactive dependency graph.`,
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			level, err := parseLogLevel(logLevel)
			if err != nil {
				return err
			}
			deps.Log = logging.NewText(os.Stderr, level)

			if timeoutStr != "" {
				d, err := time.ParseDuration(timeoutStr)
				if err != nil {
					return fmt.Errorf("invalid timeout %q: %w", timeoutStr, err)
				}
				_ = d
			}
			return nil
		},
	}

	cmd.PersistentFlags().StringVar(&logLevel, "log-level", "warn", "Log level (debug, info, warn, error)")
	cmd.PersistentFlags().StringVar(&timeoutStr, "timeout", "30s", "Timeout for external commands (e.g. 30s, 2m)")
	cmd.PersistentFlags().Bool("no-color", false, "Disable color in log output")

	cmd.AddCommand(
		newAnalyzeCmd(deps),
		newExportCmd(deps),
		newVersionCmd(deps),
		newServeCmd(deps),
		newAuditCmd(deps),
	)

	return cmd
}

func parseLogLevel(s string) (slog.Level, error) {
	switch s {
	case "debug":
		return slog.LevelDebug, nil
	case "info":
		return slog.LevelInfo, nil
	case "warn":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	default:
		return slog.LevelWarn, fmt.Errorf("unknown log level %q; use debug, info, warn, or error", s)
	}
}
