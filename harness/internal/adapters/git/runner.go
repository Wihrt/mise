package git

import (
	"bytes"
	"context"
	"errors"
	"os/exec"
	"strings"
)

// Runner runs any git command in a directory, for the gates.
type Runner struct{}

// NewRunner builds the adapter.
func NewRunner() *Runner { return &Runner{} }

// Run returns git's exit code and its standard output; its error output is dropped.
func (r *Runner) Run(dir string, args ...string) (int, string) {
	// The gate chooses the subcommands; revisions and paths come from its command line.
	cmd := exec.CommandContext(context.Background(), "git", append([]string{"-C", dir}, args...)...) //nolint:gosec // G204
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	text := strings.ToValidUTF8(out.String(), "�")
	var exit *exec.ExitError
	switch {
	case err == nil:
		return 0, text
	case errors.As(err, &exit):
		return exit.ExitCode(), text
	default:
		return 127, text
	}
}

// OriginURL is the URL of the origin remote of the repository in dir.
func (r *Runner) OriginURL(dir string) (string, error) {
	code, out := r.Run(dir, "remote", "get-url", "origin")
	if code != 0 {
		return "", errors.New("git remote get-url origin failed in " + dir)
	}
	return out, nil
}
