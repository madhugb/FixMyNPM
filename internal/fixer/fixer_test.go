package fixer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/madhugb/fixmynpm/internal/doctor"
)

func writeNpmrc(t *testing.T, dir, content string) string {
	t.Helper()
	path := filepath.Join(dir, ".npmrc")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

func readNpmrc(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestApply_FixesLine(t *testing.T) {
	dir := t.TempDir()
	path := writeNpmrc(t, dir, "registry=http://registry.npmjs.org/\nstrict-ssl=true\n")

	issues := []doctor.Issue{
		{
			File: path,
			Line: 1,
			Key:  "registry",
			Fix:  "registry=https://registry.npmjs.org/",
		},
	}

	n, err := Apply(issues, false)
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Errorf("expected 1 fix applied, got %d", n)
	}

	content := readNpmrc(t, path)
	if !strings.Contains(content, "registry=https://registry.npmjs.org/") {
		t.Errorf("fix not applied, got: %s", content)
	}
	// other lines unchanged
	if !strings.Contains(content, "strict-ssl=true") {
		t.Errorf("other lines should be preserved, got: %s", content)
	}
}

func TestApply_DryRunDoesNotWrite(t *testing.T) {
	dir := t.TempDir()
	original := "registry=http://registry.npmjs.org/\n"
	path := writeNpmrc(t, dir, original)

	issues := []doctor.Issue{
		{
			File: path,
			Line: 1,
			Key:  "registry",
			Fix:  "registry=https://registry.npmjs.org/",
		},
	}

	n, err := Apply(issues, true)
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Errorf("expected 1 fix counted in dry-run, got %d", n)
	}

	// file must be unchanged
	content := readNpmrc(t, path)
	if content != original {
		t.Errorf("dry-run should not modify file, got: %s", content)
	}
}

func TestApply_SkipsIssuesWithNoFix(t *testing.T) {
	dir := t.TempDir()
	path := writeNpmrc(t, dir, "_authToken=secret\n")

	issues := []doctor.Issue{
		{
			File:    path,
			Line:    1,
			Key:     "_authToken",
			Message: "token in project npmrc",
			Fix:     "", // no auto-fix
		},
	}

	n, err := Apply(issues, false)
	if err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Errorf("expected 0 fixes (no Fix value), got %d", n)
	}
}

func TestApply_MultipleFixesSameFile(t *testing.T) {
	dir := t.TempDir()
	path := writeNpmrc(t, dir, "registry=http://registry.npmjs.org/\nstrict-ssl=false\n")

	issues := []doctor.Issue{
		{File: path, Line: 1, Key: "registry", Fix: "registry=https://registry.npmjs.org/"},
		{File: path, Line: 2, Key: "strict-ssl", Fix: "strict-ssl=true"},
	}

	n, err := Apply(issues, false)
	if err != nil {
		t.Fatal(err)
	}
	if n != 2 {
		t.Errorf("expected 2 fixes, got %d", n)
	}

	content := readNpmrc(t, path)
	if !strings.Contains(content, "registry=https://registry.npmjs.org/") {
		t.Error("registry fix not applied")
	}
	if !strings.Contains(content, "strict-ssl=true") {
		t.Error("strict-ssl fix not applied")
	}
}

func TestApply_MultipleFiles(t *testing.T) {
	dir := t.TempDir()
	pathA := writeNpmrc(t, dir, "strict-ssl=false\n")

	dirB := t.TempDir()
	pathB := writeNpmrc(t, dirB, "registry=http://registry.npmjs.org/\n")

	issues := []doctor.Issue{
		{File: pathA, Line: 1, Key: "strict-ssl", Fix: "strict-ssl=true"},
		{File: pathB, Line: 1, Key: "registry", Fix: "registry=https://registry.npmjs.org/"},
	}

	n, err := Apply(issues, false)
	if err != nil {
		t.Fatal(err)
	}
	if n != 2 {
		t.Errorf("expected 2 fixes across files, got %d", n)
	}
}

func TestApply_EmptyIssues(t *testing.T) {
	n, err := Apply(nil, false)
	if err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Errorf("expected 0, got %d", n)
	}
}
