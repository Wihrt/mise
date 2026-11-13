"""Acceptance tests, slug code-minimal: PRV-009. Boundary: the harness gate command line."""
from test_tdd_gate import (  # noqa: F401  (fixtures are used by name)
    char_setup, commit, make_patch, mutants, repo, run_green,
)

PROD = "src/calc.txt"
# The task passes only when the production file holds a line "needed".
NEEDED_TASK = (
    "mkdir -p .tdd\n"
    "if grep -q '^needed$' src/calc.txt; then cp tests/pass.xml .tdd/junit.xml; exit 0; fi\n"
    "cp tests/fail.xml .tdd/junit.xml\nexit 1\n"
)
BASE = "header\nmiddle\n"
# Two separate added blocks: line 2 ("needed") and line 4 ("extra"), line 3 unchanged between them.
FEAT = "header\nneeded\nmiddle\nextra\n"
UNNEEDED_BLOCK = f"{PROD}:4-4"
NEEDED_BLOCK = f"{PROD}:2-2"


def two_blocks(repo, mutants):
    t = char_setup(repo, files={PROD: BASE}, task=NEEDED_TASK)
    f = commit(repo, {PROD: FEAT}, "feat(div)")
    make_patch(repo, mutants, "CALC-003", {PROD: "header\nmiddle\nextra\n"})
    return t, f


def unneeded_lines(out):
    return [line for line in out.splitlines() if "UNNEEDED" in line]


def test_PRV009_unneeded_block_is_refused(repo, mutants):
    t, f = two_blocks(repo, mutants)
    rc, out = run_green(repo, t, f, exempt=False, proofs=mutants)
    lines = unneeded_lines(out)
    assert any(line.startswith(f"UNNEEDED {UNNEEDED_BLOCK}") for line in lines), out
    assert not any(NEEDED_BLOCK in line for line in lines), out
    assert out.splitlines()[-1] == "FAIL", out


def test_PRV009_unneeded_ok_accepts_the_block(repo, mutants):
    t, f = two_blocks(repo, mutants)
    rc, out = run_green(repo, t, f, exempt=False, proofs=mutants, unneeded_ok=[UNNEEDED_BLOCK])
    lines = unneeded_lines(out)
    assert any(line.startswith(f"UNNEEDED ACCEPTED {UNNEEDED_BLOCK}") for line in lines), out
    assert all(line.startswith("UNNEEDED ACCEPTED ") for line in lines), out
    assert out.splitlines()[-1] == "PASS", out


def test_PRV009_deletion_and_test_file_are_not_judged(repo, mutants):
    # Base has an extra line "old" (line 3) that the feat commit deletes; the added blocks are line 2 ("needed")
    # and line 5 ("extra"); a test file is added too and the task never reads it.
    t = char_setup(repo, files={PROD: "header\nmiddle\nold\ntail\n"}, task=NEEDED_TASK)
    f = commit(repo, {PROD: "header\nneeded\nmiddle\ntail\nextra\n", "tests/test_extra.py": "def test_x():\n    pass\n"},
               "feat(div)")
    make_patch(repo, mutants, "CALC-003", {PROD: "header\nmiddle\ntail\nextra\n"})
    block = f"{PROD}:5-5"
    rc, out = run_green(repo, t, f, exempt=False, proofs=mutants, unneeded_ok=[block])
    assert [line for line in out.splitlines() if line.startswith("UNNEEDED")] == [f"UNNEEDED ACCEPTED {block}"], out
    assert out.splitlines()[-1] == "PASS", out
