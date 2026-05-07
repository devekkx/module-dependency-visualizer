package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"module-dependency-visualizer/internal/config"
	"module-dependency-visualizer/internal/diff"
	"module-dependency-visualizer/internal/graph"
	"module-dependency-visualizer/internal/provider"
)

func newDiffCmd(deps *Deps) *cobra.Command {
	var fromRef string
	var toRef string
	var format string
	var output string
	var timeout string

	cmd := &cobra.Command{
		Use:   "diff [path]",
		Short: "Show dependency changes between two git refs",
		Long: `Compare the dependency graph at two git refs and report what was added,
removed, or version-bumped.

Defaults to comparing HEAD~1 (before) against the current working tree (after).

Examples:
  mdv diff .
  mdv diff . --from v1.0.0 --to v2.0.0
  mdv diff . --from main --to feature/new-dep
  mdv diff . --format json`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			rootPath := "."
			if len(args) == 1 {
				rootPath = args[0]
			}

			absRoot, err := filepath.Abs(rootPath)
			if err != nil {
				return fmt.Errorf("diff: resolve path: %w", err)
			}

			d, err := parseDuration(timeout)
			if err != nil {
				return err
			}

			ctx, cancel := context.WithTimeout(context.Background(), d)
			defer cancel()

			fromGraph, _, err := parseAtRef(ctx, deps.Providers, absRoot, fromRef)
			if err != nil {
				return fmt.Errorf("diff: parse from ref %q: %w", fromRef, err)
			}

			var toResult *diff.Result

			if toRef == "" {
				gTo, _, err2 := parseProject(ctx, deps, absRoot)
				if err2 != nil {
					return fmt.Errorf("diff: parse working tree: %w", err2)
				}
				toResult = diff.Diff(fromGraph, gTo)
			} else {
				gTo, _, err2 := parseAtRef(ctx, deps.Providers, absRoot, toRef)
				if err2 != nil {
					return fmt.Errorf("diff: parse to ref %q: %w", toRef, err2)
				}
				toResult = diff.Diff(fromGraph, gTo)
			}

			w, closer, err := openOutput(output)
			if err != nil {
				return err
			}
			defer closer()

			label := fmt.Sprintf("%s → %s", fromRef, orWorktree(toRef))
			switch format {
			case "json":
				return writeDiffJSON(w, toResult, label)
			case "markdown":
				return writeDiffMarkdown(w, toResult, label)
			default:
				return writeDiffTable(w, toResult, label)
			}
		},
	}

	cmd.Flags().StringVar(&fromRef, "from", "HEAD~1", "From git ref (commit, branch, or tag)")
	cmd.Flags().StringVar(&toRef, "to", "", "To git ref (default: working tree)")
	cmd.Flags().StringVarP(&format, "format", "f", "table", "Output format: table, json, markdown")
	cmd.Flags().StringVarP(&output, "output", "o", "-", `Output path ("-" for stdout)`)
	cmd.Flags().StringVar(&timeout, "timeout", config.DefaultTimeout.String(), "Timeout for dependency resolution")

	return cmd
}

// parseAtRef checks out a git ref into a temporary worktree, runs the
// provider on it, then removes the worktree.
func parseAtRef(ctx context.Context, providers *provider.Registry, repoPath, ref string) (
	*graph.Graph, provider.Project, error,
) {
	// Resolve the repo root (the path may be a subdirectory).
	repoRoot, err := gitRepoRoot(ctx, repoPath)
	if err != nil {
		return nil, provider.Project{}, fmt.Errorf("git root: %w", err)
	}

	tmpDir, err := os.MkdirTemp("", "mdv-diff-*")
	if err != nil {
		return nil, provider.Project{}, fmt.Errorf("mktemp: %w", err)
	}

	// git worktree add --detach <tmpDir> <ref>
	cmd := exec.CommandContext(ctx, "git", "-C", repoRoot, "worktree", "add", "--detach", tmpDir, ref)
	if out, err := cmd.CombinedOutput(); err != nil {
		_ = os.RemoveAll(tmpDir)
		return nil, provider.Project{}, fmt.Errorf("git worktree add: %w\n%s", err, out)
	}

	defer func() {
		// git worktree remove --force <tmpDir>
		_ = exec.Command("git", "-C", repoRoot, "worktree", "remove", "--force", tmpDir).Run()
		_ = os.RemoveAll(tmpDir)
	}()

	// Determine the sub-path relative to the repo root and reapply it.
	subPath, err := filepath.Rel(repoRoot, repoPath)
	if err != nil {
		subPath = "."
	}
	targetPath := filepath.Join(tmpDir, subPath)

	prov, err := providers.Detect(ctx, targetPath)
	if err != nil {
		return nil, provider.Project{}, fmt.Errorf("detect provider: %w", err)
	}

	g, proj, err := prov.Parse(ctx, targetPath, provider.ParseOptions{MaxDepth: -1})
	if err != nil {
		return nil, provider.Project{}, fmt.Errorf("parse: %w", err)
	}

	return g, proj, nil
}

