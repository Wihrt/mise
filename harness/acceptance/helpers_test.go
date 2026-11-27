// Package acceptance holds the acceptance tests of the harness binary. They are
// black-box tests: each one runs the compiled binary on a throwaway git
// repository and reads its standard output and its exit code. A test named
// Test<ID>_... proves the criterion <ID> of the "Critères d'acceptation" section of the spec.
package acceptance

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// harness is the binary under test: $HARNESS, else built from ../cmd/harness.
var harness string

func TestMain(m *testing.M) {
	os.Exit(runTests(m))
}

func runTests(m *testing.M) int {
	// Inside a git hook, git exports the repository it runs in: the tests work on their own repositories.
	if out, err := exec.CommandContext(context.Background(), "git", "rev-parse", "--local-env-vars").Output(); err == nil {
		for _, name := range strings.Fields(string(out)) {
			_ = os.Unsetenv(name)
		}
	}
	harness = os.Getenv("HARNESS")
	if harness == "" {
		dir, err := os.MkdirTemp("", "harness-bin-")
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		defer func() { _ = os.RemoveAll(dir) }()
		harness = filepath.Join(dir, "harness")
		if out, err := exec.CommandContext(context.Background(), "go", "build", "-o", harness, "../cmd/harness").CombinedOutput(); err != nil { //nolint:gosec // G204: fixed command, temporary output path
			fmt.Fprintf(os.Stderr, "go build: %v\n%s", err, out)
			return 1
		}
	}
	return m.Run()
}

// files maps a path in the repository to its content.
type files = map[string]string

func merge(all ...files) files {
	res := files{}
	for _, f := range all {
		maps.Copy(res, f)
	}
	return res
}

// repo is a throwaway git repository with, next to it, a directory of mutants.
type repo struct {
	t       *testing.T
	dir     string
	parent  string
	mutants string
	// wiring holds the wiring files of the first commit; nil means a hook citing `mise run test`.
	wiring files
	// reports are the coverage report paths of covRepo, sorted.
	reports []string
}

func newRepo(t *testing.T) *repo {
	t.Helper()
	parent, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("MISE_TRUSTED_CONFIG_PATHS", parent)
	t.Setenv("MISE_YES", "1")
	r := &repo{t: t, dir: filepath.Join(parent, "repo"), parent: parent, mutants: filepath.Join(parent, "mutants")}
	mkdir(t, r.dir)
	r.sh("git", "init", "-q", "-b", "main")
	return r
}

func mkdir(t *testing.T, dir string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o750); err != nil {
		t.Fatal(err)
	}
}

