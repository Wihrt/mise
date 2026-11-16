// Package fsys gives the gates the local file system.
package fsys

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Files reads and writes local files.
type Files struct{}

// New builds the adapter.
func New() *Files { return &Files{} }

// Read returns the content of a file, invalid UTF-8 replaced.
func (*Files) Read(path string) (string, error) {
	data, err := os.ReadFile(path) //nolint:gosec // G304: the gate reads the files of the repository it judges
	return strings.ToValidUTF8(string(data), "�"), err
}

// IsFile reports a regular file.
func (*Files) IsFile(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.Mode().IsRegular()
}

// Remove deletes a file; a missing file is fine.
func (*Files) Remove(path string) { _ = os.Remove(path) }

// MkdirAll creates a directory and its parents.
func (*Files) MkdirAll(path string) error {
	return os.MkdirAll(path, 0o755) //nolint:gosec // G301: a report directory inside the repository
}

// Patches lists the *.patch files of a directory, sorted.
func (*Files) Patches(dir string) []string {
	matches, _ := filepath.Glob(filepath.Join(dir, "*.patch"))
	sort.Strings(matches)
	return matches
}

// Resolve makes a path absolute, symbolic links followed.
func (*Files) Resolve(path string) string {
	abs, err := filepath.Abs(path)
	if err != nil {
		return path
	}
	if real, err := filepath.EvalSymlinks(abs); err == nil {
		return real
	}
	return abs
}

// TempDir creates a temporary directory and returns it with the function that deletes it.
func (*Files) TempDir(prefix string) (string, func(), error) {
	dir, err := os.MkdirTemp("", prefix)
	if err != nil {
		return "", func() {}, err
	}
	return dir, func() { _ = os.RemoveAll(dir) }, nil
}
