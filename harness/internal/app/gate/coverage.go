package gate

import (
	"path"
	"sort"
	"strings"

	"github.com/Wihrt/mise/harness/internal/domain/coverage"
	"github.com/Wihrt/mise/harness/internal/domain/tdd"
)

// coverage runs the task once, reads the reports it wrote and fails for every
// block of production code that no report saw executed.
func (r *run) coverage() (bool, error) {
	repo := r.g.files.Resolve(r.o.Repo)
	rc, listed := r.g.git.Run(repo, "ls-files")
	if rc != 0 {
		return false, failf("%s is not a git repository", repo)
	}
	tracked := map[string]bool{}
	for _, f := range strings.Fields(listed) {
		tracked[f] = true
	}
	abs := func(p string) string {
		if strings.HasPrefix(p, "/") {
			return p
		}
		return repo + "/" + p
	}
	for _, p := range r.o.Coverage {
		r.g.files.Remove(abs(p))
	}
	rc, _, err := r.runMise(repo, r.o.Task)
	if err != nil {
		return false, err
	}
	if rc != 0 {
		r.outf("TASK FAILED %s", r.o.Task)
		return false, nil
	}
	modules, err := r.goModules(repo, tracked)
	if err != nil {
		return false, err
	}
	var measures []coverage.Measure
	for _, p := range r.o.Coverage {
		if !r.g.files.IsFile(abs(p)) {
			r.outf("COVERAGE missing: %s n'a pas écrit %s", r.o.Task, p)
			return false, nil
		}
		data, err := r.g.files.Read(abs(p))
		if err != nil {
			return false, failure{err.Error()}
		}
		m, err := coverage.Parse(data, modules)
		if err != nil {
			return false, failf("%s: %v", p, err)
		}
		r.outf("COVERAGE %s: %d files", p, len(m))
		measures = append(measures, m)
	}
	production := func(f string) bool { return tracked[f] && !tdd.IsTestPath(f) }
	merged := coverage.Merge(measures...)
	findings := merged.Select(production).Findings()
	sources := map[string]string{}
	source := func(f string) (string, error) {
		if s, ok := sources[f]; ok {
			return s, nil
		}
		s, err := r.g.files.Read(abs(f))
		sources[f] = s
		return s, err
	}
	for i, f := range findings {
		s, err := source(f.File)
		if err != nil {
			return false, failure{err.Error()}
		}
		findings[i].Text = coverage.LineText(s, f.Start)
	}
	findings, ignored, err := r.withoutErrorReturns(findings, source)
	if err != nil {
		return false, err
	}
	notJudged := 0
	for f := range tracked {
		ext := path.Ext(f)
		if ext != ".go" && ext != ".py" {
			notJudged++
			if coverage.IsCoverageConfig(f) {
				s, err := source(f)
				if err != nil {
					return false, failure{err.Error()}
				}
				findings = append(findings, coverage.Exclusions(f, s)...)
			}
			continue
		}
		if !production(f) {
			continue
		}
		s, err := source(f)
		if err != nil {
			return false, failure{err.Error()}
		}
		if ext == ".py" {
			findings = append(findings, coverage.Pragmas(f, s)...)
		}
		if _, measured := merged[f]; !measured && (ext == ".py" || coverage.HasFunc(s)) {
			findings = append(findings, coverage.Finding{Kind: "NOT MEASURED", File: f})
		}
	}
	exceptions, err := r.exceptions(abs("COVERAGE.md"))
	if err != nil {
		return false, err
	}
	accepted, refused, stale := exceptions.Apply(findings)
	if r.o.Baseline {
		if debt := exceptions.WithDebt(refused); debt.Markdown() != exceptions.Markdown() {
			if err := r.g.files.Write(abs("COVERAGE.md"), debt.Markdown()); err != nil {
				return false, failure{err.Error()}
			}
			accepted, refused, stale = debt.Apply(findings)
		}
	}
	pass := len(stale) == 0
	all := append(accepted, refused...)
	sort.SliceStable(all, func(i, j int) bool {
		if all[i].File != all[j].File {
			return all[i].File < all[j].File
		}
		return all[i].Start < all[j].Start
	})
	for _, f := range all {
		switch {
		case f.Reason != "" && f.Kind == "NOT MEASURED":
			r.outf("ACCEPTED %s: %s", f.File, f.Reason)
		case f.Reason != "":
			r.outf("ACCEPTED %s:%d-%d: %s", f.File, f.Start, f.End, f.Reason)
		case f.Kind == "UNCOVERED":
			pass = false
			r.outf("UNCOVERED %s:%d-%d", f.File, f.Start, f.End)
		case f.Kind == "NOT MEASURED":
			pass = false
			r.outf("NOT MEASURED %s", f.File)
		default:
			pass = false
			r.outf("%s %s:%d", f.Kind, f.File, f.Start)
		}
	}
	for _, s := range stale {
		r.outf("STALE EXCEPTION %s: %s", s.File, s.Text)
	}
	r.outf("ERROR RETURNS IGNORED %d", ignored)
	r.outf("NOT JUDGED %d files: neither Go nor Python", notJudged)
	return pass, nil
}

// withoutErrorReturns drops the never-executed Go blocks that only return an
// error, and counts them.
func (r *run) withoutErrorReturns(findings []coverage.Finding, source func(string) (string, error)) ([]coverage.Finding, int, error) {
	returns := map[string]map[[2]int]bool{}
	var kept []coverage.Finding
	ignored := 0
	for _, f := range findings {
		if f.Kind != "UNCOVERED" || path.Ext(f.File) != ".go" {
			kept = append(kept, f)
			continue
		}
		if returns[f.File] == nil {
			s, err := source(f.File)
			if err != nil {
				return nil, 0, failure{err.Error()}
			}
			if returns[f.File], err = coverage.ErrorReturns(s); err != nil {
				return nil, 0, failf("%s: %v", f.File, err)
			}
		}
		if returns[f.File][[2]int{f.Start, f.Col}] {
			ignored++
			continue
		}
		kept = append(kept, f)
	}
	return kept, ignored, nil
}

// exceptions reads COVERAGE.md; a missing file holds no exception.
func (r *run) exceptions(file string) (coverage.Exceptions, error) {
	if !r.g.files.IsFile(file) {
		return nil, nil
	}
	md, err := r.g.files.Read(file)
	if err != nil {
		return nil, failure{err.Error()}
	}
	e, err := coverage.ParseExceptions(md)
	if err != nil {
		return nil, failure{err.Error()}
	}
	return e, nil
}

// goModules maps the module path of each tracked go.mod to its directory.
func (r *run) goModules(repo string, tracked map[string]bool) (map[string]string, error) {
	modules := map[string]string{}
	for f := range tracked {
		if path.Base(f) != "go.mod" {
			continue
		}
		text, err := r.g.files.Read(repo + "/" + f)
		if err != nil {
			return nil, failure{err.Error()}
		}
		for _, l := range strings.Split(text, "\n") {
			if mod, ok := strings.CutPrefix(strings.TrimSpace(l), "module "); ok {
				modules[strings.Trim(strings.TrimSpace(mod), `"`)] = path.Dir(f)
				break
			}
		}
	}
	return modules, nil
}
