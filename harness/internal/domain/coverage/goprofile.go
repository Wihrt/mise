package coverage

import (
	"fmt"
	"path"
	"strings"
)

// ParseGoProfile reads a Go coverage profile (any mode: only executed or not
// counts). modules maps a module path to its directory in the repository, "."
// for the root; a file goes to the module with the longest path.
func ParseGoProfile(text string, modules map[string]string) (Measure, error) {
	m := Measure{}
	for _, line := range strings.Split(text, "\n") {
		if line == "" || strings.HasPrefix(line, "mode:") {
			continue
		}
		var name string
		var s span
		var stmts, count int
		i := strings.LastIndex(line, ":")
		if i < 0 {
			return nil, fmt.Errorf("bad profile line %q", line)
		}
		name = line[:i]
		if _, err := fmt.Sscanf(line[i+1:], "%d.%d,%d.%d %d %d",
			&s.start.line, &s.start.col, &s.end.line, &s.end.col, &stmts, &count); err != nil {
			return nil, fmt.Errorf("bad profile line %q", line)
		}
		file, ok := repoFile(name, modules)
		if !ok || stmts == 0 {
			continue
		}
		if m[file] == nil {
			m[file] = map[span]bool{}
		}
		m[file][s] = m[file][s] || count > 0
	}
	return m, nil
}

// repoFile maps a file named by module path to its path in the repository.
func repoFile(name string, modules map[string]string) (string, bool) {
	best := ""
	found := false
	for mod := range modules {
		if strings.HasPrefix(name, mod+"/") && (!found || len(mod) > len(best)) {
			best, found = mod, true
		}
	}
	if !found {
		return "", false
	}
	return path.Join(modules[best], strings.TrimPrefix(name, best+"/")), true
}
