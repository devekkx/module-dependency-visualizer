package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/devekkx/module-dependency-visualizer/internal/audit"
	"github.com/devekkx/module-dependency-visualizer/internal/config"
	"github.com/devekkx/module-dependency-visualizer/internal/schema"
)

func newAuditCmd(deps *Deps) *cobra.Command {
	var output string
	var format string
	var skipVuln bool
	var skipLicense bool
	var skipConflicts bool
	var timeout string

	cmd := &cobra.Command{
		Use:   "audit <path>",
		Short: "Run vulnerability, license, and conflict audit on dependencies",
		Long: `Audit scans dependencies for known security vulnerabilities (via OSV),
license information (via deps.dev), and version conflicts.

Requires network access for vulnerability and license checks.`,
		Args: cobra.ExactArgs(1),
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
				return fmt.Errorf("audit: %w", err)
			}

			a := audit.New(audit.Options{
				SkipVulnScan:  skipVuln,
				SkipLicense:   skipLicense,
				SkipConflicts: skipConflicts,
			})

			result, err := a.Run(ctx, g, proj.Language)
			if err != nil {
				return fmt.Errorf("audit: %w", err)
			}

			w, closer, err := openOutput(output)
			if err != nil {
				return err
			}
			defer closer()

			switch format {
			case "json":
				return writeAuditJSON(w, result)
			default:
				return writeAuditTable(w, result)
			}
		},
	}

	cmd.Flags().StringVarP(&output, "output", "o", "-", `Output path ("-" for stdout)`)
	cmd.Flags().StringVarP(&format, "format", "f", "table", "Output format: table or json")
	cmd.Flags().BoolVar(&skipVuln, "skip-vuln", false, "Skip vulnerability scanning")
	cmd.Flags().BoolVar(&skipLicense, "skip-license", false, "Skip license fetching")
	cmd.Flags().BoolVar(&skipConflicts, "skip-conflicts", false, "Skip version-conflict detection")
	cmd.Flags().StringVar(&timeout, "timeout", config.DefaultAuditTimeout.String(), "Timeout for network requests")

	return cmd
}

// AuditResultToDTO converts an audit.Result to a schema.AuditDTO for embedding
// in a JSON document.
func AuditResultToDTO(r *audit.Result) *schema.AuditDTO {
	dto := &schema.AuditDTO{
		ScannedAt: r.ScannedAt,
		Licenses:  r.Licenses,
	}
	for _, v := range r.Vulnerabilities {
		dto.Vulnerabilities = append(dto.Vulnerabilities, schema.VulnDTO{
			NodeID:   v.NodeID,
			ID:       v.ID,
			Summary:  v.Summary,
			Severity: v.Severity,
			FixedIn:  v.FixedIn,
			Link:     v.Link,
		})
	}
	for _, c := range r.Conflicts {
		dto.Conflicts = append(dto.Conflicts, schema.ConflictDTO{
			Module:   c.Module,
			Versions: c.Versions,
		})
	}
	return dto
}

func writeAuditJSON(w io.Writer, r *audit.Result) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(r)
}

func writeAuditTable(w io.Writer, r *audit.Result) error {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)

	// Vulnerabilities
	fmt.Fprintln(w, "=== Vulnerabilities ===")
	if len(r.Vulnerabilities) == 0 {
		fmt.Fprintln(w, "  None found")
	} else {
		fmt.Fprintln(tw, "MODULE\tID\tSEVERITY\tSUMMARY")
		for _, v := range r.Vulnerabilities {
			fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n", v.NodeID, v.ID, v.Severity, truncateStr(v.Summary, 60))
			if v.FixedIn != "" {
				fmt.Fprintf(tw, "\t\t→ fix: %s\t\n", v.FixedIn)
			}
		}
		_ = tw.Flush()
	}
	fmt.Fprintln(w)

	// Conflicts
	fmt.Fprintln(w, "=== Version Conflicts ===")
	if len(r.Conflicts) == 0 {
		fmt.Fprintln(w, "  None found")
	} else {
		fmt.Fprintln(tw, "MODULE\tVERSIONS")
		for _, c := range r.Conflicts {
			fmt.Fprintf(tw, "%s\t%s\n", c.Module, strings.Join(c.Versions, ", "))
		}
		_ = tw.Flush()
	}
	fmt.Fprintln(w)

	// Licenses
	fmt.Fprintln(w, "=== Licenses ===")
	if len(r.Licenses) == 0 {
		fmt.Fprintln(w, "  None found")
	} else {
		fmt.Fprintln(tw, "MODULE\tLICENSE")
		for nodeID, lic := range r.Licenses {
			fmt.Fprintf(tw, "%s\t%s\n", nodeID, lic)
		}
		_ = tw.Flush()
	}

	return nil
}

func truncateStr(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}
