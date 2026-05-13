package audit

import (
	"github.com/madhugb/fixmynpm/internal/doctor"
	"github.com/madhugb/fixmynpm/internal/npmrc"
)

// Report is the result of auditing one or more .npmrc files.
type Report struct {
	File   string
	Issues []doctor.Issue
}

// Run parses and checks each provided .npmrc path.
func Run(paths []string) ([]Report, error) {
	var reports []Report

	for _, p := range paths {
		f, err := npmrc.Parse(p)
		if err != nil {
			// record as a parse error issue
			reports = append(reports, Report{
				File: p,
				Issues: []doctor.Issue{{
					File:    p,
					Message: "failed to parse file: " + err.Error(),
				}},
			})
			continue
		}

		issues := doctor.Check(f)
		reports = append(reports, Report{
			File:   p,
			Issues: issues,
		})
	}

	return reports, nil
}
