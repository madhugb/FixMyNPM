package doctor

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/madhugb/fixmynpm/internal/npmrc"
)

// Issue describes a problem found in an .npmrc file.
type Issue struct {
	File     string
	Line     int
	Key      string
	Message  string
	Fix      string
	Severity string // "error" | "warning" | "info"
	Rule     string // short rule ID for reference
}

// Run checks the global ~/.npmrc and returns any issues found.
func Run() ([]Issue, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("could not determine home directory: %w", err)
	}

	path := filepath.Join(home, ".npmrc")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		fmt.Println("No ~/.npmrc found — nothing to check.")
		return nil, nil
	}

	f, err := npmrc.Parse(path)
	if err != nil {
		return nil, fmt.Errorf("failed to parse %s: %w", path, err)
	}

	return Check(f), nil
}

// Check audits any parsed npmrc file and returns issues.
func Check(f *npmrc.File) []Issue {
	var issues []Issue

	issues = append(issues, checkRegistry(f)...)
	issues = append(issues, checkStrictSSL(f)...)
	issues = append(issues, checkIgnoreScripts(f)...)
	issues = append(issues, checkAllowGit(f)...)
	issues = append(issues, checkMinReleaseAge(f)...)
	issues = append(issues, checkAuthTokens(f)...)
	issues = append(issues, checkSaveExact(f)...)
	issues = append(issues, checkScopedRegistries(f)...)

	return issues
}

// checkRegistry ensures the registry key uses https and is a valid URL.
func checkRegistry(f *npmrc.File) []Issue {
	var issues []Issue
	for _, e := range f.Entries {
		if e.Key != "registry" {
			continue
		}
		if !strings.HasPrefix(e.Value, "http://") && !strings.HasPrefix(e.Value, "https://") {
			issues = append(issues, Issue{
				File:     f.Path,
				Line:     e.Line,
				Key:      e.Key,
				Message:  fmt.Sprintf("registry %q is not a valid URL", e.Value),
				Fix:      "registry=https://registry.npmjs.org/",
				Severity: "error",
				Rule:     "registry-invalid-url",
			})
		} else if strings.HasPrefix(e.Value, "http://") {
			issues = append(issues, Issue{
				File:     f.Path,
				Line:     e.Line,
				Key:      e.Key,
				Message:  "registry uses insecure http — upgrade to https to prevent MITM attacks",
				Fix:      fmt.Sprintf("registry=%s", strings.Replace(e.Value, "http://", "https://", 1)),
				Severity: "error",
				Rule:     "registry-insecure-http",
			})
		}
	}
	return issues
}

// checkStrictSSL flags strict-ssl=false which disables TLS certificate validation.
func checkStrictSSL(f *npmrc.File) []Issue {
	var issues []Issue
	for _, e := range f.Entries {
		if e.Key == "strict-ssl" && e.Value == "false" {
			issues = append(issues, Issue{
				File:     f.Path,
				Line:     e.Line,
				Key:      e.Key,
				Message:  "strict-ssl=false disables TLS certificate verification — severe security risk",
				Fix:      "strict-ssl=true",
				Severity: "error",
				Rule:     "strict-ssl-disabled",
			})
		}
	}
	return issues
}

// checkIgnoreScripts flags when ignore-scripts is not explicitly set to true.
// Lifecycle scripts in dependencies have been used in supply chain attacks.
func checkIgnoreScripts(f *npmrc.File) []Issue {
	val, found := f.Get("ignore-scripts")
	if !found || val != "true" {
		line := 0
		if found {
			for _, e := range f.Entries {
				if e.Key == "ignore-scripts" {
					line = e.Line
					break
				}
			}
		}
		msg := "ignore-scripts is not enabled — post-install scripts in dependencies can execute arbitrary code"
		if found {
			msg = fmt.Sprintf("ignore-scripts=%s — should be true to block post-install script execution", val)
		}
		return []Issue{{
			File:     f.Path,
			Line:     line,
			Key:      "ignore-scripts",
			Message:  msg,
			Fix:      "ignore-scripts=true",
			Severity: "warning",
			Rule:     "ignore-scripts-disabled",
		}}
	}
	return nil
}

// checkAllowGit flags when allow-git is not set to "none".
// Git-based dependencies bypass registry security scanning.
func checkAllowGit(f *npmrc.File) []Issue {
	val, found := f.Get("allow-git")
	if !found || val != "none" {
		line := 0
		if found {
			for _, e := range f.Entries {
				if e.Key == "allow-git" {
					line = e.Line
					break
				}
			}
		}
		msg := "allow-git is not set — git-based dependencies bypass registry security scanning"
		if found {
			msg = fmt.Sprintf("allow-git=%s — set to 'none' to block git dependencies that bypass registry controls", val)
		}
		return []Issue{{
			File:     f.Path,
			Line:     line,
			Key:      "allow-git",
			Message:  msg,
			Fix:      "allow-git=none",
			Severity: "warning",
			Rule:     "allow-git-unrestricted",
		}}
	}
	return nil
}

