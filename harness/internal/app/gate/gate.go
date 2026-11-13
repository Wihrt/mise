package gate

import (
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"
	"sync"

	"github.com/Wihrt/mise/harness/internal/domain/tdd"
)

// Options are the arguments of one gate run.
type Options struct {
	Mode            string // red, green, characterize or replay
	Slug            string
	Test, Feat      string
	Task            string
	Repo            string
	Spec            string
	AC              []string
	ACChanged       []string
	Replaces        []string
	AllowCompileRed bool
	Proofs          string
	ProofExempt     bool
	NoNeutral       bool
	Jobs            int
	Format          string
	Report          string
	Timeout         int
	Base, Head      string
	AcceptLost      []string
}

// Gate runs the gates.
type Gate struct {
	git       Git
	files     Files
	tasks     Tasks
	taskFile  TaskFile
	stdout    io.Writer
	worktrees sync.Mutex // git worktree add/remove, one at a time
}

// New builds the gate over its ports.
func New(git Git, files Files, tasks Tasks, taskFile TaskFile, stdout io.Writer) *Gate {
	return &Gate{git: git, files: files, tasks: tasks, taskFile: taskFile, stdout: stdout}
}

// failure is a usage or environment problem: the gate answers FAIL with exit code 2.
type failure struct{ msg string }

func (f failure) Error() string { return f.msg }

var errNoIDs = failure{"no ids"}

func failf(format string, a ...any) error { return failure{fmt.Sprintf(format, a...)} }

// run is one gate run. lines, when set, captures the output of a proof judged
// in parallel, printed when its turn comes.
type run struct {
	g      *Gate
	o      Options
	lines  *[]string
	judged map[string]judgement
}

func (r *run) out(msg string) {
	if r.lines != nil {
		*r.lines = append(*r.lines, msg)
		return
	}
	_, _ = fmt.Fprintln(r.g.stdout, msg)
}

func (r *run) outf(format string, a ...any) { r.out(fmt.Sprintf(format, a...)) }

