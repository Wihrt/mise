package fsys

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExists(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "f")
	if err := os.WriteFile(file, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	f := New()
	if !f.Exists(file) || !f.Exists(dir) {
		t.Error("a file and a folder exist")
	}
	if f.Exists(filepath.Join(dir, "absent")) {
		t.Error("an absent path does not exist")
	}
}
