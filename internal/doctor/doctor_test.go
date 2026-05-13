package doctor

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/madhugb/fixmynpm/internal/npmrc"
)

// parse builds an npmrc.File from a raw string, placed at path.
func parse(t *testing.T, path, content string) *npmrc.File {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	f, err := npmrc.Parse(path)
	if err != nil {
		t.Fatal(err)
	}
	return f
}

// hasRule returns true if any issue in the slice has the given rule ID.
func hasRule(issues []Issue, rule string) bool {
	for _, i := range issues {
		if i.Rule == rule {
			return true
		}
	}
	return false
}

func TestRegistryInsecureHTTP(t *testing.T) {
	dir := t.TempDir()
	f := parse(t, filepath.Join(dir, ".npmrc"), "registry=http://registry.npmjs.org/\n")
	issues := Check(f)
	if !hasRule(issues, "registry-insecure-http") {
		t.Error("expected registry-insecure-http")
	}
	// fix should upgrade to https
	for _, i := range issues {
		if i.Rule == "registry-insecure-http" && i.Fix == "" {
			t.Error("expected a Fix suggestion")
		}
	}
}

func TestRegistryInvalidURL(t *testing.T) {
	dir := t.TempDir()
	f := parse(t, filepath.Join(dir, ".npmrc"), "registry=not-a-url\n")
	issues := Check(f)
	if !hasRule(issues, "registry-invalid-url") {
		t.Error("expected registry-invalid-url")
	}
}

func TestRegistryHTTPS_Clean(t *testing.T) {
	dir := t.TempDir()
	f := parse(t, filepath.Join(dir, ".npmrc"), "registry=https://registry.npmjs.org/\n")
	issues := Check(f)
	if hasRule(issues, "registry-insecure-http") || hasRule(issues, "registry-invalid-url") {
		t.Error("https registry should not trigger registry rules")
	}
}

func TestStrictSSLDisabled(t *testing.T) {
	dir := t.TempDir()
	f := parse(t, filepath.Join(dir, ".npmrc"), "strict-ssl=false\n")
	issues := Check(f)
	if !hasRule(issues, "strict-ssl-disabled") {
		t.Error("expected strict-ssl-disabled")
	}
}

func TestStrictSSLEnabled_Clean(t *testing.T) {
	dir := t.TempDir()
	f := parse(t, filepath.Join(dir, ".npmrc"), "strict-ssl=true\n")
	issues := Check(f)
	if hasRule(issues, "strict-ssl-disabled") {
		t.Error("strict-ssl=true should not trigger rule")
	}
}

func TestIgnoreScriptsDisabled(t *testing.T) {
	dir := t.TempDir()
	f := parse(t, filepath.Join(dir, ".npmrc"), "ignore-scripts=false\n")
	issues := Check(f)
	if !hasRule(issues, "ignore-scripts-disabled") {
		t.Error("expected ignore-scripts-disabled")
	}
}

func TestIgnoreScriptsMissing(t *testing.T) {
	dir := t.TempDir()
	f := parse(t, filepath.Join(dir, ".npmrc"), "registry=https://registry.npmjs.org/\n")
	issues := Check(f)
	if !hasRule(issues, "ignore-scripts-disabled") {
		t.Error("expected ignore-scripts-disabled when key is absent")
	}
}

func TestIgnoreScriptsEnabled_Clean(t *testing.T) {
	dir := t.TempDir()
	f := parse(t, filepath.Join(dir, ".npmrc"), "ignore-scripts=true\n")
	issues := Check(f)
	if hasRule(issues, "ignore-scripts-disabled") {
		t.Error("ignore-scripts=true should not trigger rule")
	}
}

func TestAllowGitUnrestricted(t *testing.T) {
	dir := t.TempDir()
	f := parse(t, filepath.Join(dir, ".npmrc"), "allow-git=all\n")
	issues := Check(f)
	if !hasRule(issues, "allow-git-unrestricted") {
		t.Error("expected allow-git-unrestricted")
	}
}

func TestAllowGitMissing(t *testing.T) {
	dir := t.TempDir()
	f := parse(t, filepath.Join(dir, ".npmrc"), "registry=https://registry.npmjs.org/\n")
	issues := Check(f)
	if !hasRule(issues, "allow-git-unrestricted") {
		t.Error("expected allow-git-unrestricted when key is absent")
	}
}

