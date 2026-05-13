package cmd

import (
	"fmt"
	"os"

	"github.com/madhugb/fixmynpm/internal/doctor"
	"github.com/spf13/cobra"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check your global ~/.npmrc for configuration issues",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("Running doctor on ~/.npmrc ...")

		issues, err := doctor.Run()
		if err != nil {
			return err
		}

		if len(issues) == 0 {
			fmt.Println("✓ No issues found in ~/.npmrc")
			return nil
		}

		fmt.Printf("Found %d issue(s):\n\n", len(issues))
		printIssues(issues)

		os.Exit(1)
		return nil
	},
}