// checkMinReleaseAge flags when min-release-age is not set or is too low.
// A cooldown period lets the community catch malicious new releases.
func checkMinReleaseAge(f *npmrc.File) []Issue {
	val, found := f.Get("min-release-age")
	if !found {
		return []Issue{{
			File:     f.Path,
			Line:     0,
			Key:      "min-release-age",
			Message:  "min-release-age is not set — newly published packages can be installed immediately, leaving no time for the community to catch malicious releases",
			Fix:      "min-release-age=3",
			Severity: "info",
			Rule:     "min-release-age-missing",
		}}
	}
	n, err := strconv.Atoi(val)
	if err != nil || n < 1 {
		line := 0
		for _, e := range f.Entries {
			if e.Key == "min-release-age" {
				line = e.Line
				break
			}
		}
		return []Issue{{
			File:     f.Path,
			Line:     line,
			Key:      "min-release-age",
			Message:  fmt.Sprintf("min-release-age=%s provides no cooldown — set to at least 3 (days)", val),
			Fix:      "min-release-age=3",
			Severity: "info",
			Rule:     "min-release-age-too-low",
		}}
	}
	return nil
}

// checkAuthTokens flags empty auth tokens and warns about tokens in project-level .npmrc.
func checkAuthTokens(f *npmrc.File) []Issue {
	var issues []Issue
	home, _ := os.UserHomeDir()
	globalNpmrc := filepath.Join(home, ".npmrc")
	isProjectLevel := f.Path != globalNpmrc

	for _, e := range f.Entries {
		isToken := strings.HasSuffix(e.Key, ":_authToken") || e.Key == "_authToken" ||
			strings.HasSuffix(e.Key, ":_auth") || e.Key == "_auth"

		if !isToken {
			continue
		}

		if strings.TrimSpace(e.Value) == "" {
			issues = append(issues, Issue{
				File:     f.Path,
				Line:     e.Line,
				Key:      e.Key,
				Message:  "auth token is empty",
				Fix:      "",
				Severity: "error",
				Rule:     "auth-token-empty",
			})
			continue
		}

		// Warn if a real token is committed in a project-level .npmrc
		if isProjectLevel {
			issues = append(issues, Issue{
				File:     f.Path,
				Line:     e.Line,
				Key:      e.Key,
				Message:  "auth token found in project-level .npmrc — tokens should live in ~/.npmrc or be injected via NPM_TOKEN env var to avoid accidental commits",
				Fix:      "",
				Severity: "error",
				Rule:     "auth-token-in-project-npmrc",
			})
		}
	}
	return issues
}

// checkSaveExact flags save-exact=false which allows floating semver ranges.
func checkSaveExact(f *npmrc.File) []Issue {
	for _, e := range f.Entries {
		if e.Key == "save-exact" && e.Value == "false" {
			return []Issue{{
				File:     f.Path,
				Line:     e.Line,
				Key:      e.Key,
				Message:  "save-exact=false allows floating semver versions — pin exact versions to avoid unexpected upgrades",
				Fix:      "save-exact=true",
				Severity: "info",
				Rule:     "save-exact-disabled",
			}}
		}
	}
	return nil
}

// checkScopedRegistries detects @scope entries without a matching @scope:registry,
// which can leave scoped packages resolving from the public registry (dependency confusion).
func checkScopedRegistries(f *npmrc.File) []Issue {
	var issues []Issue

	// collect all @scope:registry mappings
	scopeRegistries := map[string]bool{}
	for _, e := range f.Entries {
		if strings.Contains(e.Key, ":registry") {
			scope := strings.TrimSuffix(e.Key, ":registry")
			scopeRegistries[scope] = true
		}
	}

	// look for scoped package references that have no registry mapping
	seen := map[string]bool{}
	for _, e := range f.Entries {
		if !strings.HasPrefix(e.Key, "@") {
			continue
		}
		// e.g. @myorg:always-auth or @myorg:_authToken — extract @myorg
		parts := strings.SplitN(e.Key, ":", 2)
		if len(parts) < 2 {
			continue
		}
		scope := parts[0]
		if seen[scope] || scopeRegistries[scope] {
			continue
		}
		seen[scope] = true
		issues = append(issues, Issue{
			File:     f.Path,
			Line:     e.Line,
			Key:      e.Key,
			Message:  fmt.Sprintf("scope %q has config entries but no registry mapping — add %s:registry=<url> to prevent dependency confusion attacks", scope, scope),
			Fix:      fmt.Sprintf("%s:registry=https://registry.npmjs.org/", scope),
			Severity: "warning",
			Rule:     "scope-missing-registry",
		})
	}
	return issues
}
