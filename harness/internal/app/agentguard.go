package app

import (
	"path/filepath"
	"slices"
	"strings"

	"github.com/Wihrt/mise/harness/internal/domain"
)

// AgentGuard holds a harness agent to its role: none pushes, the acceptance-writer
// edits test files only.
type AgentGuard struct {
	parser CommandParser
	role   string
}

// NewAgentGuard builds the policy for one role.
func NewAgentGuard(parser CommandParser, role string) *AgentGuard {
	return &AgentGuard{parser: parser, role: role}
}

var (
	guardedRoles = []string{"implementer", "acceptance-writer"}
	testBases    = []string{"mise.toml", ".gitignore", "conftest.py"}
	testGlobs    = []string{"*_test.*", "test_*.py", "*.tftest.hcl"}
	testDirs     = []string{"test", "tests", "acceptance", "testdata"}
)

// Evaluate implements Policy.
func (p *AgentGuard) Evaluate(req domain.Request) domain.Decision {
	if !slices.Contains(guardedRoles, p.role) {
		return domain.Allowed()
	}
	if req.EditsFile() {
		if p.role != "acceptance-writer" || req.Path == "" || isTestPath(req.Path, req.Cwd) {
			return domain.Allowed()
		}
		return domain.Denied("acceptance-writer may only edit test files, mise.toml, .gitignore.")
	}
	if len(gitCalls(p.parser, req, "push")) > 0 {
		return domain.Denied(p.role + " must not push. The controller pushes.")
	}
	return domain.Allowed()
}

// isTestPath reports a path the acceptance-writer owns.
func isTestPath(path, cwd string) bool {
	if !strings.HasPrefix(path, "/") {
		path = cwd + "/" + path
	}
	base := filepath.Base(path)
	if slices.Contains(testBases, base) {
		return true
	}
	for _, g := range testGlobs {
		if ok, _ := filepath.Match(g, base); ok {
			return true
		}
	}
	for _, d := range strings.Split(filepath.Dir(path), "/") {
		if slices.Contains(testDirs, d) {
			return true
		}
	}
	return false
}
