package npmrc

import (
	"os"
	"path/filepath"
	"testing"
)

func writeTemp(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, ".npmrc")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestParse_KeyValue(t *testing.T) {
	path := writeTemp(t, "registry=https://registry.npmjs.org/\nstrict-ssl=true\n")
	f, err := Parse(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(f.Entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(f.Entries))
	}
	if f.Entries[0].Key != "registry" || f.Entries[0].Value != "https://registry.npmjs.org/" {
		t.Errorf("unexpected entry[0]: %+v", f.Entries[0])
	}
	if f.Entries[1].Key != "strict-ssl" || f.Entries[1].Value != "true" {
		t.Errorf("unexpected entry[1]: %+v", f.Entries[1])
	}
}

func TestParse_SkipsCommentsAndBlanks(t *testing.T) {
	path := writeTemp(t, "# comment\n; also comment\n\nregistry=https://registry.npmjs.org/\n")
	f, err := Parse(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(f.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(f.Entries))
	}
}

func TestParse_SkipsInvalidLines(t *testing.T) {
	path := writeTemp(t, "not-a-key-value\nregistry=https://registry.npmjs.org/\n")
	f, err := Parse(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(f.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(f.Entries))
	}
}

func TestParse_ValueWithEquals(t *testing.T) {
	// values that themselves contain = should be preserved
	path := writeTemp(t, "//registry.npmjs.org/:_authToken=abc=def==\n")
	f, err := Parse(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(f.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(f.Entries))
	}
	if f.Entries[0].Value != "abc=def==" {
		t.Errorf("unexpected value: %q", f.Entries[0].Value)
	}
}

func TestParse_LineNumbers(t *testing.T) {
	path := writeTemp(t, "# comment\nregistry=https://registry.npmjs.org/\nstrict-ssl=true\n")
	f, err := Parse(path)
	if err != nil {
		t.Fatal(err)
	}
	if f.Entries[0].Line != 2 {
		t.Errorf("expected line 2, got %d", f.Entries[0].Line)
	}
	if f.Entries[1].Line != 3 {
		t.Errorf("expected line 3, got %d", f.Entries[1].Line)
	}
}

func TestParse_MissingFile(t *testing.T) {
	_, err := Parse("/nonexistent/.npmrc")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestGet(t *testing.T) {
	path := writeTemp(t, "registry=https://registry.npmjs.org/\n")
	f, _ := Parse(path)

	val, ok := f.Get("registry")
	if !ok || val != "https://registry.npmjs.org/" {
		t.Errorf("Get(registry) = %q, %v", val, ok)
	}

	_, ok = f.Get("nonexistent")
	if ok {
		t.Error("Get(nonexistent) should return false")
	}
}
