package coverage

import "testing"

func TestIsTestPath(t *testing.T) {
	cases := map[string]bool{
		"TESTS.md":              true,
		"sub/mise.toml":         true,
		".gitignore":            true,
		"pkg/conftest.py":       true,
		"pkg/a_test.go":         true,
		"pkg/test_a.py":         true,
		"pkg/test_a.txt":        false,
		"test_.py":              true,
		"pkg/a.tftest.hcl":      true,
		"a.hcl":                 false,
		"test/a.go":             true,
		"pkg/tests/a.py":        true,
		"acceptance/a.go":       true,
		"pkg/testdata/a.json":   true,
		"pkg/a.go":              false,
		"README.md":             false,
		"contest/a.go":          false,
		"pkg/test_ab.py":        true,
		"pkg/testing/a.go":      false,
		"pkg/dir/a.tftest.hclx": false,
	}
	for p, want := range cases {
		if got := IsTestPath(p); got != want {
			t.Errorf("IsTestPath(%q) = %v, want %v", p, got, want)
		}
	}
}
