package npmrc

import (
	"bufio"
	"os"
	"strings"
)

// Entry represents a single key=value line in an .npmrc file.
type Entry struct {
	Key   string
	Value string
	Line  int
}

// File holds the parsed contents of an .npmrc file.
type File struct {
	Path    string
	Entries []Entry
}

// Parse reads an .npmrc file and returns its entries.
func Parse(path string) (*File, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	npmrc := &File{Path: path}
	scanner := bufio.NewScanner(f)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())

		// skip blank lines and comments
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		npmrc.Entries = append(npmrc.Entries, Entry{
			Key:   strings.TrimSpace(parts[0]),
			Value: strings.TrimSpace(parts[1]),
			Line:  lineNum,
		})
	}

	return npmrc, scanner.Err()
}

// Get returns the value for the given key, and whether it was found.
func (f *File) Get(key string) (string, bool) {
	for _, e := range f.Entries {
		if e.Key == key {
			return e.Value, true
		}
	}
	return "", false
}
