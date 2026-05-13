package scan

import (
	"os"
	"path/filepath"
	"testing"
)

func makeFile(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("registry=https://registry.npmjs.org/\n"), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestWalk_FindsNpmrcFiles(t *testing.T) {
	root := t.TempDir()
	makeFile(t, filepath.Join(root, ".npmrc"))
	makeFile(t, filepath.Join(root, "proj-a", ".npmrc"))
	makeFile(t, filepath.Join(root, "proj-b", ".npmrc"))

	results, err := Walk(root, WalkOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 3 {
		t.Errorf("expected 3 results, got %d", len(results))
	}
}

func TestWalk_SkipsNodeModulesByDefault(t *testing.T) {
	root := t.TempDir()
	makeFile(t, filepath.Join(root, ".npmrc"))
	makeFile(t, filepath.Join(root, "node_modules", "evil-pkg", ".npmrc"))

	results, err := Walk(root, WalkOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Errorf("expected 1 result (node_modules skipped), got %d", len(results))
	}
}

func TestWalk_IncludesNodeModulesInIncidentMode(t *testing.T) {
	root := t.TempDir()
	makeFile(t, filepath.Join(root, ".npmrc"))
	makeFile(t, filepath.Join(root, "node_modules", "evil-pkg", ".npmrc"))

	results, err := Walk(root, WalkOptions{IncludeNodeModules: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 {
		t.Errorf("expected 2 results, got %d", len(results))
	}
}

func TestWalk_InsideNodeModulesFlag(t *testing.T) {
	root := t.TempDir()
	makeFile(t, filepath.Join(root, ".npmrc"))
	makeFile(t, filepath.Join(root, "node_modules", "evil-pkg", ".npmrc"))

	results, err := Walk(root, WalkOptions{IncludeNodeModules: true})
	if err != nil {
		t.Fatal(err)
	}

	counts := map[bool]int{}
	for _, r := range results {
		counts[r.InsideNodeModules]++
	}
	if counts[false] != 1 {
		t.Errorf("expected 1 non-node_modules result, got %d", counts[false])
	}
	if counts[true] != 1 {
		t.Errorf("expected 1 node_modules result, got %d", counts[true])
	}
}

func TestWalk_SkipsHiddenDirs(t *testing.T) {
	root := t.TempDir()
	makeFile(t, filepath.Join(root, ".npmrc"))
	makeFile(t, filepath.Join(root, ".git", ".npmrc"))
	makeFile(t, filepath.Join(root, ".cache", "something", ".npmrc"))

	results, err := Walk(root, WalkOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Errorf("expected 1 result (hidden dirs skipped), got %d", len(results))
	}
}

func TestWalk_EmptyDir(t *testing.T) {
	root := t.TempDir()
	results, err := Walk(root, WalkOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}

func TestWalk_NestedProjects(t *testing.T) {
	root := t.TempDir()
	makeFile(t, filepath.Join(root, "a", "b", "c", ".npmrc"))
	makeFile(t, filepath.Join(root, "x", "y", ".npmrc"))

	results, err := Walk(root, WalkOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 {
		t.Errorf("expected 2 results, got %d", len(results))
	}
}