func write(t *testing.T, path, content string) {
	t.Helper()
	mkdir(t, filepath.Dir(path))
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func read(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path) //nolint:gosec // G304: a file of the test's own temporary directory
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// run runs a command in dir and returns its exit code and its standard output.
func run(t *testing.T, dir string, env []string, name string, args ...string) (int, string) {
	t.Helper()
	code, out, _ := runBoth(t, dir, env, name, args...)
	return code, out
}

// runBoth runs a command in dir and returns its exit code, its standard output
// and its standard error.
func runBoth(t *testing.T, dir string, env []string, name string, args ...string) (int, string, string) {
	t.Helper()
	cmd := exec.CommandContext(t.Context(), name, args...) //nolint:gosec // G204: the tests choose the command
	var out, errOut bytes.Buffer
	cmd.Dir, cmd.Stdout, cmd.Stderr, cmd.Env = dir, &out, &errOut, append(os.Environ(), env...)
	err := cmd.Run()
	var exit *exec.ExitError
	switch {
	case err == nil:
		return 0, out.String(), errOut.String()
	case errors.As(err, &exit):
		return exit.ExitCode(), out.String(), errOut.String()
	default:
		t.Fatalf("%s %v: %v", name, args, err)
		return 0, "", ""
	}
}

// sh runs a command in the repository, which must succeed, and returns its trimmed output.
func (r *repo) sh(name string, args ...string) string {
	r.t.Helper()
	code, out := run(r.t, r.dir, nil, name, args...)
	if code != 0 {
		r.t.Fatalf("%s %v: exit %d\n%s", name, args, code, out)
	}
	return strings.TrimSpace(out)
}

func (r *repo) write(name, content string) {
	r.t.Helper()
	write(r.t, filepath.Join(r.dir, name), content)
}

// commit writes the files and commits everything. The first commit also
// carries the wiring files: by default a commit hook citing `mise run test`.
func (r *repo) commit(f files, msg string) string {
	r.t.Helper()
	if code, _ := run(r.t, r.dir, nil, "git", "rev-parse", "--verify", "-q", "HEAD"); code != 0 {
		wiring := r.wiring
		if wiring == nil {
			wiring = files{".pre-commit-config.yaml": precommit("test")}
		}
		f = merge(wiring, f)
	}
	for name, content := range f {
		r.write(name, content)
	}
	r.sh("git", "add", "-A")
	r.sh("git", "-c", "user.name=t", "-c", "user.email=t@t", "commit", "-q", "--no-verify", "-m", msg)
	return r.sh("git", "rev-parse", "HEAD")
}

// gate runs `harness gate <args> --repo <repo>`.
func (r *repo) gate(args ...string) (int, string) {
	r.t.Helper()
	return run(r.t, r.dir, nil, harness, append(append([]string{"gate"}, args...), "--repo", r.dir)...)
}

type miseOpts struct {
	checkRC      int
	tools        string
	task         string // default "test"
	checkDepends string
	// testDepends adds the aggregate `[tasks.test] depends = ["test:*"]` (for a task such as test:render).
	testDepends bool
}

func miseToml(o miseOpts) string {
	if o.task == "" {
		o.task = "test"
	}
	s := fmt.Sprintf("[tools]\n%s\n[tasks.%q]\nrun = \"sh run_test.sh\"\n", o.tools, o.task)
	if o.testDepends {
		s += "[tasks.test]\ndepends = [\"test:*\"]\n"
	}
	s += "[tasks.check]\n"
	if o.checkDepends != "" {
		s += fmt.Sprintf("depends = [%q]\n", o.checkDepends)
	}
	return s + fmt.Sprintf("run = \"exit %d\"\n", o.checkRC)
}

// precommit is a .pre-commit-config.yaml whose local hooks cite `mise run <task>` for each task.
func precommit(tasks ...string) string {
	hooks := ""
	for i, task := range tasks {
		hooks += fmt.Sprintf("      - id: h%d\n        name: h%d\n        entry: mise run %s\n        language: system\n        pass_filenames: false\n", i, i, task)
	}
	return "repos:\n  - repo: local\n    hooks:\n" + hooks
}

var ()

func lines(out string) []string {
	if out == "" {
		return nil
	}
	return strings.Split(strings.TrimSuffix(out, "\n"), "\n")
}

// last is the last line of the output: PASS or FAIL.
func last(out string) string {
	l := lines(out)
	if len(l) == 0 {
		return ""
	}
	return l[len(l)-1]
}

// with fails the test unless the output contains every string.
func with(t *testing.T, out string, wants ...string) {
	t.Helper()
	for _, w := range wants {
		if !strings.Contains(out, w) {
			t.Errorf("output lacks %q:\n%s", w, out)
		}
	}
}

// without fails the test when the output contains one of the strings.
func without(t *testing.T, out string, bans ...string) {
	t.Helper()
	for _, b := range bans {
		if strings.Contains(out, b) {
			t.Errorf("output contains %q:\n%s", b, out)
		}
	}
}

// ends fails the test unless the exit code and the last line are the expected ones.
func ends(t *testing.T, code int, out string, wantCode int, wantLast string) {
	t.Helper()
	if code != wantCode || last(out) != wantLast {
		t.Errorf("exit %d, last line %q; want %d, %q:\n%s", code, last(out), wantCode, wantLast, out)
	}
}

// covRepo is a repository for `harness gate coverage`: it commits f, a mise
// task `test`, `.gitignore` and, per report (key = path in the repo, value =
// content), a tracked .txt fixture that the task copies to that path. A report
// with an empty content is not written by the task. The task then exits rc.
func covRepo(t *testing.T, f, reports files, rc int) *repo {
	t.Helper()
	r := newRepo(t)
	r.reports = slices.Sorted(maps.Keys(reports))
	script := ""
	fixtures := files{}
	for i, path := range r.reports {
		if reports[path] == "" {
			continue
		}
		fixture := fmt.Sprintf("covfixtures/report%d.txt", i)
		fixtures[fixture] = reports[path]
		script += fmt.Sprintf("mkdir -p %q\ncp %s %s\n", filepath.Dir(path), fixture, path)
	}
	r.commit(merge(f, fixtures, files{
		"mise.toml":   miseToml(miseOpts{}),
		".gitignore":  ".tdd\n",
		"run_test.sh": script + fmt.Sprintf("exit %d\n", rc),
	}), "coverage repo")
	return r
}

// coverage runs `harness gate coverage` on the task `test` with one --coverage
// option per report of covRepo, in sorted order, then the extra arguments.
func (r *repo) coverage(extra ...string) (int, string) {
	r.t.Helper()
	args := []string{"coverage", "--task", "test"}
	for _, path := range r.reports {
		args = append(args, "--coverage", path)
	}
	return r.gate(append(args, extra...)...)
}
