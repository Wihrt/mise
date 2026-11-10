"""Acceptance tests, slug forme: FRM-001, FRM-003, FRM-005. Boundary: the tdd-gate command line."""
import pytest

from test_tdd_gate import (  # noqa: F401  (fixtures are used by name)
    SPEC, STAB_MD, TESTS_MD, base_commit, char, char_setup, commit, gate, green_setup, junit, mutants, repo,
    run_green,
)

# FRM-001: section div (the judged slug) only cites the spec; section pow keeps the old format too.
OLD_FORMAT_MD = """## pow — Power
### CALC-001 — a
- **Critère** : docs/specs/x.md › CALC-001
- **Test** : `a_test.go` › `TestCALC001_A`
### CALC-002 — b
- **Critère** : docs/specs/x.md › CALC-002
- **Test** : `a_test.go` › `TestCALC002_B`

## div — Division
### CALC-003 — c
- **Critère** : docs/specs/x.md › CALC-003
- **Test** : `a_test.go` › `TestCALC003_C`
"""


def check_incomplete(rc, out):
    lines = out.splitlines()
    assert any("INCOMPLETE CALC-003" in ln for ln in lines), out
    assert lines[-1] == "FAIL", out
    assert not [ln for ln in lines if "INCOMPLETE" in ln and ("CALC-001" in ln or "CALC-002" in ln)], out


def test_FRM001_red_incomplete_entry(repo):
    sha = base_commit(repo, junit(("TestCALC003_C", "fail")), md=OLD_FORMAT_MD)
    check_incomplete(*gate(repo, "red", "--slug", "div", "--test", sha, "--task", "test"))


def test_FRM001_green_incomplete_entry(repo):
    t, f = green_setup(repo, md=OLD_FORMAT_MD)
    check_incomplete(*run_green(repo, t, f))


def test_FRM001_characterize_incomplete_entry(repo, mutants):
    md = STAB_MD.replace("- **Critère** :\n  - **Étant donné** x\n  - **Quand** y\n  - **Alors** z\n- **Test** : `d_test.go`",
                         "- **Critère** : docs/specs/x.md › CALC-003\n- **Test** : `d_test.go`")
    assert md != STAB_MD
    sha = char_setup(repo, md=md)
    check_incomplete(*char(repo, sha, mutants))


# FRM-003: a task that exits 0 without writing a report.
def test_FRM003_characterize_report_missing(repo, mutants):
    sha = char_setup(repo, task="exit 0\n")
    rc, out = char(repo, sha, mutants)
    lines = out.splitlines()
    missing = [ln for ln in lines if "REPORT missing" in ln]
    assert missing, out
    assert "test" in missing[0] and ".tdd/junit.xml" in missing[0], out
    assert lines[-1] == "FAIL", out


# FRM-005: the spec is in the repo with an uncommitted modification.
def dirty_spec(repo):
    (repo / "spec.md").write_text(SPEC + "\nuncommitted edit\n")


def test_FRM005_red_spec_not_committed(repo):
    commit(repo, {"spec.md": SPEC}, "spec")
    sha = base_commit(repo, junit(("TestCALC003_C", "fail")))
    dirty_spec(repo)
    rc, out = gate(repo, "red", "--slug", "div", "--test", sha, "--task", "test", "--spec", "spec.md", "--ac", "CALC-003")
    assert "SPEC NOT COMMITTED" in out and out.splitlines()[-1] == "FAIL", out


def test_FRM005_characterize_spec_not_committed(repo, mutants):
    sha = char_setup(repo)
    dirty_spec(repo)
    rc, out = char(repo, sha, mutants)
    assert "SPEC NOT COMMITTED" in out and out.splitlines()[-1] == "FAIL", out
