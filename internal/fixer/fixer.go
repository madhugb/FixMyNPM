package fixer

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/madhugb/fixmynpm/internal/doctor"
)

// Apply writes fixes for the given issues into their respective files.
// It rewrites only the lines that have a non-empty Fix suggestion.
// Returns the number of fixes applied.
func Apply(issues []doctor.Issue, dryRun bool) (int, error) {
	// group issues by file
	byFile := map[string][]doctor.Issue{}
	for _, iss := range issues {
		if iss.Fix == "" {
			continue
		}
		byFile[iss.File] = append(byFile[iss.File], iss)
	}

	applied := 0
	for path, fileIssues := range byFile {
		n, err := applyToFile(path, fileIssues, dryRun)
		if err != nil {
			return applied, fmt.Errorf("fixing %s: %w", path, err)
		}
		applied += n
	}
	return applied, nil
}

func applyToFile(path string, issues []doctor.Issue, dryRun bool) (int, error) {
	// build a map of line number → replacement line
	replacements := map[int]string{}
	for _, iss := range issues {
		if iss.Line > 0 && iss.Fix != "" {
			replacements[iss.Line] = iss.Fix
		}
	}

	f, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	var lines []string
	scanner := bufio.NewScanner(f)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		if replacement, ok := replacements[lineNum]; ok {
			lines = append(lines, replacement)
		} else {
			lines = append(lines, scanner.Text())
		}
	}
	if err := scanner.Err(); err != nil {
		return 0, err
	}
	f.Close()

	count := len(replacements)
	if dryRun {
		fmt.Printf("[dry-run] would apply %d fix(es) to %s\n", count, path)
		for lineNum, fix := range replacements {
			fmt.Printf("  line %d → %s\n", lineNum, fix)
		}
		return count, nil
	}

	if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0644); err != nil {
		return 0, err
	}

	fmt.Printf("Applied %d fix(es) to %s\n", count, path)
	return count, nil
}
