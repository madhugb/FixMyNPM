package cmd

import (
	"fmt"
	"os"

	"github.com/madhugb/fixmynpm/internal/packages"
	"github.com/madhugb/fixmynpm/internal/scan"
	"github.com/spf13/cobra"
)

var (
	scanRoot        string
	scanPackage     string
	scanVersion     string
	scanNpmrcOnly   bool
	scanIncident    bool
)

var scanCmd = &cobra.Command{
	Use:   "scan",
	Short: "Find .npmrc files or search for installed packages across a directory tree",
	Example: `  # Find all .npmrc files (default)
  fixmynpm scan
  fixmynpm scan --root /Users/me/projects

  # Find a specific package across all node_modules
  fixmynpm scan --package lodash
  fixmynpm scan --package lodash --version ">=4.17.0 <4.17.21"

  # Incident response: find packages that have a bundled .npmrc
  fixmynpm scan --package lodash --npmrc
  fixmynpm scan --incident --package ua-parser-js

  # Full incident sweep: all packages with a bundled .npmrc under a path
  fixmynpm scan --incident --root /Users/me/projects`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// --incident implies --npmrc and includes node_modules in .npmrc walk
		if scanIncident {
			scanNpmrcOnly = true
		}

		if scanPackage != "" {
			return runPackageScan(cmd)
		}
		return runNpmrcScan(cmd)
	},
}

// runNpmrcScan is the original behaviour: find .npmrc files across the tree.
func runNpmrcScan(cmd *cobra.Command) error {
	label := rootLabel(scanRoot)
	if scanIncident {
		fmt.Printf("Incident scan — searching for .npmrc files (including node_modules) under: %s\n\n", label)
	} else {
		fmt.Printf("Scanning for .npmrc files under: %s\n\n", label)
	}

	results, err := scan.Walk(scanRoot, scan.WalkOptions{IncludeNodeModules: scanIncident})
	if err != nil {
		return err
	}

	if len(results) == 0 {
		fmt.Println("No .npmrc files found.")
		return nil
	}

	fmt.Printf("Found %d .npmrc file(s):\n", len(results))
	for _, r := range results {
		flag := ""
		if r.InsideNodeModules {
			flag = "  ⚠  inside node_modules"
		}
		fmt.Printf("  %s%s\n", r.Path, flag)
	}
	return nil
}

// runPackageScan searches node_modules for a specific package.
func runPackageScan(cmd *cobra.Command) error {
	label := rootLabel(scanRoot)
	versionLabel := ""
	if scanVersion != "" {
		versionLabel = fmt.Sprintf(" @ %s", scanVersion)
	}
	fmt.Printf("Searching for package %q%s under: %s\n\n", scanPackage, versionLabel, label)

	opts := packages.Options{
		PackageName:       scanPackage,
		VersionConstraint: scanVersion,
		IncludeNpmrcOnly:  scanNpmrcOnly,
	}

	matches, err := packages.Find(scanRoot, opts)
	if err != nil {
		return fmt.Errorf("scan failed: %w", err)
	}

	if len(matches) == 0 {
		fmt.Printf("No installations of %q found", scanPackage)
		if scanVersion != "" {
			fmt.Printf(" matching version %q", scanVersion)
		}
		if scanNpmrcOnly {
			fmt.Print(" with a bundled .npmrc")
		}
		fmt.Println(".")
		return nil
	}

	fmt.Printf("Found %d installation(s):\n\n", len(matches))
	hasIssues := false
	for _, m := range matches {
		lockInfo := ""
		if m.LockVersion != "" && m.LockVersion != m.Version {
			lockInfo = fmt.Sprintf(" (lock: %s)", m.LockVersion)
		} else if m.LockVersion != "" {
			lockInfo = fmt.Sprintf(" (locked)")
		}

		npmrcWarn := ""
		if m.HasNpmrc {
			npmrcWarn = "  ⚠  .npmrc found inside package — potential supply chain risk"
			hasIssues = true
		}

		fmt.Printf("  %s@%s%s\n", m.Name, m.Version, lockInfo)
		fmt.Printf("  Path: %s\n", m.Path)
		if npmrcWarn != "" {
			fmt.Println(npmrcWarn)
		}
		fmt.Println()
	}

	if hasIssues {
		fmt.Println("Packages with bundled .npmrc files may redirect registry or inject auth tokens.")
		fmt.Println("Run `fixmynpm audit` on those paths for a full analysis.")
		os.Exit(1)
	}

	return nil
}

func init() {
	scanCmd.Flags().StringVar(&scanRoot, "root", "", "Root directory to scan (default: home directory)")
	scanCmd.Flags().StringVar(&scanPackage, "package", "", "Package name to search for in node_modules (e.g. lodash, @scope/pkg)")
	scanCmd.Flags().StringVar(&scanVersion, "version", "", "Semver constraint to filter matched packages (e.g. \">=1.0.0 <2.0.0\"), requires --package")
	scanCmd.Flags().BoolVar(&scanNpmrcOnly, "npmrc", false, "Only show packages/paths that have a bundled .npmrc file")
	scanCmd.Flags().BoolVar(&scanIncident, "incident", false, "Incident response mode: include node_modules in .npmrc walk and flag bundled .npmrc files")

	scanCmd.MarkFlagsMutuallyExclusive("npmrc", "incident")
}
