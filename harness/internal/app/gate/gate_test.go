package gate

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

// The fakes below stand for git, the file system and mise: the use cases run
// without any process or file.

type fakeGit struct{ status string }

func (g fakeGit) Run(_ string, args ...string) (int, string) {
	if args[0] == "status" {
		return 0, g.status
	}
	return 0, ""
}

type fakeFiles map[string]string

func (f fakeFiles) Read(path string) (string, error) {
	if text, ok := f[path]; ok {
		return text, nil
	}
	return "", errors.New("no such file")
}
func (f fakeFiles) Write(path, content string) error { f[path] = content; return nil }
func (f fakeFiles) IsFile(path string) bool          { _, ok := f[path]; return ok }
func (f fakeFiles) Remove(path string)               { delete(f, path) }
func (f fakeFiles) MkdirAll(string) error            { return nil }
func (f fakeFiles) Patches(string) []string          { return nil }
func (f fakeFiles) Resolve(path string) string       { return path }
func (f fakeFiles) TempDir(string) (string, func(), error) {
	return "/tmp/x", func() {}, nil
}

// fakeTasks writes the report the task would write, then answers.
type fakeTasks struct {
	files  fakeFiles
	report string
	result TaskResult
	err    error
}

func (t fakeTasks) Run(dir, _ string, _ int) (TaskResult, error) {
	if t.report != "" {
		t.files[dir+"/.tdd/junit.xml"] = t.report
	}
	return t.result, t.err
}

type noTasks struct{}

func (noTasks) Depends(string) map[string][]string { return nil }

const md = "## div — Division\n### CALC-003 — c\n- **Critère** :\n  - **Étant donné** x\n  - **Quand** y\n  - **Alors** z\n- **Test** : `a_test.go` › `TestCALC003_C`\n"

func red(files fakeFiles, tasks fakeTasks, slug string) (int, string) {
	var out bytes.Buffer
	tasks.files = files
	code := New(fakeGit{}, files, tasks, noTasks{}, &out).Run(Options{Mode: "red", Slug: slug, Test: "abc", Task: "test", Repo: "/r", Timeout: 5, Jobs: 1})
	return code, out.String()
}

func lastLine(out string) string {
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	return lines[len(lines)-1]
}

func TestRed(t *testing.T) {
	repo := func() fakeFiles {
		return fakeFiles{"/r/TESTS.md": md, "/r/a_test.go": "package x\nfunc TestCALC003_C(t *testing.T) { Div() }\n", "/r/.tdd/junit.xml": "stale"}
	}
	failing := `<testsuite><testcase name="TestCALC003_C"><failure/></testcase></testsuite>`
	passing := `<testsuite><testcase name="TestCALC003_C"/></testsuite>`
	tests := []struct {
		name     string
		tasks    fakeTasks
		slug     string
		wantCode int
		wantLine string
	}{
		{"test fails: red", fakeTasks{report: failing, result: TaskResult{Code: 1}}, "div", 0, "RED CALC-003: TestCALC003_C"},
		{"test passes: not red", fakeTasks{report: passing}, "div", 1, "NOT RED CALC-003: TestCALC003_C pass"},
		{"stale report is deleted", fakeTasks{result: TaskResult{Code: 1}}, "div", 1, "REPORT missing: test n'a pas écrit .tdd/junit.xml"},
		{"compile error naming a symbol of the test", fakeTasks{result: TaskResult{Code: 1, Output: "a_test.go:2: undefined: Div\n"}}, "div", 0, "excused by compile/collect error: `Div` used in a_test.go:2"},
		{"task timed out", fakeTasks{result: TaskResult{Code: 124, Output: "partial", TimedOut: true}}, "div", 1, "TIMEOUT after 5s"},
		{"unknown slug", fakeTasks{}, "ghost", 2, "NO IDS FOUND"},
		{"mise missing", fakeTasks{err: ErrNoMise}, "div", 2, "ERROR: mise not found in PATH"},
	}
	for _, tt := range tests {
		code, out := red(repo(), tt.tasks, tt.slug)
		if code != tt.wantCode || !strings.Contains(out, tt.wantLine) {
			t.Errorf("%s: exit %d, want %d and %q in:\n%s", tt.name, code, tt.wantCode, tt.wantLine, out)
		}
		if want := map[int]string{0: "PASS", 1: "FAIL", 2: "FAIL"}[code]; lastLine(out) != want {
			t.Errorf("%s: last line %q, want %q", tt.name, lastLine(out), want)
		}
	}
}

func TestRedWithoutTestsFile(t *testing.T) {
	code, out := red(fakeFiles{}, fakeTasks{}, "div")
	if code != 2 || !strings.Contains(out, "ERROR: /r/TESTS.md not found") {
		t.Errorf("exit %d, out %q", code, out)
	}
}