// Run runs the gate and returns its exit code: 0 PASS, 1 FAIL, 2 usage or
// environment error. The last output line is PASS or FAIL.
func (g *Gate) Run(o Options) int {
	r := &run{g: g, o: o}
	modes := map[string]func() (bool, error){
		"red": r.red, "green": r.green, "characterize": r.characterize, "replay": r.replay,
	}
	ok, err := modes[o.Mode]()
	if err != nil {
		if !errors.Is(err, errNoIDs) {
			r.out("ERROR: " + err.Error())
		}
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

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func normSet(ids []string) map[string]bool {
	set := map[string]bool{}
	for _, i := range ids {
		set[tdd.Norm(i)] = true
	}
	return set
}

func lineSet(text string) map[string]bool {
	set := map[string]bool{}
	for _, l := range strings.Split(text, "\n") {
		if l != "" {
			set[l] = true
		}
	}
	return set
}

func (r *run) status(repo string, args ...string) string {
	_, st := r.g.git.Run(repo, append([]string{"status", "--short"}, args...)...)
	return st
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

var defaultReport = map[string]string{"junit": ".tdd/junit.xml", "tofu-json": ".tdd/tofu.json"}

// Formats lists the report formats, in the order they are looked for.
var Formats = []string{"junit", "tofu-json"}

func parent(path string) string {
	if i := strings.LastIndex(path, "/"); i > 0 {
		return path[:i]
	}
	return "."
}

// runTask runs the judged task and reads the report it wrote. A stale report is deleted first.
func (r *run) runTask(repo string) (int, string, tdd.Report, error) {
	rel := func(format string) string {
		if r.o.Report != "" {
			return r.o.Report
		}
		return defaultReport[format]
	}
	path := func(format string) string {
		if p := rel(format); strings.HasPrefix(p, "/") {
			return p
		}
		return repo + "/" + rel(format)
	}
	formats := Formats
	if r.o.Format != "" {
		formats = []string{r.o.Format}
	}
	for _, f := range formats {
		if err := r.g.files.MkdirAll(parent(path(f))); err != nil {
			return 0, "", tdd.Report{}, failure{err.Error()}
		}
		r.g.files.Remove(path(f))
	}
	rc, output, err := r.runMise(repo, r.o.Task)
	if err != nil {
		return 0, "", tdd.Report{}, err
	}
	format := formats[0]
	for _, f := range formats {
		if r.g.files.IsFile(path(f)) {
			format = f
			break
		}
	}
	rep := tdd.Report{State: tdd.ReportMissing}
	if r.g.files.IsFile(path(format)) {
		data, _ := r.g.files.Read(path(format))
		if format == "tofu-json" {
			rep = tdd.ParseTofuJSON(data)
		} else {
			rep = tdd.ParseJUnit(data)
		}
	}
	if rep.State == tdd.ReportMissing {
		r.outf("REPORT missing: %s n'a pas écrit %s", r.o.Task, rel(format))
	} else {
		r.outf("REPORT %s: %s, %d testcases", path(format), rep.State, len(rep.Cases))
	}
	return rc, output, rep, nil
}

// citedFiles lists, sorted, the test files TESTS.md cites that exist in the repository.
func (r *run) citedFiles(text, repo string) []string {
	var files []string
	for _, f := range tdd.CitedTests(text) {
		if r.g.files.IsFile(repo + "/" + f) {
			files = append(files, f)
		}
	}
	sort.Strings(files)
	return files
}

// scan reads the tests of the files, in the work tree or at rev.
func (r *run) scan(repo string, files []string, rev string) ([]tdd.Entry, map[string]string) {
	var entries []tdd.Entry
	pre := map[string]string{}
	for _, f := range files {
		var content string
		if rev != "" {
			_, content = r.g.git.Run(repo, "show", rev+":"+f)
		} else {
			content, _ = r.g.files.Read(repo + "/" + f)
		}
		e, p := tdd.Scan(f, content)
		entries = append(entries, e...)
		pre[f] = p
	}
	return entries, pre
}

type prepared struct {
	repo, text string
	ids        []string
	entries    []tdd.Entry
	pre        map[string]string
}

func (r *run) prepare() (prepared, error) {
	repo := r.g.files.Resolve(r.o.Repo)
	testsMD := repo + "/TESTS.md"
	if !r.g.files.IsFile(testsMD) {
		return prepared{}, failf("%s not found", testsMD)
	}
	if rc, _ := r.g.git.Run(repo, "rev-parse", "--git-dir"); rc != 0 {
		return prepared{}, failf("%s is not a git repository", repo)
	}
	text, err := r.g.files.Read(testsMD)
	if err != nil {
		return prepared{}, failure{err.Error()}
	}
	ids := tdd.SectionIDs(text, r.o.Slug)
	if len(ids) == 0 {
		r.out("NO IDS FOUND")
		return prepared{}, errNoIDs
	}
	sorted := append([]string(nil), ids...)
	sort.Strings(sorted)
	r.out("IDS: " + strings.Join(sorted, " "))
	entries, pre := r.scan(repo, r.citedFiles(text, repo), "")
	return prepared{repo, text, ids, entries, pre}, nil
}

func (r *run) checkEntries(text string) bool {
	ids := tdd.IncompleteEntries(text, r.o.Slug)
	for _, i := range ids {
		r.outf("INCOMPLETE %s: Étant donné/Quand/Alors manquant", i)
	}
	return len(ids) == 0
}

func (r *run) checkCriteria(repo string, ids []string, text string) (bool, error) {
	sp := r.o.Spec
	if !strings.HasPrefix(sp, "/") {
		sp = repo + "/" + sp
	}
	if !r.g.files.IsFile(sp) {
		return false, failf("%s not found", sp)
	}
	spec, err := r.g.files.Read(sp)
	if err != nil {
		return false, failure{err.Error()}
	}
	crit := tdd.SpecCriteria(spec)
	r.out("CRITERIA: " + strings.Join(crit, " "))
	specN, acN, secN := normSet(crit), normSet(r.o.AC), normSet(ids)
	ok := true
	if strings.TrimSpace(r.status(repo, "--", sp)) != "" {
		r.outf("SPEC NOT COMMITTED %s", r.o.Spec)
		ok = false
	}
	for _, i := range r.o.AC {
		if !specN[tdd.Norm(i)] {
			r.outf("NOT IN SPEC %s", i)
			ok = false
		}
		if !secN[tdd.Norm(i)] {
			r.outf("NOT IN SECTION %s", i)
			ok = false
		}
	}
	for _, i := range ids {
		if !acN[tdd.Norm(i)] {
			r.outf("NOT IN --ac %s", i)
			ok = false
		}
		n, m := tdd.ClauseCount(tdd.EntryText(text, i)), tdd.ClauseCount(tdd.EntryText(spec, i))
		if n != m {
			r.outf("CLAUSES %s: %d in TESTS.md, %d in spec", i, n, m)
			ok = false
		}
	}
	return ok, nil
}

func (r *run) checkAcceptancePaths(repo, sha string) bool {
	ok := true
	_, files := r.g.git.Run(repo, "show", "--name-only", "--format=", sha)
	for _, f := range strings.Split(files, "\n") {
		if f != "" && !tdd.IsTestPath(f) {
			r.outf("NON-TEST FILE IN ACCEPTANCE COMMIT: %s", f)
			ok = false
		}
	}
	return ok
}

func candidates(entries []tdd.Entry) map[string]bool {
	names := map[string]bool{}
	for _, e := range entries {
		if e.Cand != "" {
			names[e.Cand] = true
		}
	}
	return names
}

func (r *run) red() (bool, error) {
	p, err := r.prepare()
	if err != nil {
		return false, err
	}
	ok := r.checkEntries(p.text)
	acceptance := r.o.Spec != ""
	if acceptance {
		criteria, err := r.checkCriteria(p.repo, p.ids, p.text)
		if err != nil {
			return false, err
		}
		ok = criteria && ok
		if !r.checkAcceptancePaths(p.repo, r.o.Test) {
			ok = false
		}
	}
	names := candidates(p.entries)
	r.out("git status --short:\n" + strings.TrimRight(r.status(p.repo), "\n"))
	for _, i := range p.ids {
		if !names[tdd.Norm(i)] {
			r.outf("MISSING %s", i)
			ok = false
		}
	}
	rc, output, rep, err := r.runTask(p.repo)
	if err != nil {
		return false, err
	}
	if rc == 0 {
		r.out("TASK PASSED, NOT RED")
		ok = false
	}
	errs := tdd.CompileErrors(output + "\n" + rep.Diag)
	for _, e := range errs {
		r.out("COMPILE OR COLLECT ERROR: " + strings.TrimSpace(e.Line))
	}
	res := tdd.ResultsByID(rep)
	for _, i := range p.ids {
		n := tdd.Norm(i)
		cases := res[n]
		if c, failed := tdd.FirstFailed(cases); failed {
			r.outf("RED %s: %s", i, c.Name)
			continue
		}
		if len(cases) > 0 {
			r.outf("NOT RED %s: %s %s", i, cases[0].Name, cases[0].Status)
			ok = false
			continue
		}
		sym, entry, hit := excuse(p, n, errs)
		switch {
		case hit && acceptance && !r.o.AllowCompileRed:
			r.outf("NOT RED %s: compile/collect error excuse refused in acceptance mode", i)
			ok = false
		case hit:
			r.outf("RED %s: no result (report %s), excused by compile/collect error: `%s` used in %s:%d",
				i, rep.State, sym, entry.File, entry.Line)
		default:
			r.outf("NOT RED %s: no result in report (%s) and no compile/collect error naming a symbol of its test", i, rep.State)
			ok = false
		}
	}
	return ok, nil
}

// excuse finds a compile error naming a symbol the test of criterion n uses.
func excuse(p prepared, n string, errs []tdd.CompileError) (string, tdd.Entry, bool) {
	for _, e := range p.entries {
		if e.Cand != n {
			continue
		}
		for _, ce := range errs {
			for _, s := range ce.Symbols {
				if tdd.ContainsWord(e.Body, s) || tdd.ContainsWord(p.pre[e.File], s) {
					return s, e, true
				}
			}
		}
	}
	return "", tdd.Entry{}, false
}
