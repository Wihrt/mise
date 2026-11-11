"""Acceptance tests, slug preuve-necessaire: PRV-006. Boundary: the tdd-gate command line."""
from test_tdd_gate import (  # noqa: F401  (fixtures are used by name)
    D_TESTS, SPEC, STAB_MD, char, char_setup, junit, make_patch, mutants, repo,
)

# X-001 is CALC-003 and X-002 is CALC-004, both in the judged section "div".
X1, X2 = "CALC-003", "CALC-004"
SECOND_CRITERION = (
    "### CALC-004 — d\n- **Critère** :\n  - **Étant donné** x\n  - **Quand** y\n  - **Alors** z\n"
    "- **Test** : `d_test.go` › `TestCALC004_D`\n"
)
MD = STAB_MD + SECOND_CRITERION
TESTS = D_TESTS + "func TestCALC004_D(t *testing.T) {}\n"
ALL = [("TestCALC001_A", "pass"), ("TestCALC002_B", "pass")]
REPORTS = {
    "tests/pass.xml": junit(*ALL, ("TestCALC003_C", "pass"), ("TestCALC004_D", "pass")),
    "tests/fail.xml": junit(*ALL, ("TestCALC003_C", "fail"), ("TestCALC004_D", "pass")),
    "tests/both.xml": junit(*ALL, ("TestCALC003_C", "fail"), ("TestCALC004_D", "fail")),
    "tests/other.xml": junit(*ALL, ("TestCALC003_C", "pass"), ("TestCALC004_D", "fail")),
}
TASK = (
    "mkdir -p .tdd\ns=$(cat src/state)\ncp tests/$s.xml .tdd/junit.xml\n"
    "case \"$s\" in pass) exit 0;; *) exit 1;; esac\n"
)
SPEC2 = SPEC.replace("### CALC-001", "### CALC-004 — d\n### CALC-001")


def judged(repo, mutants, mutant_x1, mutant_x2):
    sha = char_setup(repo, md=MD, task=TASK, files={"spec.md": SPEC2},
                     acc_files={"d_test.go": TESTS, **REPORTS})
    make_patch(repo, mutants, X1, {"src/state": mutant_x1 + "\n"})
    make_patch(repo, mutants, X2, {"src/state": mutant_x2 + "\n"})
    return char(repo, sha, mutants, ac=f"{X1},{X2}")


def test_PRV006_criterion_covered_by_another_is_redundant(repo, mutants):
    # mutant of X-002 fails the tests of X-001 and X-002; mutant of X-001 fails only the test of X-001
    rc, out = judged(repo, mutants, "fail", "both")
    redundant = [line for line in out.splitlines() if "REDUNDANT CALC-004" in line]
    assert redundant and all(X1 in line for line in redundant), out
    assert "REDUNDANT CALC-003" not in out, out
    assert rc != 0 and out.splitlines()[-1] == "FAIL", out
