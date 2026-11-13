package acceptance

import (
	"slices"
	"strings"
	"testing"
	"time"
)

// The minimal-code proof judged in parallel on request: PRV-020, PRV-021.

// runSeconds is the fixed duration of one run of the test task of PRV-020.
const runSeconds = 2

func TestPRV020_MinimalCodeIsJudgedFasterWithJobs(t *testing.T) {
	r := newRepo(t)
	// Four separate blocks: line 2 ("needed"), then lines 4, 6 and 8.
	test := r.charSetup(charOpts{files: files{prod: "header\nmiddle\nm2\nm3\n"}, task: "sleep 2\n" + neededTask})
	feat := r.commit(files{prod: "header\nneeded\nmiddle\nx1\nm2\nx2\nm3\nx3\n"}, "feat(div)")
	r.makePatch("CALC-003", files{prod: "header\nmiddle\nx1\nm2\nx2\nm3\nx3\n"})
	opts := greenRun{noExempt: true, proofs: true, unneededOK: []string{prod + ":4-4", prod + ":6-6", prod + ":8-8"}}

	start := time.Now()
	r.runGreen(test, feat, opts)
	sequential := time.Since(start)

	opts.options = []string{"--jobs", "4"}
	start = time.Now()
	r.runGreen(test, feat, opts)
	parallel := time.Since(start)

	if gain, want := sequential-parallel, 2*runSeconds*time.Second; gain < want {
		t.Errorf("--jobs 4 took %s, without --jobs %s: gained %s, want at least %s", parallel, sequential, gain, want)
	}
}

func TestPRV021_JobsGiveTheSameVerdictsAndLeaveTheRepoIntact(t *testing.T) {
	r := newRepo(t)
	shared := r.parent + "/shared"
	mkdir(t, shared)
	// The task records where it runs whenever the second block ("extra") is removed and the first is there.
	record := "if grep -q '^needed$' src/calc.txt && ! grep -q '^extra$' src/calc.txt; then pwd >> " + shared + "/removed; fi\n"
	test := r.charSetup(charOpts{files: files{prod: prodBase}, task: record + neededTask})
	feat := r.commit(files{prod: "header\nneeded\nmiddle\nextra\n"}, "feat(div)")
	r.makePatch("CALC-003", files{prod: "header\nmiddle\nextra\n"})
	worktrees, status := r.sh("git", "worktree", "list"), r.status()
	block := prod + ":4-4"

	code, out := r.runGreen(test, feat, greenRun{noExempt: true, proofs: true, options: []string{"--jobs", "2"}})
	unneeded := grep(out, "UNNEEDED")
	if len(unneeded) != 1 || unneeded[0] != "UNNEEDED "+block {
		t.Errorf("UNNEEDED lines = %q", unneeded)
	}
	ends(t, code, out, 1, "FAIL")

	code, out = r.runGreen(test, feat, greenRun{noExempt: true, proofs: true, options: []string{"--jobs", "2"}, unneededOK: []string{block}})
	unneeded = grep(out, "UNNEEDED")
	if len(unneeded) != 1 || unneeded[0] != "UNNEEDED ACCEPTED "+block {
		t.Errorf("UNNEEDED lines = %q", unneeded)
	}
	ends(t, code, out, 0, "PASS")

	if !exists(shared + "/removed") {
		t.Error("the task never ran with the second block removed")
	} else if slices.Contains(strings.Fields(read(t, shared+"/removed")), r.dir) {
		t.Errorf("the task ran in the repository itself with a block removed: %q", read(t, shared+"/removed"))
	}
	if got := r.sh("git", "worktree", "list"); got != worktrees {
		t.Errorf("git worktree list changed:\n%s\nwas:\n%s", got, worktrees)
	}
	if got := r.status(); got != status {
		t.Errorf("git status --short changed:\n%s\nwas:\n%s", got, status)
	}
}
