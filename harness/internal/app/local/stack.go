package local

import (
	"fmt"
	"path/filepath"

	"github.com/pelletier/go-toml/v2"

	"github.com/Wihrt/mise/harness/internal/domain/layout"
)

// Stack prints the stack of each folder of the repository: its root, or every
// config_roots entry of a monorepo. The code is 2 when the layout is wrong.
func (l *Local) Stack(repo string) (int, error) {
	roots, err := l.roots(repo)
	if err != nil {
		return 2, err
	}
	has := func(rel string) bool { return l.files.Exists(filepath.Join(repo, rel)) }
	for _, root := range roots {
		if !has(root) {
			return 2, fmt.Errorf("config_roots entry %s is not a folder of the repository", root)
		}
	}
	for _, root := range roots {
		for _, line := range layout.Detect(has, root) {
			_, _ = fmt.Fprintln(l.stdout, line)
		}
	}
	return 0, nil
}

// roots are the folders to detect: "." alone unless a root mise.toml makes a monorepo.
func (l *Local) roots(repo string) ([]string, error) {
	for _, name := range []string{"mise.toml", ".config/mise.toml", ".mise.toml"} {
		text, err := l.files.Read(filepath.Join(repo, name))
		if err != nil {
			continue
		}
		var cfg struct {
			MonorepoRoot bool `toml:"monorepo_root"`
			Monorepo     struct {
				ConfigRoots []string `toml:"config_roots"`
			} `toml:"monorepo"`
		}
		if err := toml.Unmarshal([]byte(text), &cfg); err != nil {
			return nil, fmt.Errorf("mise.toml: %w", err)
		}
		roots, err := layout.Roots(cfg.MonorepoRoot, cfg.Monorepo.ConfigRoots)
		if err != nil || roots != nil {
			return roots, err
		}
		break // the first root mise.toml found wins
	}
	return []string{"."}, nil
}
