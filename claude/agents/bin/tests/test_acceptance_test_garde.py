"""Acceptance tests, slug test-garde: FRM-002. Boundary: the tdd-gate command line."""
from test_tdd_gate import (  # noqa: F401  (fixtures are used by name)
    D_TESTS, P_TESTS, STAB_MD, char, char_setup, make_patch, mutants, repo,
)

ENTRY = "### {id} — {t}\n- **Critère** :\n  - **Étant donné** x\n  - **Quand** y\n  - **Alors** z\n- **Test** : `{f}` › `{n}`\n"
# The judged slug (div) takes over pow's section; CALC-001 and CALC-002 stay in TESTS.md, in a section of their own.
KEPT_MD = (("## div — Division\n" + ENTRY.format(id="CALC-003", t="c", f="d_test.go", n="TestCALC003_C"))
           + "\n## keep — Kept\n" + ENTRY.format(id="CALC-001", t="a", f="p_test.go", n="TestCALC001_A")
           + ENTRY.format(id="CALC-002", t="b", f="p_test.go", n="TestCALC002_B"))
# CALC-001 (kept, not listed) and CALC-002 (listed in --ac-changed) both have a line modified.
P_MODIFIED = ("package x\nfunc TestCALC001_A(t *testing.T) { _ = 1 }\n"
              "func TestCALC002_B(t *testing.T) { _ = 2 }\n")


def test_FRM002_kept_test_modified_is_reported(repo, mutants):
    sha = char_setup(repo, md=KEPT_MD, acc_files={"p_test.go": P_MODIFIED})
    make_patch(repo, mutants, "CALC-003", {"src/state": "fail\n"})
    rc, out = char(repo, sha, mutants, "--replaces", "pow", "--ac-changed", "CALC-002")
    lines = out.splitlines()
    assert any("KEPT TEST MODIFIED" in ln and "p_test.go" in ln and "CALC-001" in ln for ln in lines), out
    assert not [ln for ln in lines if "KEPT TEST MODIFIED" in ln and "CALC-002" in ln], out
    assert lines[-1] == "FAIL", out
