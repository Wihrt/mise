"""Acceptance tests, slug preuves-optionnelles: PRV-017, PRV-018. Boundary: the tdd-gate command line."""
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


def feat(repo, mutants, content):
    """Acceptance commit, feat commit writing `content` in the production file, and a mutant killing CALC-003."""
    t = char_setup(repo, files={PROD: BASE}, task=NEEDED_TASK)
    f = commit(repo, {PROD: content}, "feat(div)")
    make_patch(repo, mutants, "CALC-003", {PROD: content.replace("needed\n", "")})
    return t, f


def test_PRV017_neutral_change_is_skipped_on_request(repo, mutants):
    t, f = feat(repo, mutants, "header\nneeded\nmiddle\n")
    rc, out = run_green(repo, t, f, exempt=False, proofs=mutants, neutral=False, options=["--no-neutral"])
    assert "KILLED CALC-003" in out and "NEUTRAL SKIPPED" in out, out
    assert "NEUTRAL MISSING" not in out, out
    assert rc == 0 and out.splitlines()[-1] == "PASS", out
    rc, out = run_green(repo, t, f, exempt=False, proofs=mutants, neutral=False)
    assert "NEUTRAL MISSING CALC-003" in out, out
    assert rc != 0 and out.splitlines()[-1] == "FAIL", out


def test_PRV018_minimal_code_is_skipped_on_request(repo, mutants):
    # Line 4 ("extra") is a block no test needs.
    t, f = feat(repo, mutants, "header\nneeded\nmiddle\nextra\n")
    rc, out = run_green(repo, t, f, exempt=False, proofs=mutants, options=["--no-minimal"])
    assert "MINIMAL SKIPPED" in out and "UNNEEDED" not in out, out
    assert rc == 0 and out.splitlines()[-1] == "PASS", out
    rc, out = run_green(repo, t, f, exempt=False, proofs=mutants)
    assert f"UNNEEDED {PROD}:4-4" in out, out
    assert rc != 0 and out.splitlines()[-1] == "FAIL", out
