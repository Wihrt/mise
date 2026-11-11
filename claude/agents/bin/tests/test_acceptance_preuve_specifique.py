"""Acceptance tests, slug preuve-specifique: PRV-003, PRV-004, PRV-005. Boundary: the tdd-gate command line."""
from test_tdd_gate import (  # noqa: F401  (fixtures are used by name)
    STAB_MD, char, char_setup, make_patch, mutants, repo,
)

# The criterion "X-001" is CALC-003 of the helpers. Its "Alors" has one clause, or two (a line then a list).
ONE_CLAUSE = "  - **Alors** z\n- **Test** : `d_test.go`"
TWO_CLAUSES = "  - **Alors**\n    - premiere clause\n    - seconde clause\n- **Test** : `d_test.go`"
assert ONE_CLAUSE in STAB_MD
TWO_CLAUSE_MD = STAB_MD.replace(ONE_CLAUSE, TWO_CLAUSES)

# A mutant makes the test of CALC-003 fail; a neutral change makes none fail.
KILLING = {"src/state": "fail\n"}
HARMLESS = {"src/note.txt": "neutral\n"}


def test_PRV003_missing_neutral_change(repo, mutants):
    sha = char_setup(repo)
    make_patch(repo, mutants, "CALC-003", KILLING)
    rc, out = char(repo, sha, mutants, neutral=False)
    assert "NEUTRAL MISSING CALC-003" in out, out
    assert out.splitlines()[-1] == "FAIL", out


NOT_APPLYING = (
    "diff --git a/src/absent.txt b/src/absent.txt\n--- a/src/absent.txt\n+++ b/src/absent.txt\n@@ -1 +1 @@\n-x\n+y\n"
)


def test_PRV003_neutral_change_not_applying_is_invalid(repo, mutants):
    sha = char_setup(repo)
    make_patch(repo, mutants, "CALC-003", KILLING)
    (mutants / "CALC-003.neutral.patch").write_text(NOT_APPLYING)
    rc, out = char(repo, sha, mutants)
    assert "NEUTRAL INVALID CALC-003" in out, out
    assert "NEUTRAL OK CALC-003" not in out, out
    assert out.splitlines()[-1] == "FAIL", out


def test_PRV004_test_breaking_under_neutral_change_is_brittle(repo, mutants):
    # The neutral change renames an element the test asserts: the test of CALC-003 fails under it.
    sha = char_setup(repo)
    make_patch(repo, mutants, "CALC-003", KILLING)
    rc, out = char(repo, sha, mutants, neutral=KILLING)
    brittle = [line for line in out.splitlines() if "BRITTLE CALC-003" in line]
    assert brittle, out
    assert "TestCALC003_C" in brittle[0], out
    assert out.splitlines()[-1] == "FAIL", out


def test_PRV005_sensitive_and_specific_test_passes(repo, mutants):
    sha = char_setup(repo, md=TWO_CLAUSE_MD)
    make_patch(repo, mutants, "CALC-003", KILLING)
    make_patch(repo, mutants, "CALC-003.2", KILLING)
    rc, out = char(repo, sha, mutants, neutral=HARMLESS)
    assert "MUTANT MISSING" not in out, out
    assert "KILLED CALC-003 clause 1" in out, out
    assert "KILLED CALC-003 clause 2" in out, out
    assert "NEUTRAL OK CALC-003" in out, out
    assert out.splitlines()[-1] == "PASS", out