func TestAllowGitNone_Clean(t *testing.T) {
	dir := t.TempDir()
	f := parse(t, filepath.Join(dir, ".npmrc"), "allow-git=none\n")
	issues := Check(f)
	if hasRule(issues, "allow-git-unrestricted") {
		t.Error("allow-git=none should not trigger rule")
	}
}

func TestMinReleaseAgeMissing(t *testing.T) {
	dir := t.TempDir()
	f := parse(t, filepath.Join(dir, ".npmrc"), "registry=https://registry.npmjs.org/\n")
	issues := Check(f)
	if !hasRule(issues, "min-release-age-missing") {
		t.Error("expected min-release-age-missing")
	}
}

func TestMinReleaseAgeTooLow(t *testing.T) {
	dir := t.TempDir()
	f := parse(t, filepath.Join(dir, ".npmrc"), "min-release-age=0\n")
	issues := Check(f)
	if !hasRule(issues, "min-release-age-too-low") {
		t.Error("expected min-release-age-too-low")
	}
}

func TestMinReleaseAgeSet_Clean(t *testing.T) {
	dir := t.TempDir()
	f := parse(t, filepath.Join(dir, ".npmrc"), "min-release-age=3\n")
	issues := Check(f)
	if hasRule(issues, "min-release-age-missing") || hasRule(issues, "min-release-age-too-low") {
		t.Error("min-release-age=3 should not trigger rule")
	}
}

func TestAuthTokenEmpty(t *testing.T) {
	dir := t.TempDir()
	f := parse(t, filepath.Join(dir, ".npmrc"), "_authToken=\n")
	issues := Check(f)
	if !hasRule(issues, "auth-token-empty") {
		t.Error("expected auth-token-empty")
	}
}

func TestAuthTokenInProjectNpmrc(t *testing.T) {
	// project-level = not in home dir
	dir := t.TempDir()
	path := filepath.Join(dir, "project", ".npmrc")
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	f := parse(t, path, "_authToken=supersecret\n")
	issues := Check(f)
	if !hasRule(issues, "auth-token-in-project-npmrc") {
		t.Error("expected auth-token-in-project-npmrc")
	}
}

func TestSaveExactDisabled(t *testing.T) {
	dir := t.TempDir()
	f := parse(t, filepath.Join(dir, ".npmrc"), "save-exact=false\n")
	issues := Check(f)
	if !hasRule(issues, "save-exact-disabled") {
		t.Error("expected save-exact-disabled")
	}
}

func TestSaveExactEnabled_Clean(t *testing.T) {
	dir := t.TempDir()
	f := parse(t, filepath.Join(dir, ".npmrc"), "save-exact=true\n")
	issues := Check(f)
	if hasRule(issues, "save-exact-disabled") {
		t.Error("save-exact=true should not trigger rule")
	}
}

func TestScopeMissingRegistry(t *testing.T) {
	dir := t.TempDir()
	f := parse(t, filepath.Join(dir, ".npmrc"), "@myorg:_authToken=abc123\n")
	issues := Check(f)
	if !hasRule(issues, "scope-missing-registry") {
		t.Error("expected scope-missing-registry")
	}
}

func TestScopeWithRegistry_Clean(t *testing.T) {
	dir := t.TempDir()
	f := parse(t, filepath.Join(dir, ".npmrc"), "@myorg:registry=https://npm.myorg.com/\n@myorg:_authToken=abc123\n")
	issues := Check(f)
	if hasRule(issues, "scope-missing-registry") {
		t.Error("scope with registry mapping should not trigger rule")
	}
}

func TestSeverityValues(t *testing.T) {
	dir := t.TempDir()
	f := parse(t, filepath.Join(dir, ".npmrc"), "strict-ssl=false\nignore-scripts=false\nmin-release-age=0\n")
	issues := Check(f)
	for _, i := range issues {
		switch i.Severity {
		case "error", "warning", "info":
		default:
			t.Errorf("issue %q has invalid severity %q", i.Rule, i.Severity)
		}
	}
}
