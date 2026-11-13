package gate

import (
	"slices"
	"strings"

	"github.com/Wihrt/mise/harness/internal/domain/tdd"
)

// checkReplaces returns TESTS.md at <test>~1; every --replaces slug must have a section there.
func (r *run) checkReplaces(repo string) (string, error) {
	rc, before := r.g.git.Run(repo, "show", r.o.Test+"~1:TESTS.md")
	if rc != 0 {
		before = ""
	}
	for _, slug := range r.o.Replaces {
		if len(tdd.SectionIDs(before, slug)) == 0 {
			return "", failf("--replaces %s: no such slug in TESTS.md at %s~1", slug, r.o.Test)
		}
	}
	return before, nil
}

func union(sections map[string]map[string]bool) map[string]bool {
	all := map[string]bool{}
	for _, files := range sections {
		for f := range files {
			all[f] = true
		}
	}
	return all
}

// checkOrphans refuses the tests of files cited at <test>~1 and no longer by
// TESTS.md, still on disk, whose ID is absent from TESTS.md: they still run.
func (r *run) checkOrphans(repo, text, before string) bool {
	now := union(tdd.SectionTests(text))
	var files []string
	for _, f := range sortedKeys(union(tdd.SectionTests(before))) {
		if !now[f] && r.g.files.IsFile(repo+"/"+f) {
			files = append(files, f)
		}
	}
	known, labels := tdd.AllIDs(text), tdd.IDLabels(before)
	ok := true
	entries, _ := r.scan(repo, files, "")
	for _, e := range entries {
		if e.Cand != "" && !known[e.Cand] {
			label, has := labels[e.Cand]
			if !has {
				label = e.Cand
			}
			r.outf("ORPHAN TEST %s:%d %s", e.File, e.Line, label)
			ok = false
		}
	}
	return ok
}

// checkStability refuses a change to the acceptance tests of earlier slugs.
func (r *run) checkStability(repo, text, before string) bool {
	replaced := map[string]bool{}
	for _, s := range r.o.Replaces {
		replaced[s] = true
	}
	owners := map[string]map[string]bool{} // earlier test file -> slugs (other than ours) citing it, now or at <test>~1
	for _, cited := range []map[string]map[string]bool{tdd.SectionTests(text), tdd.SectionTests(before)} {
		for slug, files := range cited {
			if slug == r.o.Slug {
				continue
			}
			for f := range files {
				if owners[f] == nil {
					owners[f] = map[string]bool{}
				}
				owners[f][slug] = true
			}
		}
	}
	changed := map[string]bool{}
	if rc, diff := r.g.git.Run(repo, "diff", "--name-only", r.o.Test+"~1", r.o.Feat); rc == 0 {
		changed = lineSet(diff)
	}
	allowed := map[string]string{}
	for _, i := range r.o.ACChanged {
		allowed[tdd.Norm(i)] = i
	}
	ok := true
	for _, f := range sortedKeys(owners) {
		if !changed[f] {
			continue
		}
		all := true
		for slug := range owners[f] {
			all = all && replaced[slug]
		}
		if all {
			for _, slug := range sortedKeys(owners[f]) {
				r.outf("REPLACED %s: %s", slug, f)
			}
			continue
		}
		hit := ""
		if r.g.files.IsFile(repo + "/" + f) {
			entries, _ := r.scan(repo, []string{f}, "")
			for _, e := range entries {
				if id, in := allowed[e.Cand]; in {
					hit = id
					break
				}
			}
		}
		if hit != "" {
			r.outf("CRITERION CHANGE ALLOWED: %s (%s)", f, hit)
		} else {
			r.outf("EARLIER ACCEPTANCE TEST MODIFIED: %s", f)
			ok = false
		}
	}
	for _, slug := range sortedKeys(replaced) {
		for _, i := range sortedKeys(tdd.ReplacedIDs(before, text, []string{slug})) {
			r.outf("REPLACED %s: %s", slug, i)
		}
	}
	was, after, labels := tdd.AllIDs(before), tdd.AllIDs(text), tdd.IDLabels(before)
	now := map[string][]string{}
	entries, _ := r.scan(repo, r.citedFiles(text, repo), "")
	for _, e := range entries {
		now[e.Cand] = append(now[e.Cand], e.Body)
	}
	old, _ := r.scan(repo, sortedKeys(union(tdd.SectionTests(before))), r.o.Test+"~1")
	for _, e := range old {
		_, changedCriterion := allowed[e.Cand]
		kept := was[e.Cand] && after[e.Cand] && !changedCriterion
		if kept && !slices.Contains(now[e.Cand], e.Body) {
			r.outf("KEPT TEST MODIFIED %s %s", e.File, labels[e.Cand])
			ok = false
		}
	}
	return ok
}

// wiredTasks lists the tasks a commit-hook or CI file of the commit runs with
// `mise run`, plus their direct `depends` in mise.toml.
func (r *run) wiredTasks(repo, sha string) map[string]bool {
	_, names := r.g.git.Run(repo, "ls-tree", "-r", "--name-only", sha)
	wired := map[string]bool{}
	for _, f := range tdd.Lines(names) {
		if tdd.IsWiringFile(f) {
			_, content := r.g.git.Run(repo, "show", sha+":"+f)
			for _, t := range tdd.MiseRunTasks(content) {
				wired[t] = true
			}
		}
	}
	_, toml := r.g.git.Run(repo, "show", sha+":mise.toml")
	tasks := r.g.taskFile.Depends(toml)
	for _, t := range sortedKeys(wired) {
		for _, d := range tasks[t] {
			for n := range tasks {
				if tdd.Fnmatch(n, d) {
					wired[n] = true
				}
			}
		}
	}
	return wired
}

