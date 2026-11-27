package layout

import (
	"slices"
	"strings"
	"testing"
)

func TestRoots(t *testing.T) {
	if got, err := Roots(true, []string{"b/x", "a", "c"}); err != nil || !slices.Equal(got, []string{"b/x", "a", "c"}) {
		t.Errorf("order: %q, %v", got, err)
	}
	if got, err := Roots(false, nil); err != nil || got != nil {
		t.Errorf("not a monorepo: %q, %v", got, err)
	}
	errs := map[string][]string{
		"config_roots":   nil,
		"/abs":           {"/abs"},
		"../out":         {"../out"},
		"..":             {".."},
		"a/../b":         {"a/../b"},
		"a/":             {"a/"},
		`"."`:            {"."},
		"listed twice":   {"a", "b", "a"},
		"nested":         {"services", "services/a"},
		"nested reverse": {"services/a", "services"},
	}
	for want, in := range errs {
		if _, err := Roots(true, in); err == nil || !strings.Contains(err.Error(), strings.Trim(strings.Split(want, " ")[0], `"`)) {
			t.Errorf("%s: err = %v", want, err)
		}
	}
	if _, err := Roots(true, []string{"a", "ab"}); err != nil {
		t.Errorf("sibling sharing a prefix: %v", err)
	}
}

func TestDetect(t *testing.T) {
	has := func(paths ...string) func(string) bool {
		return func(rel string) bool { return slices.Contains(paths, rel) }
	}
	tests := map[string]struct {
		has  func(string) bool
		want []string
	}{
		"go":         {has("d/go.mod", "d/internal"), []string{"d stack=stack-go-app image=no"}},
		"go cmd":     {has("d/go.mod", "d/cmd"), []string{"d stack=stack-go-app image=no"}},
		"go no dir":  {has("d/go.mod"), []string{"d stack=none"}},
		"python":     {has("d/pyproject.toml", "d/src"), []string{"d stack=stack-py-app image=no"}},
		"py no src":  {has("d/pyproject.toml"), []string{"d stack=none"}},
		"both image": {has("d/go.mod", "d/cmd", "d/pyproject.toml", "d/src", "d/Dockerfile"), []string{"d stack=stack-go-app image=yes", "d stack=stack-py-app image=yes"}},
		"none image": {has("d/Dockerfile"), []string{"d stack=none"}},
		"root":       {has("go.mod", "cmd"), []string{". stack=stack-go-app image=no"}},
	}
	for name, tc := range tests {
		dir := "d"
		if name == "root" {
			dir = "."
		}
		var got []string
		for _, l := range Detect(tc.has, dir) {
			got = append(got, l.String())
		}
		if !slices.Equal(got, tc.want) {
			t.Errorf("%s: %q, want %q", name, got, tc.want)
		}
	}
}
