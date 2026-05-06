package cli

import (
	"github.com/spf13/cobra"

	"module-dependency-visualizer/internal/config"
	"module-dependency-visualizer/internal/logging"
)

// Deps holds the application-level dependencies injected at startup.
type Deps struct {
	Log       logging.Logger
	BuildInfo config.BuildInfo
}

// Execute builds the root command, wires all subcommands, and runs it.
// Returns the process exit code.
func Execute() int {
	deps := &Deps{
		Log:       logging.Discard(),
		BuildInfo: config.Default,
	}
	root := newRootCmd(deps)
	if err := root.Execute(); err != nil {
		return config.ExitUser
	}
	return config.ExitSuccess
}

func newRootCmd(deps *Deps) *cobra.Command {
	var logLevel string
	var timeout string

	cmd := &cobra.Command{
		Use:   "mdv",
		Short: "Language-agnostic module dependency visualizer",
		Long: `mdv parses dependency manifests (go.mod, package.json, requirements.txt, and more)
into a unified, interactive dependency graph.`,
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	cmd.PersistentFlags().StringVar(&logLevel, "log-level", "warn", "Log level (debug, info, warn, error)")
	cmd.PersistentFlags().StringVar(&timeout, "timeout", "30s", "Timeout for external commands (e.g. 30s, 2m)")
	cmd.PersistentFlags().Bool("no-color", false, "Disable color in output")

	cmd.AddCommand(newVersionCmd(deps))

	_ = logLevel
	_ = timeout

	return cmd
}
