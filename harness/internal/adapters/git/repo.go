// Package git answers questions about a repository by running git.
package git

import (
	"context"
	"os"
	"os/exec"
	"strings"
	"time"
)

// Repo reads the state of the repository a directory belongs to.
type Repo struct{}

// NewRepo builds the adapter.
func NewRepo() *Repo { return &Repo{} }

// run returns git's trimmed output, and whether it succeeded.
func run(dir string, args ...string) (string, bool) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	// The subcommands are fixed by this package; only the directory and the path come from the hook input.
	out, err := exec.CommandContext(ctx, "git", append([]string{"-C", dir}, args...)...).Output() //nolint:gosec // G204
	return strings.TrimSpace(string(out)), err == nil
}

// Exists reports whether dir is a directory.
func (r *Repo) Exists(dir string) bool {
	info, err := os.Stat(dir)
	return err == nil && info.IsDir()
}

// InMainCheckout reports a directory in the main checkout of a repository that
// has commits; false outside git, in a linked worktree or in an empty repository.
func (r *Repo) InMainCheckout(dir string) bool {
	if inside, ok := run(dir, "rev-parse", "--is-inside-work-tree"); !ok || inside != "true" {
		return false
	}
	if _, ok := run(dir, "rev-parse", "--verify", "--quiet", "HEAD"); !ok {
		return false
	}
	gitDir, ok1 := run(dir, "rev-parse", "--path-format=absolute", "--git-dir")
	common, ok2 := run(dir, "rev-parse", "--path-format=absolute", "--git-common-dir")
	return ok1 && ok2 && gitDir == common
}

// Ignored reports whether git ignores path.
func (r *Repo) Ignored(dir, path string) bool {
	_, ok := run(dir, "check-ignore", "-q", path)
	return ok
}

// Default is the default branch: origin/HEAD, else main or master.
func (r *Repo) Default(dir string) string {
	if ref, ok := run(dir, "symbolic-ref", "--short", "refs/remotes/origin/HEAD"); ok {
		return strings.TrimPrefix(ref, "origin/")
	}
	for _, b := range []string{"main", "master"} {
		if _, ok := run(dir, "show-ref", "--verify", "--quiet", "refs/heads/"+b); ok {
			return b
		}
	}
	return ""
}

// Current is the checked-out branch, or "" on a detached HEAD.
func (r *Repo) Current(dir string) string {
	branch, _ := run(dir, "symbolic-ref", "--short", "HEAD")
	return branch
}
