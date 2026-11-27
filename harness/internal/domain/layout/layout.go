// Package layout tells which folders a repository holds as services, and the
// stack skill each one follows.
package layout

import (
	"errors"
	"fmt"
	"path"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

// Line is the stack of a folder; Stack is "none" when the folder has none, and
// Image is then not reported.
type Line struct {
	Dir   string
	Stack string
	Image bool
}

func (l Line) String() string {
	if l.Stack == "none" {
		return l.Dir + " stack=none"
	}
	image := "no"
	if l.Image {
		image = "yes"
	}
	return fmt.Sprintf("%s stack=%s image=%s", l.Dir, l.Stack, image)
}

// Roots are the service folders a root mise.toml lists in config_roots, or nil
// for a repository that is not a monorepo (no monorepo_root = true).
func Roots(miseToml []byte) ([]string, error) {
	var cfg struct {
		MonorepoRoot bool `toml:"monorepo_root"`
		Monorepo     struct {
			ConfigRoots []string `toml:"config_roots"`
		} `toml:"monorepo"`
	}
	if err := toml.Unmarshal(miseToml, &cfg); err != nil {
		return nil, fmt.Errorf("mise.toml: %w", err)
	}
	if !cfg.MonorepoRoot {
		return nil, nil
	}
	roots := cfg.Monorepo.ConfigRoots
	if len(roots) == 0 {
		return nil, errors.New("monorepo_root = true without config_roots in mise.toml")
	}
	for i, root := range roots {
		if err := checkRoot(root, roots[:i]); err != nil {
			return nil, err
		}
	}
	return roots, nil
}

func checkRoot(root string, before []string) error {
	if path.IsAbs(root) || path.Clean(root) != root || root == "." || root == ".." || strings.HasPrefix(root, "../") {
		return fmt.Errorf("config_roots entry %q must be a clean relative path inside the repository", root)
	}
	for _, other := range before {
		switch {
		case other == root:
			return fmt.Errorf("config_roots entry %q is listed twice", root)
		case strings.HasPrefix(root, other+"/") || strings.HasPrefix(other, root+"/"):
			return fmt.Errorf("config_roots entries %q and %q are nested", other, root)
		}
	}
	return nil
}

// Detect names the stack of dir, one line per stack, a single "none" line when
// it has neither. has reports whether a path relative to the repository exists.
func Detect(has func(rel string) bool, dir string) []Line {
	at := func(name string) bool { return has(path.Join(dir, name)) }
	image := at("Dockerfile")
	var lines []Line
	if at("go.mod") && (at("cmd") || at("internal")) {
		lines = append(lines, Line{Dir: dir, Stack: "stack-go-app", Image: image})
	}
	if at("pyproject.toml") && at("src") {
		lines = append(lines, Line{Dir: dir, Stack: "stack-py-app", Image: image})
	}
	if lines == nil {
		lines = []Line{{Dir: dir, Stack: "none"}}
	}
	return lines
}
