package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"module-dependency-visualizer/internal/audit"
	"module-dependency-visualizer/internal/config"
	"module-dependency-visualizer/internal/docgen"
)

func newDocsCmd(deps *Deps) *cobra.Command {
	var output string
	var withAudit bool
	var timeout string

	cmd := &cobra.Command{
		Use:   "docs [path]",
		Short: "Generate a Markdown dependency document",
		Long: `Parse the dependency graph and write a Markdown document (DEPENDENCIES.md by default).

The document includes an overview table, direct and indirect dependency lists,
and an optional security section when --audit is set.

Examples:
  mdv docs .
  mdv docs . --audit
  mdv docs . --output docs/DEPENDENCIES.md`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			rootPath := "."
			if len(args) == 1 {
				rootPath = args[0]
			}

			absRoot, err := filepath.Abs(rootPath)
			if err != nil {
				return fmt.Errorf("docs: resolve path: %w", err)
			}

			d, err := parseDuration(timeout)
			if err != nil {
				return err
			}

			ctx, cancel := context.WithTimeout(context.Background(), d)
			defer cancel()

			g, proj, err := parseProject(ctx, deps, absRoot)
			if err != nil {
				return fmt.Errorf("docs: parse project: %w", err)
			}

			opts := docgen.Options{}

			if withAudit {
				auditor := audit.New(audit.Options{})
				auditCtx, auditCancel := context.WithTimeout(context.Background(), config.DefaultAuditTimeout)
				defer auditCancel()

				result, err := auditor.Run(auditCtx, g, proj.Language)
				if err != nil {
					fmt.Fprintf(os.Stderr, "warning: audit failed: %v\n", err)
				} else {
					opts.Audit = AuditResultToDTO(result)
				}
			}

			outPath := output
			if outPath == "" {
				outPath = filepath.Join(absRoot, "DEPENDENCIES.md")
			}

			f, err := os.Create(outPath)
			if err != nil {
				return fmt.Errorf("docs: create output file: %w", err)
			}
			defer f.Close()

			if err := docgen.Generate(f, g, proj, opts); err != nil {
				return fmt.Errorf("docs: generate: %w", err)
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Written to %s\n", outPath)
			return nil
		},
	}

	cmd.Flags().StringVarP(&output, "output", "o", "", "Output path (default: <path>/DEPENDENCIES.md)")
	cmd.Flags().BoolVar(&withAudit, "audit", false, "Embed vulnerability and license data in the document")
	cmd.Flags().StringVar(&timeout, "timeout", config.DefaultTimeout.String(), "Timeout for dependency resolution")

	return cmd
}
