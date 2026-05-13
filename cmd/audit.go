package cmd

import (
	"fmt"
	"os"

	"github.com/madhugb/fixmynpm/internal/audit"
	"github.com/madhugb/fixmynpm/internal/scan"
	"github.com/spf13/cobra"
)

var auditRoot string

var auditCmd = &cobra.Command{
	Use:   "audit",
	Short: "Scan and audit all .npmrc files, reporting all issues found",
	Example: `  fixmynpm audit
  fixmynpm audit --root /Users/me/projects`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Printf("Scanning for .npmrc files under: %s\n", rootLabel(auditRoot))

		results, err := scan.Walk(auditRoot, scan.WalkOptions{})
		if err != nil {
			return err
		}

		if len(results) == 0 {
			fmt.Println("No .npmrc files found.")
			return nil
		}

		paths := make([]string, len(results))
		for i, r := range results {
			paths[i] = r.Path
		}

		fmt.Printf("Auditing %d file(s)...\n\n", len(paths))
		reports, err := audit.Run(paths)
		if err != nil {
			return err
		}

		totalIssues := 0
		for _, rep := range reports {
			if len(rep.Issues) == 0 {
				fmt.Printf("✓ %s — no issues\n\n", rep.File)
				continue
			}
			fmt.Printf("✗ %s — %d issue(s):\n\n", rep.File, len(rep.Issues))
			printIssues(rep.Issues)
			totalIssues += len(rep.Issues)
		}

		if totalIssues > 0 {
			fmt.Printf("Total: %d issue(s) found. Run `fixmynpm fixer` to apply fixes.\n", totalIssues)
			os.Exit(1)
		}
		return nil
	},
}

func init() {
	auditCmd.Flags().StringVar(&auditRoot, "root", "", "Root directory to scan (default: home directory)")
}
