package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/madhugb/fixmynpm/internal/doctor"
	"github.com/spf13/cobra"
)

// version is set at build time via ldflags.
var version = "dev"

var rootCmd = &cobra.Command{
	Use:     "fixmynpm",
	Version: version,
	Short:   "Diagnose and fix .npmrc configuration issues across your projects",
	Long: `fixmynpm helps you find and fix npm configuration problems.

Commands:
  doctor  — check your global ~/.npmrc for issues
  scan    — find all .npmrc files across a directory tree
  audit   — audit discovered .npmrc files and report problems
  fixer   — apply recommended fixes to .npmrc files`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// rootLabel returns a display label for the root directory.
func rootLabel(r string) string {
	if r == "" {
		return "~ (home directory)"
	}
	return r
}

// severityIcon returns a short prefix for display.
func severityIcon(s string) string {
	switch strings.ToLower(s) {
	case "error":
		return "[error]"
	case "warning":
		return "[warn] "
	default:
		return "[info] "
	}
}

// printIssues renders a list of issues to stdout.
func printIssues(issues []doctor.Issue) {
	for _, iss := range issues {
		loc := ""
		if iss.Line > 0 {
			loc = fmt.Sprintf(" line %d", iss.Line)
		}
		fmt.Printf("  %s  rule=%-35s key=%s%s\n", severityIcon(iss.Severity), iss.Rule, iss.Key, loc)
		fmt.Printf("           %s\n", iss.Message)
		if iss.Fix != "" {
			fmt.Printf("           Fix: %s\n", iss.Fix)
		}
		fmt.Println()
	}
}

func init() {
	rootCmd.AddCommand(doctorCmd)
	rootCmd.AddCommand(scanCmd)
	rootCmd.AddCommand(auditCmd)
	rootCmd.AddCommand(fixerCmd)
}
