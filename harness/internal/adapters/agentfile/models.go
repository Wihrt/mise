// Package agentfile reads the agent definitions of a Claude Code directory.
package agentfile

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

// Dir is a directory of <agent>.md definitions.
type Dir struct {
	path string
}

// NewDir builds the adapter.
func NewDir(path string) *Dir { return &Dir{path: path} }

// Model is the first word of the definition's "model:" line, or "".
func (d *Dir) Model(agentType string) string {
	file, err := os.Open(filepath.Join(d.path, filepath.Base(agentType)+".md"))
	if err != nil {
		return ""
	}
	defer func() { _ = file.Close() }()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		if v, ok := strings.CutPrefix(scanner.Text(), "model:"); ok {
			fields := strings.Fields(v)
			if len(fields) == 0 {
				return ""
			}
			return fields[0]
		}
	}
	return ""
}