// gitRepoRoot returns the top-level directory of the git repo containing path.
func gitRepoRoot(ctx context.Context, path string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "-C", path, "rev-parse", "--show-toplevel")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("not a git repository: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

func orWorktree(s string) string {
	if s == "" {
		return "working tree"
	}
	return s
}

// ── output formatters ─────────────────────────────────────────────────────────

func writeDiffTable(w io.Writer, r *diff.Result, label string) error {
	fmt.Fprintf(w, "Dependency diff: %s\n\n", label)

	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)

	writeSection := func(title string, changes []diff.Change) {
		fmt.Fprintf(w, "=== %s (%d) ===\n", title, len(changes))
		if len(changes) == 0 {
			fmt.Fprintln(w, "  (none)")
		} else {
			for _, c := range changes {
				indirect := ""
				if c.Indirect {
					indirect = " (indirect)"
				}
				switch c.Kind {
				case diff.ChangeAdded:
					fmt.Fprintf(tw, "  + %s\t%s%s\n", c.Name, c.ToVersion, indirect)
				case diff.ChangeRemoved:
					fmt.Fprintf(tw, "  - %s\t%s%s\n", c.Name, c.FromVersion, indirect)
				case diff.ChangeUpdated:
					fmt.Fprintf(tw, "  ~ %s\t%s → %s%s\n", c.Name, c.FromVersion, c.ToVersion, indirect)
				}
			}
			_ = tw.Flush()
		}
		fmt.Fprintln(w)
	}

	writeSection("ADDED", r.Added)
	writeSection("REMOVED", r.Removed)
	writeSection("UPDATED", r.Updated)
	fmt.Fprintf(w, "UNCHANGED: %d modules\n", r.Unchanged)
	return nil
}

func writeDiffMarkdown(w io.Writer, r *diff.Result, label string) error {
	fmt.Fprintf(w, "## Dependency Changes: %s\n\n", label)

	total := len(r.Added) + len(r.Removed) + len(r.Updated)
	if total == 0 {
		fmt.Fprintf(w, "No dependency changes.\n")
		return nil
	}

	if len(r.Added) > 0 {
		fmt.Fprintf(w, "### Added (%d)\n\n", len(r.Added))
		fmt.Fprintf(w, "| Module | Version |\n|--------|:-------:|\n")
		for _, c := range r.Added {
			fmt.Fprintf(w, "| `%s` | `%s` |\n", c.Name, c.ToVersion)
		}
		fmt.Fprintf(w, "\n")
	}

	if len(r.Removed) > 0 {
		fmt.Fprintf(w, "### Removed (%d)\n\n", len(r.Removed))
		fmt.Fprintf(w, "| Module | Version |\n|--------|:-------:|\n")
		for _, c := range r.Removed {
			fmt.Fprintf(w, "| `%s` | `%s` |\n", c.Name, c.FromVersion)
		}
		fmt.Fprintf(w, "\n")
	}

	if len(r.Updated) > 0 {
		fmt.Fprintf(w, "### Updated (%d)\n\n", len(r.Updated))
		fmt.Fprintf(w, "| Module | From | To |\n|--------|:----:|:--:|\n")
		for _, c := range r.Updated {
			fmt.Fprintf(w, "| `%s` | `%s` | `%s` |\n", c.Name, c.FromVersion, c.ToVersion)
		}
		fmt.Fprintf(w, "\n")
	}

	fmt.Fprintf(w, "_%d module%s unchanged._\n", r.Unchanged, suf(r.Unchanged))
	return nil
}

func writeDiffJSON(w io.Writer, r *diff.Result, label string) error {
	out := struct {
		Label     string       `json:"label"`
		Added     []diff.Change `json:"added"`
		Removed   []diff.Change `json:"removed"`
		Updated   []diff.Change `json:"updated"`
		Unchanged int          `json:"unchanged"`
	}{
		Label:     label,
		Added:     r.Added,
		Removed:   r.Removed,
		Updated:   r.Updated,
		Unchanged: r.Unchanged,
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}

func suf(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}
