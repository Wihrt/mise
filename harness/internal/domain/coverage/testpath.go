package coverage

import "strings"

var (
	testBasenames = map[string]bool{"mise.toml": true, ".gitignore": true, "conftest.py": true}
	testDirs      = map[string]bool{"test": true, "tests": true, "acceptance": true, "testdata": true}
)

// IsTestPath reports a path that holds tests or their wiring, not production code.
func IsTestPath(path string) bool {
	parts := strings.Split(path, "/")
	base := parts[len(parts)-1]
	if testBasenames[base] || strings.Contains(base, "_test.") ||
		(strings.HasPrefix(base, "test_") && strings.HasSuffix(base, ".py") && len(base) >= len("test_.py")) ||
		strings.HasSuffix(base, ".tftest.hcl") {
		return true
	}
	for _, d := range parts[:len(parts)-1] {
		if testDirs[d] {
			return true
		}
	}
	return false
}
