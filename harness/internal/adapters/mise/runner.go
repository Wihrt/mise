// Package mise runs mise tasks and reads mise.toml.
package mise

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"slices"
	"strings"
	"syscall"
	"time"

	"github.com/pelletier/go-toml/v2"

	"github.com/Wihrt/mise/harness/internal/app/gate"
)

// Runner runs `mise run <task>`.
type Runner struct{ environ []string }

// NewRunner builds the adapter over the process environment.
func NewRunner() *Runner { return &Runner{environ: os.Environ()} }

// goEnv returns the environment with -count=1 in GOFLAGS: go's test cache must
// not hide source changes.
func goEnv(environ []string) []string {
	env, flags := []string{}, []string{}
	for _, kv := range environ {
		if v, ok := strings.CutPrefix(kv, "GOFLAGS="); ok {
			flags = strings.Fields(v)
			continue
		}
		env = append(env, kv)
	}
	if !slices.Contains(flags, "-count=1") {
		flags = append(flags, "-count=1")
	}
	return append(env, "GOFLAGS="+strings.Join(flags, " "))
}

// Run runs the task in dir, standard output and error merged. Past the
// timeout the task and its children are killed.
func (r *Runner) Run(dir, task string, timeoutSeconds int) (gate.TaskResult, error) {
	path, err := exec.LookPath("mise")
	if err != nil {
		return gate.TaskResult{}, gate.ErrNoMise
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutSeconds)*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, path, "run", task) //nolint:gosec // G204: the task is the one the gate is asked to judge
	var out bytes.Buffer
	cmd.Dir, cmd.Env, cmd.Stdout, cmd.Stderr = dir, goEnv(r.environ), &out, &out
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error { return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL) }
	cmd.WaitDelay = 2 * time.Second
	err = cmd.Run()
	text := strings.ToValidUTF8(out.String(), "�")
	if ctx.Err() != nil {
		return gate.TaskResult{Code: 124, Output: text, TimedOut: true}, nil
	}
	var exit *exec.ExitError
	switch {
	case err == nil:
		return gate.TaskResult{Output: text}, nil
	case errors.As(err, &exit):
		code := exit.ExitCode()
		if ws, ok := exit.Sys().(syscall.WaitStatus); ok && ws.Signaled() {
			code = -int(ws.Signal())
		}
		return gate.TaskResult{Code: code, Output: text}, nil
	default:
		return gate.TaskResult{}, err
	}
}

// TaskFile reads the tasks of a mise.toml.
type TaskFile struct{}

// Depends maps each task of the file to its direct `depends`; an unreadable file has no task.
func (TaskFile) Depends(text string) map[string][]string {
	var file struct {
		Tasks map[string]map[string]any `toml:"tasks"`
	}
	if err := toml.Unmarshal([]byte(text), &file); err != nil {
		return nil
	}
	depends := map[string][]string{}
	for name, task := range file.Tasks {
		depends[name] = nil
		switch d := task["depends"].(type) {
		case string:
			depends[name] = []string{d}
		case []any:
			for _, v := range d {
				if s, ok := v.(string); ok {
					depends[name] = append(depends[name], s)
				}
			}
		}
	}
	return depends
}
