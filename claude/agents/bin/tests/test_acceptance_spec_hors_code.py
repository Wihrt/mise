"""Acceptance tests, slug spec-hors-code: PRV-014. Boundary: the harness gate command line."""
from test_tdd_gate import (  # noqa: F401  (fixtures are used by name)
    char_setup, commit, gate, make_patch, mutants, prepare_proofs, repo,
)

PROD = "src/calc.txt"
SPEC = "docs/specs/design.md"
NEEDED_TASK = (
    "mkdir -p .tdd\n"
    "if grep -q '^needed$' src/calc.txt; then cp tests/pass.xml .tdd/junit.xml; exit 0; fi\n"
    "cp tests/fail.xml .tdd/junit.xml\nexit 1\n"
)


def test_PRV014_spec_lines_added_between_two_acceptance_commits_are_not_unneeded(repo, mutants):
    first = char_setup(repo, files={PROD: "header\nmiddle\n", SPEC: "# Spec\n"}, task=NEEDED_TASK)
    commit(repo, {SPEC: "# Spec\n\n### X-001\nnew criterion line\n"}, "docs: spec change")
    second = commit(repo, {"tests/test_extra.py": "def test_x():\n    pass\n"}, "test(div): acceptance")
    feat = commit(repo, {PROD: "header\nneeded\nmiddle\n"}, "feat(div)")
    make_patch(repo, mutants, "CALC-003", {PROD: "header\nmiddle\n"})
    prepare_proofs(repo, mutants)
    rc, out = gate(repo, "green", "--slug", "div", "--test", second, "--since", first, "--feat", feat,
                   "--task", "test", "--proofs", str(mutants), "--spec", SPEC)
    assert not any(ln.startswith("UNNEEDED") and SPEC in ln for ln in out.splitlines()), out
    assert out.splitlines()[-1:] == ["PASS"], out
