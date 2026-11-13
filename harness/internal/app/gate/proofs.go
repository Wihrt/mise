package gate

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/Wihrt/mise/harness/internal/domain/tdd"
)

var createMode = regexp.MustCompile(`(?m)^ create mode \d+ (.+)$`)

// applyMutant applies the patch to the working tree and returns the files it creates.
func (r *run) applyMutant(repo, patch string) ([]string, error) {
	_, summary := r.g.git.Run(repo, "apply", "--summary", patch)
	var created []string
	for _, m := range createMode.FindAllStringSubmatch(summary, -1) {
		created = append(created, m[1])
	}
	if rc, _ := r.g.git.Run(repo, "apply", patch); rc != 0 {
		return nil, failf("git apply failed: %s", patch)
	}
	return created, nil
}

func (r *run) restoreTree(repo string, created []string) {
	r.g.git.Run(repo, "checkout", "--", ".")
	if len(created) > 0 {
		r.g.git.Run(repo, append([]string{"clean", "-fdq", "--"}, created...)...)
	}
	for _, f := range created {
		r.g.files.Remove(repo + "/" + f)
	}
}

// mutantProblem tells why the patch cannot be used as a mutant, "" when it can.
func (r *run) mutantProblem(repo, patch string) string {
	if rc, _ := r.g.git.Run(repo, "apply", "--check", patch); rc != 0 {
		return "patch does not apply"
	}
	_, numstat := r.g.git.Run(repo, "apply", "--numstat", patch)
	for _, line := range tdd.Lines(numstat) {
		parts := strings.SplitN(line, "\t", 3)
		if f := parts[len(parts)-1]; f != "mise.toml" && tdd.IsTestPath(f) {
			return "touches test path " + f
		}
	}
	old, _ := r.g.files.Read(repo + "/mise.toml")
	text, _ := r.g.files.Read(patch)
	return tdd.MisePatchProblem(tdd.Lines(old), text, r.o.Task)
}

// judgement is what applying one patch and running the task shows.
type judgement struct {
	problem string
	results map[string][]tdd.Case
	status  string // git status of the tree once restored
	lines   []string
}

// runPatch applies the patch, runs the task and restores the tree.
func (r *run) runPatch(repo, patch, kind, tag string) (judgement, error) {
	if j, done := r.judged[patch]; done {
		for _, line := range j.lines {
			r.out(line)
		}
		return j, nil
	}
	if problem := r.mutantProblem(repo, patch); problem != "" {
		return judgement{problem: problem}, nil
	}
	created, err := r.applyMutant(repo, patch)
	if err != nil {
		return judgement{}, err
	}
	r.outf("%s %s: patch applied", kind, tag)
	rc, _, rep, err := r.runTask(repo)
	r.restoreTree(repo, created)
	if err != nil {
		return judgement{}, err
	}
	if rc == 124 {
		return judgement{}, failf("%s %s: task timed out", strings.ToLower(kind), tag)
	}
	return judgement{results: tdd.ResultsByID(rep), status: strings.TrimSpace(r.status(repo))}, nil
}

type proof struct{ patch, kind, tag string }

func mutantPatch(proofs, id string, clause int) string {
	if clause == 1 {
		return proofs + "/" + id + ".patch"
	}
	return fmt.Sprintf("%s/%s.%d.patch", proofs, id, clause)
}

// proofPatches lists every proof runProofs judges, in its order.
func (r *run) proofPatches(proofs, text string, ids []string) []proof {
	var list []proof
	for _, i := range ids {
		for clause := 1; clause <= tdd.ClauseCount(tdd.EntryText(text, i)); clause++ {
			list = append(list, proof{mutantPatch(proofs, i, clause), "MUTANT", fmt.Sprintf("%s clause %d", i, clause)})
		}
		if !r.o.NoNeutral {
			list = append(list, proof{proofs + "/" + i + ".neutral.patch", "NEUTRAL", i})
		}
	}
	return list
}

// judgeInWorktree judges a proof on a throwaway worktree of HEAD, the repository untouched.
func (r *run) judgeInWorktree(repo string, p proof) (judgement, error) {
	var lines []string
	w := &run{g: r.g, o: r.o, lines: &lines}
	tmp, cleanup, err := r.g.files.TempDir("tdd-proof-")
	if err != nil {
		return judgement{}, failure{err.Error()}
	}
	defer cleanup()
	wt := tmp + "/wt"
	r.g.worktrees.Lock()
	r.g.git.Run(repo, "worktree", "add", "--detach", wt, "HEAD")
	r.g.worktrees.Unlock()
	defer func() {
		r.g.worktrees.Lock()
		r.g.git.Run(repo, "worktree", "remove", "--force", wt)
		r.g.worktrees.Unlock()
	}()
	if problem := w.mutantProblem(wt, p.patch); problem != "" {
		return judgement{problem: problem, lines: lines}, nil
	}
	if _, err := w.applyMutant(wt, p.patch); err != nil {
		return judgement{}, err
	}
	w.outf("%s %s: patch applied", p.kind, p.tag)
	rc, _, rep, err := w.runTask(wt)
	if err != nil {
		return judgement{}, err
	}
	if rc == 124 {
		return judgement{}, failf("%s %s: task timed out", strings.ToLower(p.kind), p.tag)
	}
	return judgement{results: tdd.ResultsByID(rep), lines: lines}, nil
}

