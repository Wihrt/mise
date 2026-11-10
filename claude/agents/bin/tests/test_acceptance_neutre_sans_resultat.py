"""Acceptance tests, slug neutre-sans-resultat: PRV-010. Boundary: the tdd-gate command line."""
from test_tdd_gate import (  # noqa: F401  (fixtures are used by name)
    char, char_setup, make_patch, mutants, repo, sh,
)

# The criterion "X-001" is CALC-003 of the helpers. The killing mutant sets the state to fail; the neutral
# change applies but sets a state for which the judged task writes no report (as the no-result mutant does).
KILLING = {"src/state": "fail\n"}
NO_RESULT = {"src/state": "none\n"}


def test_PRV010_neutral_change_without_result_is_invalid(repo, mutants):
    sha = char_setup(repo)
    make_patch(repo, mutants, "CALC-003", KILLING)
    rc, out = char(repo, sha, mutants, neutral=NO_RESULT)
    assert "NEUTRAL INVALID CALC-003" in out, out
    assert "NEUTRAL OK CALC-003" not in out, out
    assert rc == 1 and out.splitlines()[-1] == "FAIL", out
