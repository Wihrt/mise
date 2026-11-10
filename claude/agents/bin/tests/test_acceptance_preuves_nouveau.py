"""Acceptance tests, slug preuves-nouveau: PRV-007. Boundary: the tdd-gate command line."""
from test_tdd_gate import (  # noqa: F401  (fixtures are used by name)
    char_setup, commit, make_patch, mutants, repo, run_green,
)


def new_feature(repo):
    """Red acceptance commit (CALC-003 fails), then a feat commit making it green."""
    t = char_setup(repo, state="fail")
    f = commit(repo, {"src/state": "pass\n"}, "feat(div)")
    return t, f


def test_PRV007_missing_proofs_option_fails(repo):
    t, f = new_feature(repo)
    rc, out = run_green(repo, t, f, exempt=False)
    assert rc == 1 and "PROOF MISSING" in out and out.splitlines()[-1] == "FAIL", out


def test_PRV007_surviving_mutant_fails(repo, mutants):
    t, f = new_feature(repo)
    make_patch(repo, mutants, "CALC-003", {"src/note.txt": "# comment\n"})
    rc, out = run_green(repo, t, f, exempt=False, proofs=mutants)
    assert rc == 1 and "SURVIVED" in out and out.splitlines()[-1] == "FAIL", out


def test_PRV007_killed_mutant_and_neutral_change_pass(repo, mutants):
    t, f = new_feature(repo)
    make_patch(repo, mutants, "CALC-003", {"src/state": "fail\n"})
    rc, out = run_green(repo, t, f, exempt=False, proofs=mutants)
    assert "KILLED CALC-003" in out and "NEUTRAL OK CALC-003" in out, out
    assert "SURVIVED" not in out and out.splitlines()[-1] == "PASS", out


def test_PRV007_proof_exempt_applies_no_mutant(repo):
    t, f = new_feature(repo)
    rc, out = run_green(repo, t, f, exempt=True)
    assert rc == 0 and "KILLED" not in out and "SURVIVED" not in out and out.splitlines()[-1] == "PASS", out