// judgeAll judges the existing proofs, jobs at a time.
func (r *run) judgeAll(repo string, proofs []proof) error {
	type result struct {
		j   judgement
		err error
	}
	results := make([]result, len(proofs))
	slots, done := make(chan struct{}, r.o.Jobs), make(chan struct{})
	started := 0
	for n, p := range proofs {
		if !r.g.files.IsFile(p.patch) {
			continue
		}
		started++
		go func() {
			slots <- struct{}{}
			j, err := r.judgeInWorktree(repo, p)
			results[n] = result{j, err}
			<-slots
			done <- struct{}{}
		}()
	}
	for range started {
		<-done
	}
	r.judged = map[string]judgement{}
	for n, p := range proofs {
		if !r.g.files.IsFile(p.patch) {
			continue
		}
		if results[n].err != nil {
			return results[n].err
		}
		r.judged[p.patch] = results[n].j
	}
	return nil
}

// runProofs judges the mutants and the neutral change of each criterion.
// dirty reports a tree a proof left dirty: the gate stops there.
func (r *run) runProofs(repo, text string, ids []string) (ok, dirty bool, err error) {
	ok = true
	var kills []tdd.Kills
	proofs := r.g.files.Resolve(r.o.Proofs)
	if r.o.Jobs > 1 {
		if err := r.judgeAll(repo, r.proofPatches(proofs, text, ids)); err != nil {
			return false, false, err
		}
	}
	for _, i := range ids {
		n := tdd.Norm(i)
		for clause := 1; clause <= tdd.ClauseCount(tdd.EntryText(text, i)); clause++ {
			patch, tag := mutantPatch(proofs, i, clause), fmt.Sprintf("%s clause %d", i, clause)
			if !r.g.files.IsFile(patch) {
				r.outf("MUTANT MISSING %s", tag)
				ok = false
				continue
			}
			j, err := r.runPatch(repo, patch, "MUTANT", tag)
			if err != nil {
				return false, false, err
			}
			cases := j.results[n]
			failed, hasFailed := tdd.FirstFailed(cases)
			switch {
			case j.problem != "":
				r.outf("MUTANT INVALID %s: %s", tag, j.problem)
				ok = false
			case hasFailed:
				r.outf("KILLED %s: %s", tag, failed.Name)
				kills = addKill(kills, n, j.results)
			case len(cases) > 0:
				r.outf("SURVIVED %s", tag)
				ok = false
			default:
				r.outf("MUTANT INVALID %s: no result", tag)
				ok = false
			}
			if j.status != "" {
				r.outf("TREE NOT CLEAN after %s:\n%s", tag, j.status)
				return false, true, nil
			}
		}
		if r.o.NoNeutral {
			continue
		}
		patch := proofs + "/" + i + ".neutral.patch"
		if !r.g.files.IsFile(patch) {
			r.outf("NEUTRAL MISSING %s", i)
			ok = false
			continue
		}
		j, err := r.runPatch(repo, patch, "NEUTRAL", i)
		if err != nil {
			return false, false, err
		}
		cases := j.results[n]
		failed, hasFailed := tdd.FirstFailed(cases)
		switch {
		case j.problem != "" || len(cases) == 0:
			r.outf("NEUTRAL INVALID %s", i)
			ok = false
		case hasFailed:
			r.outf("BRITTLE %s: %s", i, failed.Name)
			ok = false
		default:
			r.outf("NEUTRAL OK %s", i)
		}
	}
	labels := tdd.IDLabels(text)
	for _, pair := range tdd.Redundant(kills) {
		r.outf("REDUNDANT %s: covered by %s", labels[pair[0]], labels[pair[1]])
		ok = false
	}
	return ok, false, nil
}

// addKill records the criteria one mutant of criterion id made fail.
func addKill(kills []tdd.Kills, id string, results map[string][]tdd.Case) []tdd.Kills {
	failed := map[string]bool{}
	for n, cases := range results {
		if _, has := tdd.FirstFailed(cases); has {
			failed[n] = true
		}
	}
	for k := range kills {
		if kills[k].ID == id {
			kills[k].PerMutant = append(kills[k].PerMutant, failed)
			return kills
		}
	}
	return append(kills, tdd.Kills{ID: id, PerMutant: []map[string]bool{failed}})
}
