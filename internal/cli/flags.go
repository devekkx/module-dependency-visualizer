package cli

import (
	"fmt"
	"regexp"
	"time"

	"github.com/spf13/cobra"
)

// analyseFlags holds the flags shared by the analyse and export subcommands.
type analyseFlags struct {
	Output     string
	Depth      int
	Include    []string
	Exclude    []string
	NoIndirect bool
	Timeout    time.Duration
}

func addAnalyseFlags(cmd *cobra.Command, f *analyseFlags) {
	cmd.Flags().StringVarP(&f.Output, "output", "o", "-", `Output path ("-" for stdout)`)
	cmd.Flags().IntVarP(&f.Depth, "depth", "d", -1, "Max dependency depth (-1 = unlimited)")
	cmd.Flags().StringArrayVar(&f.Include, "include", nil, "Include only modules matching regex (repeatable)")
	cmd.Flags().StringArrayVar(&f.Exclude, "exclude", nil, "Exclude modules matching regex (repeatable)")
	cmd.Flags().BoolVar(&f.NoIndirect, "no-indirect", false, "Omit indirect dependencies")
}

// compilePatterns compiles a slice of regex strings.
func compilePatterns(patterns []string) ([]*regexp.Regexp, error) {
	compiled := make([]*regexp.Regexp, len(patterns))
	for i, p := range patterns {
		r, err := regexp.Compile(p)
		if err != nil {
			return nil, fmt.Errorf("invalid regex %q: %w", p, err)
		}
		compiled[i] = r
	}
	return compiled, nil
}

// classifyError maps an error to an exit code based on its type.
func classifyError(err error) int {
	if err == nil {
		return exitSuccess
	}
	return exitUser
}

const (
	exitSuccess  = 0
	exitUser     = 1
	exitProvider = 2
	exitInternal = 3
)
