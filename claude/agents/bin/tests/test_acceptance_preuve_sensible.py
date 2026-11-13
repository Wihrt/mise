"""Acceptance tests, slug preuve-sensible: PRV-001, PRV-002. Boundary: the harness gate command line."""
from test_tdd_gate import (  # noqa: F401  (fixtures are used by name)
    STAB_MD, char, char_setup, make_patch, mutants, repo,
)

# The criterion "X-001" is CALC-003 of the helpers, whose "Alors" has two clauses (a line followed by a list).
ONE_CLAUSE = "  - **Alors** z\n- **Test** : `d_test.go`"
TWO_CLAUSES = "  - **Alors**\n    - premiere clause\n    - seconde clause\n- **Test** : `d_test.go`"
assert ONE_CLAUSE in STAB_MD
TWO_CLAUSE_MD = STAB_MD.replace(ONE_CLAUSE, TWO_CLAUSES)

# Mutant of clause 1: a test fails under it. Mutant of clause 2 and neutral change: no test fails.
KILLING = {"src/state": "fail\n"}
HARMLESS = {"src/note.txt": "# comment\n"}


def proofs(repo, mutants, *, clause1=True, clause2=True):
    if clause1:
        make_patch(repo, mutants, "CALC-003", KILLING)
    if clause2:
        make_patch(repo, mutants, "CALC-003.2", HARMLESS)
    make_patch(repo, mutants, "CALC-003.neutral", {"src/note.txt": "neutral\n"})


def test_PRV001_missing_mutant_for_clause_2(repo, mutants):
    sha = char_setup(repo, md=TWO_CLAUSE_MD)
    proofs(repo, mutants, clause2=False)
    rc, out = char(repo, sha, mutants)
    assert "MUTANT MISSING CALC-003 clause 2" in out, out
    assert out.splitlines()[-1] == "FAIL", out


def test_PRV002_clause_without_defending_test_survives(repo, mutants):
    sha = char_setup(repo, md=TWO_CLAUSE_MD)
    proofs(repo, mutants)
    rc, out = char(repo, sha, mutants)
    assert "KILLED CALC-003 clause 1" in out, out
    assert "SURVIVED CALC-003 clause 2" in out, out
    assert out.splitlines()[-1] == "FAIL", out
