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
	findings := coverage.Merge(measures...).Select(production).Findings()
	for f := range tracked {
		if !strings.HasSuffix(f, ".py") || !production(f) {
			continue
		}
		source, err := r.g.files.Read(abs(f))
		if err != nil {
			return false, failure{err.Error()}
		}
		findings = append(findings, coverage.Pragmas(f, source)...)
	}
	sort.SliceStable(findings, func(i, j int) bool {
		if findings[i].File != findings[j].File {
			return findings[i].File < findings[j].File
		}
		return findings[i].Start < findings[j].Start
	})
	for _, f := range findings {
		if f.Kind == "UNCOVERED" {
			r.outf("UNCOVERED %s:%d-%d", f.File, f.Start, f.End)
		} else {
			r.outf("%s %s:%d", f.Kind, f.File, f.Start)
		}
	}
	r.out("ERROR RETURNS IGNORED 0")
	return len(findings) == 0, nil
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
