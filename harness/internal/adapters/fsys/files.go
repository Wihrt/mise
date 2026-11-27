// Package fsys gives the gates the local file system.
package fsys

import (
	"os"
	"path/filepath"
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

// Write replaces the content of a file.
func (*Files) Write(path, content string) error {
	return os.WriteFile(path, []byte(content), 0o644) //nolint:gosec // G306: COVERAGE.md is committed, not secret
}

// IsFile reports a regular file.
func (*Files) IsFile(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.Mode().IsRegular()
}

// Remove deletes a file; a missing file is fine.
func (*Files) Remove(path string) { _ = os.Remove(path) }

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
