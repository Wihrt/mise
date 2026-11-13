package acceptance

import (
	"slices"
	"strings"
	"testing"
)

// The mutants judged in parallel on request: PRV-019.

// barrierTask: under a mutant (src/state = fail) the task waits, for ten
// seconds at most, until two runs have started, then records its directory;
// alone, it exits 3 without a report.
func barrierTask(shared string) string {
	return "mkdir -p .tdd\ns=$(cat src/state)\n" +
		"if [ \"$s\" = fail ]; then\n" +
		"  mkdir -p " + shared + "/started && touch " + shared + "/started/$$\n" +
		"  n=0\n" +
		"  while [ $(ls " + shared + "/started | wc -l) -lt 2 ] && [ $n -lt 100 ]; do sleep 0.1; n=$((n+1)); done\n" +
		"  [ $(ls " + shared + "/started | wc -l) -lt 2 ] && exit 3\n" +
		"  pwd >> " + shared + "/dirs\n" +
		"fi\n" +
		"cp tests/$s.xml .tdd/junit.xml\n" +
		"[ \"$s\" = fail ] && exit 1\nexit 0\n"
}

func TestPRV019_MutantsAreJudgedInParallel(t *testing.T) {
	r := newRepo(t)
	shared := r.parent + "/shared"
	mkdir(t, shared)
	sha := r.charSetup(charOpts{md: twoClauseMD, task: barrierTask(shared)})
	r.makePatch("CALC-003", killing)
	r.makePatch("CALC-003.2", killing)
	worktrees := r.sh("git", "worktree", "list")
	code, out := r.char(sha, charRun{neutral: neutral{change: harmless}}, "--jobs", "2")
	with(t, out, "KILLED CALC-003 clause 1", "KILLED CALC-003 clause 2")
	ends(t, code, out, 0, "PASS")
	dirs := strings.Fields(read(t, shared+"/dirs"))
	slices.Sort(dirs)
	if distinct := slices.Compact(slices.Clone(dirs)); len(distinct) != 2 || slices.Contains(dirs, r.dir) {
		t.Errorf("the two mutants did not run in two worktrees of their own: %q", dirs)
	}
	if got := r.sh("git", "worktree", "list"); got != worktrees {
		t.Errorf("a worktree is left behind:\n%s", got)
	}
	clean(t, r)
}
