package cli

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"module-dependency-visualizer/internal/config"
	"module-dependency-visualizer/internal/exporter/jsonexp"
	"module-dependency-visualizer/internal/graph"
	"module-dependency-visualizer/internal/provider"
	"module-dependency-visualizer/internal/schema"
)

func newAnalyzeCmd(deps *Deps) *cobra.Command {
	var f analyzeFlags
	var timeout string

	cmd := &cobra.Command{
		Use:   "analyze <path>",
		Short: "Parse dependencies and emit JSON",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			rootPath := args[0]

			d, err := parseDuration(timeout)
			if err != nil {
				return err
			}

			ctx, cancel := context.WithTimeout(context.Background(), d)
			defer cancel()

			g, proj, err := parseProject(ctx, deps, rootPath)
			if err != nil {
				return fmt.Errorf("analyze: %w", err)
			}

			filtered, err := applyFilters(g, f)
			if err != nil {
				return err
			}

			enc := jsonexp.New(schema.EncodeOptions{
				Project: schema.Project{
					Name:       proj.Name,
					Language:   proj.Language,
					RootPath:   proj.RootPath,
					MainModule: proj.MainModule,
				},
			})

			w, closer, err := openOutput(f.Output)
			if err != nil {
				return err
			}
			defer closer()

			return enc.Write(ctx, w, filtered)
		},
	}

	addAnalyzeFlags(cmd, &f)
	cmd.Flags().StringVar(&timeout, "timeout", config.DefaultTimeout.String(), "Timeout for dependency resolution")

	return cmd
}

func newExportCmd(deps *Deps) *cobra.Command {
	var f analyzeFlags
	var format string
	var timeout string

	cmd := &cobra.Command{
		Use:   "export <path>",
		Short: "Parse dependencies and export to a specific format",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			rootPath := args[0]

			d, err := parseDuration(timeout)
			if err != nil {
				return err
			}

			ctx, cancel := context.WithTimeout(context.Background(), d)
			defer cancel()

			exp, err := deps.Exporters.Get(format)
			if err != nil {
				return fmt.Errorf("export: %w", err)
			}

			g, _, err := parseProject(ctx, deps, rootPath)
			if err != nil {
				return fmt.Errorf("export: %w", err)
			}

			filtered, err := applyFilters(g, f)
			if err != nil {
				return err
			}

			w, closer, err := openOutput(f.Output)
			if err != nil {
				return err
			}
			defer closer()

			return exp.Write(ctx, w, filtered)
		},
	}

	addAnalyzeFlags(cmd, &f)
	cmd.Flags().StringVarP(&format, "format", "f", "json", "Output format (json, dot, mermaid)")
	cmd.Flags().StringVar(&timeout, "timeout", config.DefaultTimeout.String(), "Timeout for dependency resolution")
	_ = cmd.MarkFlagRequired("format")

	return cmd
}

// parseProject auto-detects the provider and parses the project at rootPath.
func parseProject(ctx context.Context, deps *Deps, rootPath string) (*graph.Graph, provider.Project, error) {
	p, err := deps.Providers.Detect(ctx, rootPath)
	if err != nil {
		return nil, provider.Project{}, fmt.Errorf("no provider for %q: %w", rootPath, err)
	}
	deps.Log.Info("detected provider", "provider", p.Name(), "path", rootPath)

	g, proj, err := p.Parse(ctx, rootPath, provider.ParseOptions{MaxDepth: -1})
	if err != nil {
		return nil, provider.Project{}, fmt.Errorf("parse %q: %w", rootPath, err)
	}
	return g, proj, nil
}

// applyFilters applies the user-supplied flags to produce a filtered graph.
func applyFilters(g *graph.Graph, f analyzeFlags) (*graph.Graph, error) {
	include, err := compilePatterns(f.Include)
	if err != nil {
		return nil, err
	}
	exclude, err := compilePatterns(f.Exclude)
	if err != nil {
		return nil, err
	}

	return graph.Filter(g, graph.FilterOptions{
		MaxDepth:        f.Depth,
		IncludePatterns: include,
		ExcludePatterns: exclude,
		NoIndirect:      f.NoIndirect,
	})
}

// openOutput returns a writer and a close function. When path is "-", stdout is used.
func openOutput(path string) (*os.File, func(), error) {
	if path == "-" {
		return os.Stdout, func() {}, nil
	}
	f, err := os.Create(path) //nolint:gosec
	if err != nil {
		return nil, func() {}, fmt.Errorf("open output %q: %w", path, err)
	}
	return f, func() { _ = f.Close() }, nil
}

func parseDuration(s string) (time.Duration, error) {
	if s == "" {
		return config.DefaultTimeout, nil
	}
	d, err := time.ParseDuration(s)
	if err != nil {
		return 0, fmt.Errorf("invalid timeout %q: %w", s, err)
	}
	return d, nil
}
