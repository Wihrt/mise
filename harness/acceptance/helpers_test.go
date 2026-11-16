// Package acceptance holds the acceptance tests of the harness binary. They are
// black-box tests: each one runs the compiled binary on a throwaway git
// repository and reads its standard output and its exit code. A test named
// Test<ID>_... proves the criterion <ID> of claude/TESTS.md.
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

func (r *repo) remove(name string) {
	r.t.Helper()
	if err := os.Remove(filepath.Join(r.dir, name)); err != nil {
		r.t.Fatal(err)
	}
}

func (r *repo) status() string { r.t.Helper(); return r.sh("git", "status", "--short") }

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

// replay runs `harness gate replay` on two commits with the mutants of the repo.
func (r *repo) replay(base, head string, extra ...string) (int, string) {
	r.t.Helper()
	return r.gate(append([]string{"replay", "--base", base, "--head", head, "--proofs", r.mutants, "--task", "test"}, extra...)...)
}

// intp is an optional exit code.
func intp(n int) *int { return &n }

// taskScript is a canned task: it copies green.xml if the feat commit added
// it, else the report; it exits rc, or 1 on any failure when rc is nil.
func taskScript(report string, rc *int, extra string) string {
	body := fmt.Sprintf("mkdir -p .tdd\n%s\nf=%s\n[ -f green.xml ] && f=green.xml\ncp $f .tdd/junit.xml\n", extra, report)
	if rc == nil {
		return body + "if grep -q '<failure\\|<error' $f; then exit 1; fi\nexit 0\n"
	}
	return body + fmt.Sprintf("exit %d\n", *rc)
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

// tc is a reported test case: its name and pass, fail or error.
type tc [2]string

func junit(cases ...tc) string {
	body := ""
	for _, c := range cases {
		child := map[string]string{"pass": "", "fail": "<failure message='x'/>", "error": "<error message='x'/>"}[c[1]]
		body += fmt.Sprintf("<testcase name='%s'>%s</testcase>", c[0], child)
	}
	return "<testsuites><testsuite name='s'>" + body + "</testsuite></testsuites>"
}

// precommit is a .pre-commit-config.yaml whose local hooks cite `mise run <task>` for each task.
func precommit(tasks ...string) string {
	hooks := ""
	for i, task := range tasks {
		hooks += fmt.Sprintf("      - id: h%d\n        name: h%d\n        entry: mise run %s\n        language: system\n        pass_filenames: false\n", i, i, task)
	}
	return "repos:\n  - repo: local\n    hooks:\n" + hooks
}

func entry(id, title, file, name string) string {
	return "### " + id + " — " + title + "\n- **Critère** :\n  - **Étant donné** x\n  - **Quand** y\n  - **Alors** z\n- **Test** : `" + file + "` › `" + name + "`\n"
}

var (
	testsMD = "## pow — Power\n" + entry("CALC-001", "a", "a_test.go", "TestCALC001_A") + entry("CALC-002", "b", "a_test.go", "TestCALC002_B") +
		"\n## div — Division\n" + entry("CALC-003", "c", "a_test.go", "TestCALC003_C")
	goTests = "package x\n" +
		"func TestSHA256Sum(t *testing.T) {}\n" +
		"func TestCALC001_A(t *testing.T) { A() }\n" +
		"func TestCALC002_B(t *testing.T) { Div() }\n" +
		"func TestCALC003_C(t *testing.T) {}\n"
	spec = "# s\n## Critères d'acceptation\n### CALC-003 — c\n- **Alors** x\n" +
		"### CALC-001 — a\n- **Alors** x\n### CALC-002 — b\n- **Alors** x\n## Annexe\n### CALC-009 — not a criterion\n"
	stabMD = strings.ReplaceAll(strings.Replace(testsMD, "`a_test.go` › `TestCALC003_C`", "`d_test.go` › `TestCALC003_C`", 1), "`a_test.go`", "`p_test.go`")
	pTests = "package x\nfunc TestCALC001_A(t *testing.T) {}\nfunc TestCALC002_B(t *testing.T) {}\n"
	dTests = "package x\nfunc TestCALC003_C(t *testing.T) {}\n"
	all3   = []tc{{"TestCALC001_A", "pass"}, {"TestCALC002_B", "pass"}, {"TestCALC003_C", "pass"}}
	// greenAll reports the three criteria green.
	greenAll = junit(all3...)
	charTask = "mkdir -p .tdd\ns=$(cat src/state)\ncp tests/$s.xml .tdd/junit.xml\n" +
		"[ \"$s\" = slow ] && sleep 3\n[ \"$s\" = fail ] && exit 1\nexit 0\n"
	charReports = files{
		"tests/pass.xml": greenAll,
		"tests/slow.xml": greenAll,
		"tests/fail.xml": junit(tc{"TestCALC001_A", "pass"}, tc{"TestCALC002_B", "pass"}, tc{"TestCALC003_C", "fail"}),
		"tests/none.xml": junit(tc{"TestCALC001_A", "pass"}, tc{"TestCALC002_B", "pass"}),
	}
	divOnly    = "## div — Division\n" + entry("CALC-003", "c", "d_test.go", "TestCALC003_C")
	powPartial = "## pow — Power\n" + entry("CALC-002", "b", "p_test.go", "TestCALC002_B") + "\n" + divOnly
)

const defaultNeutral = "diff --git a/src/note.txt b/src/note.txt\n--- a/src/note.txt\n+++ b/src/note.txt\n@@ -1 +1 @@\n-note\n+neutral\n"

type baseOpts struct {
	rc    *int
	md    string // default testsMD
	extra string
	mise  string
}

func (r *repo) baseCommit(report string, o baseOpts) string {
	r.t.Helper()
	if o.md == "" {
		o.md = testsMD
	}
	if o.mise == "" {
		o.mise = miseToml(miseOpts{})
	}
	return r.commit(files{
		"TESTS.md":    o.md,
		"a_test.go":   goTests,
		"mise.toml":   o.mise,
		"run_test.sh": taskScript("report.xml", o.rc, o.extra),
		"report.xml":  report,
	}, "test(div)")
}

type greenOpts struct {
	featFiles    files
	featReport   string
	checkRC      int
	featMise     string
	md           string
	task         string
	checkDepends string
	testDepends  bool
}

// greenSetup makes a red acceptance commit then a feat commit; it returns both.
func (r *repo) greenSetup(o greenOpts) (string, string) {
	r.t.Helper()
	test := r.baseCommit(junit(tc{"TestCALC003_C", "fail"}), baseOpts{md: o.md,
		mise: miseToml(miseOpts{task: o.task, checkDepends: o.checkDepends, testDepends: o.testDepends})})
	f := merge(files{"impl.go": "package x\n"}, o.featFiles)
	f["green.xml"] = o.featReport
	if o.featReport == "" {
		f["green.xml"] = greenAll
	}
	if o.featMise != "" {
		f["mise.toml"] = o.featMise
	} else if o.checkRC != 0 {
		f["mise.toml"] = miseToml(miseOpts{checkRC: o.checkRC})
	}
	return test, r.commit(f, "feat(div)")
}

// neutral says which neutral change the proofs directory gets: by default a
// valid one for each judged criterion; none leaves them out; change replaces
// CALC-003's by that change.
type neutral struct {
	none   bool
	change files
}

type greenRun struct {
	// noExempt drops --proof-exempt.
	noExempt bool
	// proofs adds --proofs <the mutants directory>, prepared as char does.
	proofs  bool
	neutral neutral
	options []string
}

func (r *repo) runGreen(test, feat string, o greenRun) (int, string) {
	r.t.Helper()
	return r.gate(r.greenArgs(test, feat, o)...)
}

// refusedGreen runs `harness gate green` and returns its exit code and what it
// wrote on its standard error.
func (r *repo) refusedGreen(test, feat string, o greenRun) (int, string) {
	r.t.Helper()
	args := append(append([]string{"gate"}, r.greenArgs(test, feat, o)...), "--repo", r.dir)
	code, _, stderr := runBoth(r.t, r.dir, nil, harness, args...)
	return code, stderr
}

func (r *repo) greenArgs(test, feat string, o greenRun) []string {
	r.t.Helper()
	extra := append([]string{}, o.options...)
	if o.proofs {
		r.prepareProofs(o.neutral, "CALC-003")
		extra = append(extra, "--proofs", r.mutants)
	}
	if !o.noExempt {
		extra = append(extra, "--proof-exempt")
	}
	return append([]string{"green", "--slug", "div", "--test", test, "--feat", feat, "--task", "test"}, extra...)
}

// accCommit makes an acceptance commit holding test paths only; the spec and
// the task script live in a preceding base commit.
func (r *repo) accCommit(report string, extra files) string {
	r.t.Helper()
	if !exists(filepath.Join(r.dir, "run_test.sh")) {
		r.commit(files{"run_test.sh": taskScript("tests/report.xml", nil, ""), "spec.md": spec}, "base")
	}
	return r.commit(merge(files{"TESTS.md": testsMD, "a_test.go": goTests, "mise.toml": miseToml(miseOpts{}), "tests/report.xml": report}, extra),
		"test(div): acceptance")
}

func (r *repo) stabilitySetup(testEditsP bool, featFiles files) (string, string) {
	r.t.Helper()
	r.commit(files{"TESTS.md": stabMD, "p_test.go": pTests, "d_test.go": dTests, "mise.toml": miseToml(miseOpts{}),
		"run_test.sh": taskScript("report.xml", nil, ""), "report.xml": junit(tc{"TestCALC001_A", "pass"})}, "base")
	f := files{"report.xml": junit(tc{"TestCALC003_C", "fail"})}
	if testEditsP {
		f["p_test.go"] = strings.Replace(pTests, "TestCALC001_A(t *testing.T) {}", "TestCALC001_A(t *testing.T) { _ = 1 }", 1)
	}
	test := r.commit(f, "test(div): acceptance")
	return test, r.commit(merge(files{"green.xml": greenAll, "impl.go": "package x\n"}, featFiles), "feat(div)")
}

// specFor is spec with, for each criterion whose entry in md has a sub-list
// Alors, as many clauses as that entry has.
func specFor(md string) string {
	s := spec
	for _, id := range []string{"CALC-001", "CALC-002", "CALC-003"} {
		marker := "### " + id + " — "
		_, after, found := strings.Cut(md, marker)
		if !found {
			continue
		}
		lines := strings.Split(after, "\n")
		n := 1
		for i, line := range lines {
			if strings.TrimSpace(line) == "- **Alors**" {
				n = 0
				for i+1+n < len(lines) && strings.HasPrefix(lines[i+1+n], "    - ") {
					n++
				}
				break
			}
		}
		if n != 1 {
			clauses := ""
			for k := range n {
				clauses += fmt.Sprintf("  - clause %d\n", k)
			}
			head, tail, _ := strings.Cut(s, marker)
			s = head + marker + strings.Replace(tail, "- **Alors** x\n", "- **Alors**\n"+clauses, 1)
		}
	}
	return s
}

type charOpts struct {
	state    string // default "pass"
	md       string // default stabMD
	task     string // default charTask
	miseTask string
	mise     string
	// files are added to the base commit, accFiles to the acceptance commit.
	files, accFiles files
	// baseMD is the TESTS.md of the base commit (default: pow's section of stabMD).
	baseMD string
}

// charSetup makes a base commit (pow's tests, task, spec, src/state), then an
// acceptance commit whose tests are already green; it returns the latter.
func (r *repo) charSetup(o charOpts) string {
	r.t.Helper()
	if o.state == "" {
		o.state = "pass"
	}
	if o.md == "" {
		o.md = stabMD
	}
	if o.task == "" {
		o.task = charTask
	}
	if o.mise == "" {
		o.mise = miseToml(miseOpts{task: o.miseTask})
	}
	if o.baseMD == "" {
		o.baseMD = stabMD[:strings.Index(stabMD, "## div")]
	}
	r.commit(merge(files{"run_test.sh": o.task, "spec.md": specFor(o.md), "src/state": o.state + "\n", "src/note.txt": "note\n",
		".gitignore": ".tdd\n", "TESTS.md": o.baseMD, "p_test.go": pTests, "mise.toml": o.mise}, o.files), "base")
	return r.commit(merge(files{"TESTS.md": o.md, "d_test.go": dTests}, charReports, o.accFiles), "test(div): acceptance")
}

// makePatch writes <mutants>/<id>.patch, the diff of the files against HEAD, leaving the tree clean.
func (r *repo) makePatch(id string, f files) { r.t.Helper(); r.patch(id, f, false) }

// patch does makePatch; bare drops the `diff --git` and `index` header lines.
func (r *repo) patch(id string, f files, bare bool) {
	r.t.Helper()
	for name, content := range f {
		r.write(name, content)
	}
	r.sh("git", "add", "-A")
	_, diff := run(r.t, r.dir, nil, "git", "diff", "--cached")
	r.sh("git", "reset", "-q", "--hard")
	r.sh("git", "clean", "-fdq")
	if bare {
		kept := ""
		for _, ln := range strings.SplitAfter(diff, "\n") {
			if !strings.HasPrefix(ln, "diff --git") && !strings.HasPrefix(ln, "index ") {
				kept += ln
			}
		}
		diff = kept
	}
	write(r.t, filepath.Join(r.mutants, id+".patch"), diff)
}

// prepareProofs completes the proofs directory with a neutral change for each
// criterion of ac (comma list), unless one is already there.
func (r *repo) prepareProofs(n neutral, ac string) {
	r.t.Helper()
	switch {
	case n.change != nil:
		r.makePatch("CALC-003.neutral", n.change)
	case !n.none:
		for _, id := range strings.Split(ac, ",") {
			if path := filepath.Join(r.mutants, id+".neutral.patch"); !exists(path) {
				write(r.t, path, defaultNeutral)
			}
		}
	}
}

type charRun struct {
	neutral neutral
	task    string // default "test"
	ac      string // default "CALC-003"
}

// char runs characterize on the criteria of ac, with the mutants directory as proofs.
func (r *repo) char(sha string, o charRun, extra ...string) (int, string) {
	r.t.Helper()
	if o.task == "" {
		o.task = "test"
	}
	if o.ac == "" {
		o.ac = "CALC-003"
	}
	r.prepareProofs(o.neutral, o.ac)
	return r.gate(append([]string{"characterize", "--slug", "div", "--test", sha, "--task", o.task, "--spec", "spec.md",
		"--ac", o.ac, "--mutants", r.mutants}, extra...)...)
}

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

// lastIs fails the test unless the last line is the expected one.
func lastIs(t *testing.T, out, want string) {
	t.Helper()
	if last(out) != want {
		t.Errorf("last line %q, want %q:\n%s", last(out), want, out)
	}
}

// grep keeps the lines that contain s.
func grep(out, s string) []string {
	var kept []string
	for _, l := range lines(out) {
		if strings.Contains(l, s) {
			kept = append(kept, l)
		}
	}
	return kept
}
