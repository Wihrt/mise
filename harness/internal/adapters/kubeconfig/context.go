// Package kubeconfig reads the current context of a kubeconfig file.
package kubeconfig

import (
	"bufio"
	"os"
	"strings"
)

// File is the first kubeconfig file of a KUBECONFIG list.
type File struct {
	path string
}

// NewFile builds the adapter from the KUBECONFIG value, falling back to
// <home>/.kube/config.
func NewFile(kubeconfig, home string) *File {
	path, _, _ := strings.Cut(kubeconfig, ":")
	if path == "" {
		path = home + "/.kube/config"
	}
	return &File{path: path}
}

// Current is the file's current-context, or "" when it has none.
func (f *File) Current() string {
	file, err := os.Open(f.path)
	if err != nil {
		return ""
	}
	defer func() { _ = file.Close() }()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		if v, ok := strings.CutPrefix(scanner.Text(), "current-context:"); ok {
			return strings.Trim(strings.TrimSpace(v), `"`)
		}
	}
	return ""
}
