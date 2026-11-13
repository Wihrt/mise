package acceptance

import (
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// Slug fichiers-locaux: LOC-001 to LOC-003. Boundary: the `harness local` command line.

// depot makes a repository whose origin is the URL.
func depot(t *testing.T, origin string) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "depot")
	mkdir(t, dir)
	for _, args := range [][]string{{"init", "-q"}, {"remote", "add", "origin", origin}} {
		if code, _ := run(t, dir, nil, "git", args...); code != 0 {
			t.Fatalf("git %v: exit %d", args, code)
		}
	}
	return dir
}

// local runs `harness local <args>` over a local directory holding the files.
func local(t *testing.T, f files, args ...string) (int, string) {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "local")
	mkdir(t, dir)
	for name, text := range f {
		write(t, filepath.Join(dir, name), text)
	}
	return run(t, dir, []string{"CLAUDE_LOCAL_DIR=" + dir}, harness, append([]string{"local"}, args...)...)
}

func forge(t *testing.T, origin string, f files, wantCode int, want string) {
	t.Helper()
	code, out := local(t, f, "forge", "--repo", depot(t, origin))
	if code != wantCode || strings.TrimSpace(out) != want {
		t.Errorf("exit %d, output %q; want %d, %q", code, out, wantCode, want)
	}
}

func TestLOC001_DeclaredHostGivesItsForgeTool(t *testing.T) {
	forge(t, "https://forge.example.test/groupe/depot.git", files{"forges": "forge.example.test glab\n"}, 0, "glab")
}

func TestLOC002_GithubHostIsGh(t *testing.T) {
	forge(t, "git@github.com:groupe/depot.git", nil, 0, "gh")
}

func TestLOC002_GitlabHostIsGlab(t *testing.T) {
	forge(t, "https://gitlab.com/groupe/depot.git", nil, 0, "glab")
}

func TestLOC002_UnknownHostExits1AndIsNamed(t *testing.T) {
	forge(t, "https://forge.example.test/groupe/depot.git", nil, 1, "hôte forge.example.test inconnu")
}

func TestLOC003_RealCaseLineNamingTheIDIsPrinted(t *testing.T) {
	const line = "| PRV-015 | dépôt interne, critère X-103 |"
	code, out := local(t, files{"cas-reels.md": "| PRV-014 | autre |\n" + line + "\n"}, "cas-reels", "PRV-015")
	if code != 0 || !slices.Contains(lines(out), line) {
		t.Errorf("exit %d, output %q", code, out)
	}
}

func TestLOC003_MissingFileGivesEmptyOutput(t *testing.T) {
	if code, out := local(t, nil, "cas-reels", "PRV-015"); code != 0 || out != "" {
		t.Errorf("exit %d, output %q", code, out)
	}
}
