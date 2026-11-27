package gate

import (
	"fmt"
	"io"
	"strings"
)

// Options are the arguments of one gate run.
type Options struct {
	Task     string
	Repo     string
	Timeout  int
	Coverage []string // coverage reports the task writes
	Baseline bool     // write the findings as exceptions of reason "dette initiale"
}

// Gate runs the gates.
type Gate struct {
	git    Git
	files  Files
	tasks  Tasks
	stdout io.Writer
}

// New builds the gate over its ports.
func New(git Git, files Files, tasks Tasks, stdout io.Writer) *Gate {
	return &Gate{git: git, files: files, tasks: tasks, stdout: stdout}
}

// failure is a usage or environment problem: the gate answers FAIL with exit code 2.
type failure struct{ msg string }

func (f failure) Error() string { return f.msg }

func failf(format string, a ...any) error { return failure{fmt.Sprintf(format, a...)} }

// run is one gate run.
type run struct {
	g *Gate
	o Options
}

func (r *run) out(msg string) { _, _ = fmt.Fprintln(r.g.stdout, msg) }

func (r *run) outf(format string, a ...any) { r.out(fmt.Sprintf(format, a...)) }

// Run runs the gate and returns its exit code: 0 PASS, 1 FAIL, 2 usage or
// environment error. The last output line is PASS or FAIL.
func (g *Gate) Run(o Options) int {
	r := &run{g: g, o: o}
	ok, err := r.coverage()
	if err != nil {
		r.out("ERROR: " + err.Error())
		r.out("FAIL")
		return 2
	}
	if ok {
		r.out("PASS")
		return 0
	}
	r.out("FAIL")
	return 1
}

// runMise runs a task and prints its output. Exit code 124 means it timed out.
func (r *run) runMise(repo, task string) (int, string, error) {
	r.out("$ mise run " + task)
	res, err := r.g.tasks.Run(repo, task, r.o.Timeout)
	if err != nil {
		return 0, "", failure{err.Error()}
	}
	if res.TimedOut {
		r.out(res.Output)
		r.outf("TIMEOUT after %ds", r.o.Timeout)
		return 124, res.Output, nil
	}
	r.out(strings.TrimRight(res.Output, "\n"))
	r.outf("exit=%d", res.Code)
	return res.Code, res.Output, nil
}
