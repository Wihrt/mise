package acceptance

import "testing"

// Slugs rejeu (RPL-001, RPL-002), revue-finale (RPL-003) and
// accept-lost-multiple (RPL-004): the mutants replayed on the base and the
// head of a stack.

const (
	guard   = "grep -q fail src/state && exit 1\nexit 0\n"
	noGuard = "exit 0\n"
)

// stack makes two commits: on the first the task fails when src/state says fail; headTask is the task of the second.
func stack(r *repo, headTask string) (string, string) {
	r.t.Helper()
	base := r.commit(files{"run_test.sh": guard, "mise.toml": miseToml(miseOpts{}), ".gitignore": ".tdd\n",
		"src/state": "ok\n", "src/note.txt": "note\n"}, "base")
	return base, r.commit(files{"run_test.sh": headTask}, "head")
}

// patchesOn writes the mutants against the commit sha, then returns to main.
func patchesOn(r *repo, sha string, patches map[string]files) {
	r.t.Helper()
	r.sh("git", "checkout", "-q", sha)
	for id, f := range patches {
		r.makePatch(id, f)
	}
	r.sh("git", "checkout", "-q", "main")
}

func TestRPL001_LostProtectionIsReportedThenAccepted(t *testing.T) {
	r := newRepo(t)
	base, head := stack(r, noGuard)
	patchesOn(r, base, map[string]files{"m-state": {"src/state": "fail\n"}})
	code, out := r.replay(base, head)
	with(t, out, "LOST m-state")
	without(t, out, "LOST ACCEPTED")
	ends(t, code, out, 1, "FAIL")
	code, out = r.replay(base, head, "--accept-lost", "m-state")
	with(t, out, "LOST ACCEPTED m-state")
	ends(t, code, out, 0, "PASS")
}

func TestRPL001_GainedProtectionIsReported(t *testing.T) {
	r := newRepo(t)
	base, head := stack(r, noGuard)
	patchesOn(r, base, map[string]files{"m-state": {"src/state": "fail\n"}})
	code, out := r.replay(head, base)
	with(t, out, "GAINED m-state")
	ends(t, code, out, 0, "PASS")
}

func TestRPL002_LosslessReplayPassesAndLeavesNothing(t *testing.T) {
	r := newRepo(t)
	base, head := stack(r, guard+"# changed\n")
	patchesOn(r, base, map[string]files{"m-killed": {"src/state": "fail\n"}, "m-alive": {"src/note.txt": "# comment\n"}})
	worktrees, status := r.sh("git", "worktree", "list"), r.status()
	code, out := r.replay(base, head)
	with(t, out, "KEPT m-killed", "NEVER m-alive")
	ends(t, code, out, 0, "PASS")
	if got := r.sh("git", "worktree", "list"); got != worktrees {
		t.Errorf("a worktree is left behind:\n%s", got)
	}
	if got := r.status(); got != status {
		t.Errorf("the tree changed:\n%s", got)
	}
}

// failingThenPassing makes a first commit whose task fails and a second whose task passes.
func failingThenPassing(r *repo) (string, string) {
	r.t.Helper()
	first := r.commit(files{"run_test.sh": "exit 1\n", "mise.toml": miseToml(miseOpts{}), ".gitignore": ".tdd\n",
		"src/state": "ok\n", "src/note.txt": "note\n"}, "first fails")
	r.makePatch("m-state", files{"src/state": "fail\n"})
	return first, r.commit(files{"run_test.sh": "exit 0\n"}, "second passes")
}

func TestRPL003_FailingTaskWithoutMutantStopsTheReplay(t *testing.T) {
	r := newRepo(t)
	first, second := failingThenPassing(r)
	_, out := r.replay(first, second)
	with(t, out, "BASELINE NOT GREEN base")
	without(t, out, "KEPT")
	lastIs(t, out, "FAIL")
}

func TestRPL003_SwappedBaseAndHeadNamesHead(t *testing.T) {
	r := newRepo(t)
	first, second := failingThenPassing(r)
	_, out := r.replay(second, first)
	with(t, out, "BASELINE NOT GREEN head")
	without(t, out, "KEPT")
	lastIs(t, out, "FAIL")
}

func TestRPL004_EachAcceptedLossIsNamed(t *testing.T) {
	r := newRepo(t)
	base := r.commit(files{"run_test.sh": "grep -q fail src/one src/two && exit 1\nexit 0\n", "mise.toml": miseToml(miseOpts{}),
		".gitignore": ".tdd\n", "src/one": "ok\n", "src/two": "ok\n"}, "base")
	head := r.commit(files{"run_test.sh": "exit 0\n"}, "head")
	patchesOn(r, base, map[string]files{"m-one": {"src/one": "fail\n"}, "m-two": {"src/two": "fail\n"}})
	code, out := r.replay(base, head, "--accept-lost", "m-one", "--accept-lost", "m-two")
	with(t, out, "LOST ACCEPTED m-one", "LOST ACCEPTED m-two")
	ends(t, code, out, 0, "PASS")
	code, out = r.replay(base, head, "--accept-lost", "m-one")
	with(t, out, "LOST m-two")
	without(t, out, "LOST ACCEPTED m-two")
	ends(t, code, out, 1, "FAIL")
}
