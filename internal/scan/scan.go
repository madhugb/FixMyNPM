package scan

import (
	"os"
	"path/filepath"
	"strings"
)

// Result holds a discovered .npmrc path.
type Result struct {
	Path              string
	InsideNodeModules bool // true when found inside a node_modules tree
}

// WalkOptions controls scan behaviour.
type WalkOptions struct {
	// IncludeNodeModules makes the walker descend into node_modules directories.
	// Used for incident-response scans.
	IncludeNodeModules bool
}

// Walk finds all .npmrc files under root. If root is empty it defaults to the
// user's home directory.
func Walk(root string, opts WalkOptions) ([]Result, error) {
	if root == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		root = home
	}

	var results []Result

	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			if os.IsPermission(err) {
				return filepath.SkipDir
			}
			return err
		}

		if d.IsDir() {
			// skip hidden dirs (e.g. .git, .cache) but not the root itself
			if path != root && len(d.Name()) > 1 && d.Name()[0] == '.' {
				return filepath.SkipDir
			}
			// skip node_modules unless incident mode is on
			if d.Name() == "node_modules" && !opts.IncludeNodeModules {
				return filepath.SkipDir
			}
			return nil
		}

		if d.Name() == ".npmrc" {
			insideNM := strings.Contains(path, string(filepath.Separator)+"node_modules"+string(filepath.Separator))
			results = append(results, Result{
				Path:              path,
				InsideNodeModules: insideNM,
			})
		}

		return nil
	})

	return results, err
}
