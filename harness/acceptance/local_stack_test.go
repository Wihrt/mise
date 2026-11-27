package acceptance

import (
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func stack(t *testing.T, dir string) (int, string, string) {
	t.Helper()
	return runBoth(t, dir, nil, harness, "local", "stack", "--repo", dir)
}

func TestMNR001_MonorepoNamesTheStackOfEachService(t *testing.T) {
	dir := t.TempDir()
	write(t, filepath.Join(dir, "mise.toml"), "monorepo_root = true\n\n[monorepo]\nconfig_roots = [\"services/a\", \"services/b\", \"charts/c\"]\n")
	write(t, filepath.Join(dir, "services/a/go.mod"), "module a\n")
	write(t, filepath.Join(dir, "services/a/cmd/main.go"), "package main\n")
	write(t, filepath.Join(dir, "services/a/Dockerfile"), "FROM scratch\n")
	write(t, filepath.Join(dir, "services/b/pyproject.toml"), "[project]\nname = \"b\"\n")
	write(t, filepath.Join(dir, "services/b/src/b/__init__.py"), "")
	write(t, filepath.Join(dir, "charts/c/pyproject.toml"), "[project]\nname = \"c\"\n")

	code, out, errOut := stack(t, dir)
	if code != 0 {
		t.Errorf("exit = %d, want 0 (stderr: %s)", code, errOut)
	}
	want := []string{
		"services/a stack=stack-go-app image=yes",
		"services/b stack=stack-py-app image=no",
		"charts/c stack=none",
	}
	if got := lines(out); !slices.Equal(got, want) {
		t.Errorf("output = %q, want %q", got, want)
	}
}

func TestMNR002_SimpleRepoKeepsItsBehaviour(t *testing.T) {
	t.Run("one line for the root", func(t *testing.T) {
		dir := t.TempDir()
		write(t, filepath.Join(dir, "go.mod"), "module x\n")
		write(t, filepath.Join(dir, "internal/x.go"), "package internal\n")

		code, out, errOut := stack(t, dir)
		if code != 0 {
			t.Errorf("exit = %d, want 0 (stderr: %s)", code, errOut)
		}
		if got, want := lines(out), []string{". stack=stack-go-app image=no"}; !slices.Equal(got, want) {
			t.Errorf("output = %q, want %q", got, want)
		}
	})

	t.Run("monorepo_root without config_roots fails", func(t *testing.T) {
		dir := t.TempDir()
		write(t, filepath.Join(dir, "mise.toml"), "monorepo_root = true\n")

		code, _, errOut := stack(t, dir)
		if code != 2 {
			t.Errorf("exit = %d, want 2", code)
		}
		if !strings.Contains(errOut, "config_roots") {
			t.Errorf("stderr = %q, want it to name %s", errOut, "config_roots")
		}
	})

	t.Run("a listed folder that is absent fails", func(t *testing.T) {
		dir := t.TempDir()
		write(t, filepath.Join(dir, "mise.toml"), "monorepo_root = true\n\n[monorepo]\nconfig_roots = [\"services/a\", \"services/absent\"]\n")
		write(t, filepath.Join(dir, "services/a/go.mod"), "module a\n")
		write(t, filepath.Join(dir, "services/a/cmd/main.go"), "package main\n")

		code, _, errOut := stack(t, dir)
		if code != 2 {
			t.Errorf("exit = %d, want 2", code)
		}
		if !strings.Contains(errOut, "services/absent") {
			t.Errorf("stderr = %q, want it to name %s", errOut, "services/absent")
		}
	})
}