func (r *run) checkWired(repo, sha, text string) bool {
	task := tdd.WiredCommand(text, r.o.Slug)
	if task != "" && !r.wiredTasks(repo, sha)[task] {
		r.outf("NOT WIRED %s", task)
		return false
	}
	return true
}

func (r *run) miseNonTools(repo, sha string) []string {
	rc, txt := r.g.git.Run(repo, "show", sha+":mise.toml")
	if rc != 0 {
		return nil
	}
	return tdd.MiseNonTools(txt)
}

func (r *run) green() (bool, error) {
	p, err := r.prepare()
	if err != nil {
		return false, err
	}
	repo, text := p.repo, p.text
	before, err := r.checkReplaces(repo)
	if err != nil {
		return false, err
	}
	ok := r.checkEntries(text)
	ok = r.checkOrphans(repo, text, before) && ok
	ok = r.checkWired(repo, r.o.Feat, text) && ok
	_, head := r.g.git.Run(repo, "rev-parse", "HEAD")
	r.outf("HEAD %s (feat %s)", strings.TrimSpace(head), r.o.Feat)
	_, tfiles := r.g.git.Run(repo, "show", "--name-only", "--format=", r.o.Test)
	_, diff := r.g.git.Run(repo, "diff", "--name-only", r.o.Test, r.o.Feat)
	prot, changed := lineSet(tfiles), lineSet(diff)
	delete(prot, "mise.toml")
	for _, f := range sortedKeys(prot) {
		if changed[f] {
			r.outf("PROTECTED FILE MODIFIED: %s", f)
			ok = false
		}
	}
	if changed["TESTS.md"] && !prot["TESTS.md"] {
		r.out("PROTECTED FILE MODIFIED: TESTS.md")
		ok = false
	}
	if !slices.Equal(r.miseNonTools(repo, r.o.Test), r.miseNonTools(repo, r.o.Feat)) {
		r.out("PROTECTED FILE MODIFIED: mise.toml (tasks)")
		ok = false
	}
	if !r.checkStability(repo, text, before) {
		ok = false
	}
	names, idsAll, own := candidates(p.entries), tdd.AllIDs(text), normSet(p.ids)
	skipped := tdd.OutOfScope(text, r.o.Task, own)
	for _, i := range sortedKeys(idsAll) {
		if _, exempt := skipped[i]; !names[i] && !exempt {
			r.outf("ID WITHOUT TEST: %s", i)
			ok = false
		}
	}
	for _, i := range sortedKeys(names) {
		if !idsAll[i] {
			r.outf("TEST WITHOUT ID: %s", i)
			ok = false
		}
	}
	for _, h := range tdd.IncompleteHeadings(text) {
		r.outf("INCOMPLETE: %s", h)
		ok = false
	}
	_, _, rep, err := r.runTask(repo)
	if err != nil {
		return false, err
	}
	res := tdd.ResultsByID(rep)
	if rep.State != tdd.ReportOK {
		r.outf("REPORT %s: no per-test result", rep.State)
		ok = false
	}
	for _, i := range p.ids {
		cases := res[tdd.Norm(i)]
		if len(cases) == 0 {
			r.outf("NO RESULT %s", i)
			ok = false
			continue
		}
		if bad, has := tdd.FirstNot(cases, tdd.Pass); has {
			r.outf("NOT GREEN %s: %s %s", i, bad.Name, bad.Status)
			ok = false
		} else {
			r.outf("GREEN %s: %d testcase(s)", i, len(cases))
		}
	}
	labels := tdd.IDLabels(text)
	r.reportOutOfScope(skipped, labels)
	for _, n := range sortedKeys(idsAll) {
		if _, out := skipped[n]; own[n] || out {
			continue
		}
		cases := res[n]
		if len(cases) == 0 {
			r.outf("NO RESULT %s", labels[n])
			ok = false
			continue
		}
		if bad, has := tdd.FirstNot(cases, tdd.Pass); has {
			r.outf("REGRESSION %s: %s %s", labels[n], bad.Name, bad.Status)
			ok = false
		}
	}
	crc, _, err := r.runMise(repo, "check")
	if err != nil {
		return false, err
	}
	if crc != 0 {
		r.out("CHECK FAILED")
		ok = false
	}
	switch st := r.status(repo); {
	case r.o.Proofs != "" && strings.TrimSpace(st) != "":
		r.out("TREE NOT CLEAN:\n" + strings.TrimRight(st, "\n"))
		ok = false
	case r.o.Proofs != "":
		proved, _, err := r.runProofs(repo, text, p.ids)
		if err != nil {
			return false, err
		}
		ok = ok && proved
		if r.o.NoNeutral {
			r.out("NEUTRAL SKIPPED")
		}
		if r.o.NoMinimal {
			r.out("MINIMAL SKIPPED")
		} else {
			minimal, err := r.runMinimal(repo)
			if err != nil {
				return false, err
			}
			ok = ok && minimal
		}
	case !r.o.ProofExempt:
		r.out("PROOF MISSING: --proofs DIR or --proof-exempt")
		ok = false
	}
	return ok, nil
}

func (r *run) reportOutOfScope(skipped map[string]string, labels map[string]string) {
	for _, n := range sortedKeys(skipped) {
		r.outf("OUT OF SCOPE %s: %s", labels[n], skipped[n])
	}
}
