package cli

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/devekkx/module-dependency-visualizer/internal/server"
)

const shutdownTimeout = 5 * time.Second

func openBrowser(url string) {
	var cmd string
	var args []string
	switch runtime.GOOS {
	case "darwin":
		cmd, args = "open", []string{url}
	case "windows":
		cmd, args = "cmd", []string{"/c", "start", url}
	default:
		cmd, args = "xdg-open", []string{url}
	}
	// Intentionally ignore errors - browser open is best-effort.
	_ = exec.Command(cmd, args...).Start()
}

// waitForSignal blocks until ctx is done. Overridden in tests to avoid
// blocking on real OS signals.
var waitForSignal = func(ctx context.Context) { <-ctx.Done() }

func newServeCmd(deps *Deps) *cobra.Command {
	var (
		port      int
		noBrowser bool
	)

	cmd := &cobra.Command{
		Use:   "serve [path]",
		Short: "Start the interactive dependency visualisation UI",
		Long: `Analyse the project at [path] (default: current directory) and launch
a local web server with an interactive D3.js force-directed graph.

The server runs until you press Ctrl-C.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			root := "."
			if len(args) == 1 {
				root = args[0]
			}

			srv := server.New(deps.Providers, server.Options{
				Port: port,
				Path: root,
			})

			ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
			defer stop()

			addr, err := srv.Start(ctx)
			if err != nil {
				return fmt.Errorf("serve: %w", err)
			}

			url := "http://" + addr
			cmd.Printf("MDV is running at %s\n", url)
			cmd.Printf("Press Ctrl-C to stop.\n")

			if !noBrowser {
				openBrowser(url)
			}

			waitForSignal(ctx)
			cmd.Println("\nShutting down…")

			shutCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
			defer cancel()
			return srv.Shutdown(shutCtx)
		},
	}

	cmd.Flags().IntVarP(&port, "port", "p", 0, "TCP port to listen on (0 = auto-assign)")
	cmd.Flags().BoolVar(&noBrowser, "no-browser", false, "Do not open the browser automatically")

	return cmd
}
