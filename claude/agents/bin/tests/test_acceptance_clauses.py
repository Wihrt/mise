"""Acceptance tests, slug clauses: FRM-006. Boundary: the harness gate command line."""
from test_tdd_gate import (  # noqa: F401  (fixtures are used by name)
    STAB_MD, TESTS_MD, acc_commit, char, char_setup, commit, gate, junit, mutants, repo, task_script,
)

# The spec gives CALC-003 an "Alors" in two clauses; the TESTS.md entry carries it on a single line.
TWO_CLAUSES_SPEC = (
    "# s\n## Critères d'acceptation\n### CALC-003 — c\n- **Étant donné** x\n- **Quand** y\n"
    "- **Alors**\n  - first clause\n  - second clause\n### CALC-001 — a\n### CALC-002 — b\n"
)
ONE_LINE_ENTRY = "  - **Alors** first clause ; second clause\n"


def one_line(md):
    """md with CALC-003's entry carrying its Alors on one line (the last 'z' Alors line of md is CALC-003's)."""
    head, _, tail = md.rpartition("  - **Alors** z\n")
    return head + ONE_LINE_ENTRY + tail


def test_FRM006_red_entry_with_fewer_clauses_than_spec_fails(repo):
    commit(repo, {"run_test.sh": task_script("tests/report.xml"), "spec.md": TWO_CLAUSES_SPEC}, "base")
    sha = acc_commit(repo, junit(("TestCALC003_C", "fail")), extra={"TESTS.md": one_line(TESTS_MD)})
    rc, out = gate(repo, "red", "--slug", "div", "--test", sha, "--task", "test", "--spec", "spec.md", "--ac", "CALC-003")
    assert "CLAUSES CALC-003: 1 in TESTS.md, 2 in spec" in out
    assert out.rstrip().splitlines()[-1] == "FAIL"


def test_FRM006_characterize_entry_with_fewer_clauses_than_spec_fails(repo, mutants):
    sha = char_setup(repo, md=one_line(STAB_MD), files={"spec.md": TWO_CLAUSES_SPEC})
    rc, out = char(repo, sha, mutants)
    assert "CLAUSES CALC-003: 1 in TESTS.md, 2 in spec" in out
    assert out.rstrip().splitlines()[-1] == "FAIL"
