package gate

import (
	"sort"
	"strings"

	"github.com/Wihrt/mise/harness/internal/domain/tdd"
)

// characterize proves tests that are already green: per criterion a mutant must make its test fail.
func (r *run) characterize() (bool, error) {
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
	criteria, err := r.checkCriteria(repo, p.ids, text)
	if err != nil {
		return false, err
	}
	ok = criteria && ok
	if !r.checkOrphans(repo, text, before) {
		ok = false
	}
	if !r.checkAcceptancePaths(repo, r.o.Test) {
		ok = false
	}
	if !r.checkWired(repo, r.o.Test, text) {
		ok = false
	}
	if st := r.status(repo); strings.TrimSpace(st) != "" {
		r.out("TREE NOT CLEAN:\n" + strings.TrimRight(st, "\n"))
		return false, nil
	}
	_, head := r.g.git.Run(repo, "rev-parse", "HEAD")
	_, test := r.g.git.Run(repo, "rev-parse", "--verify", "-q", r.o.Test+"^{commit}")
	if strings.TrimSpace(head) != strings.TrimSpace(test) {
		r.outf("HEAD IS NOT --test: %s (test %s)", strings.TrimSpace(head), r.o.Test)
		return false, nil
	}
	rc, _, rep, err := r.runTask(repo)
	if err != nil {
		return false, err
	}
	if rc == 124 {
		return false, failf("baseline run timed out")
	}
	res, labels := tdd.ResultsByID(rep), tdd.IDLabels(text)
	keep := normSet(p.ids)
	for n := range normSet(r.o.AC) {
		keep[n] = true
	}
	skipped := tdd.OutOfScope(text, r.o.Task, keep)
	r.reportOutOfScope(skipped, labels)
	green := true
	for _, n := range sortedKeys(tdd.AllIDs(text)) {
		if _, out := skipped[n]; out {
			continue
		}
		cases := res[n]
		if len(cases) == 0 {
			r.outf("BASELINE NOT GREEN %s: no result (%s)", labels[n], rep.State)
			green = false
		} else if bad, has := tdd.FirstNot(cases, tdd.Pass); has {
			r.outf("BASELINE NOT GREEN %s: %s %s", labels[n], bad.Name, bad.Status)
			green = false
		}
	}
	if !green {
		return false, nil
	}
	proved, dirty, err := r.runProofs(repo, text, r.o.AC)
	if err != nil {
		return false, err
	}
	if dirty {
		return false, nil
	}
	ok = ok && proved
	r.o.Feat = r.o.Test // nothing is committed: stability covers the acceptance commit
	if !r.checkStability(repo, text, before) {
		ok = false
	}
	return ok, nil
}

// killed applies the patch ("" for none) on a throwaway worktree of sha and
// runs the task: true when the task fails.
func (r *run) killed(repo, sha, patch string) (bool, error) {
	tmp, cleanup, err := r.g.files.TempDir("tdd-replay-")
	if err != nil {
		return false, failure{err.Error()}
	}
	defer cleanup()
	wt := tmp + "/wt"
	r.g.git.Run(repo, "worktree", "add", "--detach", wt, sha)
	defer r.g.git.Run(repo, "worktree", "remove", "--force", wt)
	if patch != "" {
		if _, err := r.applyMutant(wt, patch); err != nil {
			return false, err
		}
	}
	rc, _, err := r.runMise(wt, r.o.Task)
	return rc != 0, err
}

// replay runs the mutants of a proofs directory on two commits and refuses a mutant the head no longer kills.
func (r *run) replay() (bool, error) {
	repo := r.g.files.Resolve(r.o.Repo)
	for _, side := range [][2]string{{"base", r.o.Base}, {"head", r.o.Head}} {
		red, err := r.killed(repo, side[1], "")
		if err != nil {
			return false, err
		}
		if red {
			r.outf("BASELINE NOT GREEN %s", side[0])
			return false, nil
		}
	}
	ok := true
	accepted := map[string]bool{}
	for _, n := range r.o.AcceptLost {
		accepted[n] = true
	}
	patches := r.g.files.Patches(r.g.files.Resolve(r.o.Proofs))
	sort.Strings(patches)
	for _, patch := range patches {
		name := strings.TrimSuffix(patch[strings.LastIndex(patch, "/")+1:], ".patch")
		was, err := r.killed(repo, r.o.Base, patch)
		if err != nil {
			return false, err
		}
		now, err := r.killed(repo, r.o.Head, patch)
		if err != nil {
			return false, err
		}
		switch {
		case was && now:
			r.outf("KEPT %s", name)
		case !was && !now:
			r.outf("NEVER %s", name)
		case now:
			r.outf("GAINED %s", name)
		case accepted[name]:
			r.outf("LOST ACCEPTED %s", name)
		default:
			r.outf("LOST %s", name)
			ok = false
		}
	}
	return ok, nil
}
