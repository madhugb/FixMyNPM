package packages

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/Masterminds/semver/v3"
	"path"
)

// Match is a package found on disk that satisfied the search criteria.
type Match struct {
	Name        string
	Version     string // from package.json inside node_modules/<pkg>
	LockVersion string // resolved version from package-lock.json, if available
	Path        string // absolute path to the package dir
	HasNpmrc    bool   // true if an .npmrc exists inside the package dir
}

// Options controls what Find searches for.
type Options struct {
	// PackageName filters to packages with this exact name (required).
	PackageName string
	// VersionConstraint is an optional semver constraint string e.g. ">=1.0.0 <2.0.0".
	VersionConstraint string
	// IncludeNpmrcOnly only returns matches that have an .npmrc inside the package dir.
	IncludeNpmrcOnly bool
}

// Find walks root looking for node_modules directories, then searches for
// packages matching opts. It also consults any package-lock.json it finds
// alongside each node_modules dir.
func Find(root string, opts Options) ([]Match, error) {
	if root == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		root = home
	}

	var constraint *semver.Constraints
	if opts.VersionConstraint != "" {
		c, err := semver.NewConstraint(opts.VersionConstraint)
		if err != nil {
			return nil, err
		}
		constraint = c
	}

	var matches []Match

	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			if os.IsPermission(err) {
				return filepath.SkipDir
			}
			return err
		}

		if !d.IsDir() {
			return nil
		}

		// skip hidden dirs
		if len(d.Name()) > 1 && d.Name()[0] == '.' {
			return filepath.SkipDir
		}

		if d.Name() != "node_modules" {
			return nil
		}

		// found a node_modules — look up package-lock.json next to it
		lockVersions := readLockfile(filepath.Join(filepath.Dir(path), "package-lock.json"))

		// search inside node_modules for the target package
		found, err := findInNodeModules(path, opts.PackageName, constraint, lockVersions)
		if err != nil {
			return err
		}
		matches = append(matches, found...)

		// don't recurse into node_modules themselves (nested nm handled separately)
		return filepath.SkipDir
	})

	if err != nil {
		return nil, err
	}

	if opts.IncludeNpmrcOnly {
		filtered := matches[:0]
		for _, m := range matches {
			if m.HasNpmrc {
				filtered = append(filtered, m)
			}
		}
		matches = filtered
	}

	return matches, nil
}

// findInNodeModules glob-matches pkgName against entries in node_modules,
// including scoped packages under @scope/ subdirectories.
func findInNodeModules(nmPath, pattern string, constraint *semver.Constraints, lockVersions map[string]string) ([]Match, error) {
	// collect candidate (pkgName, pkgDir) pairs by glob-matching
	type candidate struct {
		name string
		dir  string
	}
	var candidates []candidate

	if strings.HasPrefix(pattern, "@") {
		// scoped: pattern is "@scope/name" or "@scope/*" or "@*/name"
		parts := strings.SplitN(pattern, "/", 2)
		scopeGlob := parts[0]
		nameGlob := "*"
		if len(parts) == 2 {
			nameGlob = parts[1]
		}

		// find matching scope dirs
		scopeDirs, _ := os.ReadDir(nmPath)
		for _, sd := range scopeDirs {
			if !sd.IsDir() || !strings.HasPrefix(sd.Name(), "@") {
				continue
			}
			matched, _ := path.Match(scopeGlob, sd.Name())
			if !matched {
				continue
			}
			scopePath := filepath.Join(nmPath, sd.Name())
			entries, _ := os.ReadDir(scopePath)
			for _, e := range entries {
				if !e.IsDir() {
					continue
				}
				if ok, _ := path.Match(nameGlob, e.Name()); ok {
					fullName := sd.Name() + "/" + e.Name()
					candidates = append(candidates, candidate{
						name: fullName,
						dir:  filepath.Join(scopePath, e.Name()),
					})
				}
			}
		}
	} else {
		// unscoped: glob against top-level node_modules entries
		entries, _ := os.ReadDir(nmPath)
		for _, e := range entries {
			if !e.IsDir() || strings.HasPrefix(e.Name(), "@") {
				continue
			}
			if ok, _ := path.Match(pattern, e.Name()); ok {
				candidates = append(candidates, candidate{
					name: e.Name(),
					dir:  filepath.Join(nmPath, e.Name()),
				})
			}
		}
	}

	var matches []Match
	for _, c := range candidates {
		version, err := readPackageVersion(c.dir)
		if err != nil || version == "" {
			continue
		}

		if constraint != nil {
			v, err := semver.NewVersion(version)
			if err != nil || !constraint.Check(v) {
				continue
			}
		}

		hasNpmrc := false
		if _, err := os.Stat(filepath.Join(c.dir, ".npmrc")); err == nil {
			hasNpmrc = true
		}

		matches = append(matches, Match{
			Name:        c.name,
			Version:     version,
			LockVersion: lockVersions[c.name],
			Path:        c.dir,
			HasNpmrc:    hasNpmrc,
		})
	}
	return matches, nil
}

// pkgJSON is a minimal package.json shape.
type pkgJSON struct {
	Version string `json:"version"`
}

func readPackageVersion(pkgDir string) (string, error) {
	data, err := os.ReadFile(filepath.Join(pkgDir, "package.json"))
	if err != nil {
		return "", err
	}
	var p pkgJSON
	if err := json.Unmarshal(data, &p); err != nil {
		return "", err
	}
	return p.Version, nil
}

// lockfile is a minimal package-lock.json shape (v2/v3).
type lockfile struct {
	Packages map[string]struct {
		Version string `json:"version"`
	} `json:"packages"`
	// v1 lockfile support
	Dependencies map[string]struct {
		Version string `json:"version"`
	} `json:"dependencies"`
}

// readLockfile parses a package-lock.json and returns a map of package name → resolved version.
func readLockfile(path string) map[string]string {
	out := map[string]string{}
	data, err := os.ReadFile(path)
	if err != nil {
		return out
	}
	var lf lockfile
	if err := json.Unmarshal(data, &lf); err != nil {
		return out
	}
	// v2/v3: packages keys are "node_modules/<name>" or "node_modules/@scope/name"
	for key, pkg := range lf.Packages {
		name := strings.TrimPrefix(key, "node_modules/")
		if name != "" && pkg.Version != "" {
			out[name] = pkg.Version
		}
	}
	// v1 fallback
	for name, dep := range lf.Dependencies {
		if _, exists := out[name]; !exists && dep.Version != "" {
			out[name] = dep.Version
		}
	}
	return out
}
