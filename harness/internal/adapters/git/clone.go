package git

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// Clone reads the history of one local clone, and runs commands on its commits.
type Clone struct{ dir string }

// NewClone builds the adapter for the clone at dir.
func NewClone(dir string) *Clone { return &Clone{dir: dir} }

// git returns git's trimmed output; its error carries what git wrote.
func (c *Clone) git(timeout time.Duration, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	// The subcommands are fixed by this package; revisions come from the forge.
	cmd := exec.CommandContext(ctx, "git", append([]string{"-C", c.dir}, args...)...) //nolint:gosec // G204
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("git %s: %s: %w", args[0], strings.TrimSpace(stderr.String()), err)
	}
	return strings.TrimSpace(stdout.String()), nil
}

// Fetch updates the remote branches.
func (c *Clone) Fetch() error {
	_, err := c.git(5*time.Minute, "fetch", "--quiet", "origin")
	return err
}

// Remote is the commit of a branch of origin.
func (c *Clone) Remote(branch string) (string, error) {
	return c.git(10*time.Second, "rev-parse", "--verify", "--quiet", "refs/remotes/origin/"+branch)
}

// Tree is the tree of a commit.
func (c *Clone) Tree(rev string) (string, error) {
	return c.git(10*time.Second, "rev-parse", "--verify", "--quiet", rev+"^{tree}")
}

// IsAncestor reports whether a commit is contained in another.
func (c *Clone) IsAncestor(ancestor, descendant string) bool {
	_, err := c.git(10*time.Second, "merge-base", "--is-ancestor", ancestor, descendant)
	return err == nil
}

// ChangedFiles lists the files that differ between two commits.
func (c *Clone) ChangedFiles(from, to string) ([]string, error) {
	out, err := c.git(time.Minute, "diff", "--name-only", from, to)
	if err != nil {
		return nil, err
	}
	return strings.Fields(out), nil
}

// Run checks a commit out in a throwaway worktree and runs a shell command there.
func (c *Clone) Run(sha, command string) error {
	parent, err := os.MkdirTemp("", "harness-validate-")
	if err != nil {
		return fmt.Errorf("temporary directory: %w", err)
	}
	defer func() { _ = os.RemoveAll(parent) }()
	tree := parent + "/tree"
	if _, err := c.git(5*time.Minute, "worktree", "add", "--quiet", "--detach", tree, sha); err != nil {
		return err
	}
	defer func() { _, _ = c.git(time.Minute, "worktree", "remove", "--force", tree) }()
	// The command is the one the user wrote after --validate.
	cmd := exec.CommandContext(context.Background(), "sh", "-c", command) //nolint:gosec // G204
	cmd.Dir = tree
	cmd.Stdout, cmd.Stderr = os.Stderr, os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("on %s: %w", sha[:min(7, len(sha))], err)
	}
	return nil
}
