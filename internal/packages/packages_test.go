package packages

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// makePackage creates a fake node_modules/<name>/package.json with the given version.
func makePackage(t *testing.T, nmDir, name, version string) string {
	t.Helper()
	var pkgDir string
	if len(name) > 0 && name[0] == '@' {
		parts := splitScoped(name)
		pkgDir = filepath.Join(nmDir, parts[0], parts[1])
	} else {
		pkgDir = filepath.Join(nmDir, name)
	}
	if err := os.MkdirAll(pkgDir, 0755); err != nil {
		t.Fatal(err)
	}
	data, _ := json.Marshal(map[string]string{"name": name, "version": version})
	if err := os.WriteFile(filepath.Join(pkgDir, "package.json"), data, 0644); err != nil {
		t.Fatal(err)
	}
	return pkgDir
}

func splitScoped(name string) [2]string {
	for i := 1; i < len(name); i++ {
		if name[i] == '/' {
			return [2]string{name[:i], name[i+1:]}
		}
	}
	return [2]string{name, ""}
}

func makeLockfile(t *testing.T, dir string, versions map[string]string) {
	t.Helper()
	pkgs := map[string]map[string]string{}
	for name, ver := range versions {
		pkgs["node_modules/"+name] = map[string]string{"version": ver}
	}
	data, _ := json.Marshal(map[string]interface{}{"lockfileVersion": 2, "packages": pkgs})
	if err := os.WriteFile(filepath.Join(dir, "package-lock.json"), data, 0644); err != nil {
		t.Fatal(err)
	}
}

func TestFind_ExactMatch(t *testing.T) {
	root := t.TempDir()
	nm := filepath.Join(root, "node_modules")
	makePackage(t, nm, "lodash", "4.17.21")

	matches, err := Find(root, Options{PackageName: "lodash"})
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 1 {
		t.Fatalf("expected 1 match, got %d", len(matches))
	}
	if matches[0].Version != "4.17.21" {
		t.Errorf("unexpected version: %s", matches[0].Version)
	}
}

func TestFind_GlobMatch(t *testing.T) {
	root := t.TempDir()
	nm := filepath.Join(root, "node_modules")
	makePackage(t, nm, "lodash", "4.17.21")
	makePackage(t, nm, "lodash-es", "4.17.21")
	makePackage(t, nm, "react", "18.0.0")

	matches, err := Find(root, Options{PackageName: "lodash*"})
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 2 {
		t.Errorf("expected 2 glob matches, got %d", len(matches))
	}
}

func TestFind_ScopedExact(t *testing.T) {
	root := t.TempDir()
	nm := filepath.Join(root, "node_modules")
	makePackage(t, nm, "@babel/core", "7.22.0")

	matches, err := Find(root, Options{PackageName: "@babel/core"})
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 1 {
		t.Fatalf("expected 1 match, got %d", len(matches))
	}
}

func TestFind_ScopedGlob(t *testing.T) {
	root := t.TempDir()
	nm := filepath.Join(root, "node_modules")
	makePackage(t, nm, "@babel/core", "7.22.0")
	makePackage(t, nm, "@babel/parser", "7.21.0")
	makePackage(t, nm, "@types/node", "18.0.0")

	matches, err := Find(root, Options{PackageName: "@babel/*"})
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 2 {
		t.Errorf("expected 2 scoped matches, got %d", len(matches))
	}
}

func TestFind_VersionConstraint(t *testing.T) {
	root := t.TempDir()
	nm := filepath.Join(root, "node_modules")
	makePackage(t, nm, "lodash", "4.17.20")

	// should match
	matches, err := Find(root, Options{PackageName: "lodash", VersionConstraint: ">=4.17.0 <4.17.21"})
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 1 {
		t.Errorf("expected 1 match for vulnerable range, got %d", len(matches))
	}

	// should not match
	matches, err = Find(root, Options{PackageName: "lodash", VersionConstraint: ">=4.17.21"})
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 0 {
		t.Errorf("expected 0 matches outside range, got %d", len(matches))
	}
}

func TestFind_InvalidVersionConstraint(t *testing.T) {
	root := t.TempDir()
	_, err := Find(root, Options{PackageName: "lodash", VersionConstraint: "not-semver!!!"})
	if err == nil {
		t.Error("expected error for invalid version constraint")
	}
}

func TestFind_HasNpmrc(t *testing.T) {
	root := t.TempDir()
	nm := filepath.Join(root, "node_modules")
	pkgDir := makePackage(t, nm, "evil-pkg", "1.0.0")
	if err := os.WriteFile(filepath.Join(pkgDir, ".npmrc"), []byte("registry=http://evil.com/\n"), 0644); err != nil {
		t.Fatal(err)
	}

	matches, err := Find(root, Options{PackageName: "evil-pkg"})
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 1 || !matches[0].HasNpmrc {
		t.Error("expected HasNpmrc=true")
	}
}

func TestFind_IncludeNpmrcOnly(t *testing.T) {
	root := t.TempDir()
	nm := filepath.Join(root, "node_modules")
	makePackage(t, nm, "clean-pkg", "1.0.0")
	pkgDir := makePackage(t, nm, "evil-pkg", "1.0.0")
	os.WriteFile(filepath.Join(pkgDir, ".npmrc"), []byte("registry=http://evil.com/\n"), 0644)

	matches, err := Find(root, Options{PackageName: "*", IncludeNpmrcOnly: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 1 || matches[0].Name != "evil-pkg" {
		t.Errorf("expected only evil-pkg, got %+v", matches)
	}
}

func TestFind_LockfileVersion(t *testing.T) {
	root := t.TempDir()
	nm := filepath.Join(root, "node_modules")
	makePackage(t, nm, "lodash", "4.17.20")
	makeLockfile(t, root, map[string]string{"lodash": "4.17.20"})

	matches, err := Find(root, Options{PackageName: "lodash"})
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 1 {
		t.Fatalf("expected 1 match, got %d", len(matches))
	}
	if matches[0].LockVersion != "4.17.20" {
		t.Errorf("expected lock version 4.17.20, got %q", matches[0].LockVersion)
	}
}

func TestFind_NoMatch(t *testing.T) {
	root := t.TempDir()
	nm := filepath.Join(root, "node_modules")
	makePackage(t, nm, "react", "18.0.0")

	matches, err := Find(root, Options{PackageName: "lodash"})
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 0 {
		t.Errorf("expected 0 matches, got %d", len(matches))
	}
}

func TestFind_MultipleNodeModules(t *testing.T) {
	root := t.TempDir()
	makePackage(t, filepath.Join(root, "proj-a", "node_modules"), "lodash", "4.17.20")
	makePackage(t, filepath.Join(root, "proj-b", "node_modules"), "lodash", "4.17.21")

	matches, err := Find(root, Options{PackageName: "lodash"})
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 2 {
		t.Errorf("expected 2 matches across projects, got %d", len(matches))
	}
}
