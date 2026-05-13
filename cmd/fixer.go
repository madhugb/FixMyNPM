package cmd

import (
	"fmt"

	"github.com/madhugb/fixmynpm/internal/audit"
	"github.com/madhugb/fixmynpm/internal/fixer"
	"github.com/madhugb/fixmynpm/internal/scan"
	"github.com/spf13/cobra"
)

var (
	fixerRoot   string
	fixerDryRun bool
)

var fixerCmd = &cobra.Command{
	Use:   "fixer",
	Short: "Apply recommended fixes to .npmrc files",
	Example: `  fixmynpm fixer
  fixmynpm fixer --dry-run
  fixmynpm fixer --root /Users/me/projects`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Printf("Scanning for .npmrc files under: %s\n", rootLabel(fixerRoot))

		results, err := scan.Walk(fixerRoot, scan.WalkOptions{})
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

		reports, err := audit.Run(paths)
		if err != nil {
			return err
		}

		var allIssues []interface{}
		_ = allIssues

		totalFixable := 0
		for _, rep := range reports {
			for _, iss := range rep.Issues {
				if iss.Fix != "" {
					totalFixable++
				}
			}
		}

		if totalFixable == 0 {
			fmt.Println("No fixable issues found.")
			return nil
		}

		if fixerDryRun {
			fmt.Printf("Dry-run mode — %d fix(es) would be applied:\n\n", totalFixable)
		} else {
			fmt.Printf("Applying %d fix(es)...\n\n", totalFixable)
		}

		total := 0
		for _, rep := range reports {
			n, err := fixer.Apply(rep.Issues, fixerDryRun)
			if err != nil {
				return err
			}
			total += n
		}

		if fixerDryRun {
			fmt.Printf("\nDry-run complete. %d fix(es) would have been applied.\n", total)
		} else {
			fmt.Printf("\nDone. %d fix(es) applied.\n", total)
		}
		return nil
	},
}

func init() {
	fixerCmd.Flags().StringVar(&fixerRoot, "root", "", "Root directory to scan (default: home directory)")
	fixerCmd.Flags().BoolVar(&fixerDryRun, "dry-run", false, "Preview fixes without writing any files")
}
