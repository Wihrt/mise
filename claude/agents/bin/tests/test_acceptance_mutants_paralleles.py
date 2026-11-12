"""Acceptance tests, slug mutants-paralleles: PRV-019. Boundary: the tdd-gate command line."""
from test_tdd_gate import (  # noqa: F401  (fixtures are used by name)
    STAB_MD, char, char_setup, make_patch, mutants, repo, sh,
)

ONE_CLAUSE = "  - **Alors** z\n- **Test** : `d_test.go`"
TWO_CLAUSES = "  - **Alors**\n    - premiere clause\n    - seconde clause\n- **Test** : `d_test.go`"
assert ONE_CLAUSE in STAB_MD
TWO_CLAUSE_MD = STAB_MD.replace(ONE_CLAUSE, TWO_CLAUSES)

KILLING = {"src/state": "fail\n"}
HARMLESS = {"src/note.txt": "neutral\n"}


def barrier_task(shared):
    """Under a mutant (src/state = fail) the task waits, for ten seconds at most, until two runs have started,
    then records its directory; alone, it exits 3 without a report."""
    return (
        "mkdir -p .tdd\ns=$(cat src/state)\n"
        'if [ "$s" = fail ]; then\n'
        f"  mkdir -p {shared}/started && touch {shared}/started/$$\n"
        "  n=0\n"
        f"  while [ $(ls {shared}/started | wc -l) -lt 2 ] && [ $n -lt 100 ]; do sleep 0.1; n=$((n+1)); done\n"
        f"  [ $(ls {shared}/started | wc -l) -lt 2 ] && exit 3\n"
        f"  pwd >> {shared}/dirs\n"
        "fi\n"
        "cp tests/$s.xml .tdd/junit.xml\n"
        '[ "$s" = fail ] && exit 1\nexit 0\n'
    )


def test_PRV019_mutants_are_judged_in_parallel(repo, mutants):
    shared = repo.parent / "shared"
    shared.mkdir()
    sha = char_setup(repo, md=TWO_CLAUSE_MD, task=barrier_task(shared))
    make_patch(repo, mutants, "CALC-003", KILLING)
    make_patch(repo, mutants, "CALC-003.2", KILLING)
    worktrees = sh(repo, "git", "worktree", "list")
    rc, out = char(repo, sha, mutants, "--jobs", "2", neutral=HARMLESS)
    assert "KILLED CALC-003 clause 1" in out and "KILLED CALC-003 clause 2" in out, out
    assert rc == 0 and out.splitlines()[-1] == "PASS", out
    dirs = (shared / "dirs").read_text().split()
    assert len(set(dirs)) == 2 and str(repo) not in dirs, dirs
    assert sh(repo, "git", "worktree", "list") == worktrees
    assert sh(repo, "git", "status", "--short") == ""
