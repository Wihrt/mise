"""Acceptance tests, slug test-garde-voisin: PRV-015. Boundary: the tdd-gate command line."""
from test_tdd_gate import (  # noqa: F401  (fixtures are used by name)
    char, char_setup, junit, make_patch, mutants, repo,
)

ENTRY = "### {id} — {t}\n- **Critère** :\n  - **Étant donné** x\n  - **Quand** y\n  - **Alors** z\n- **Test** : `{f}` › `{n}`\n"


def kept_md(entries):
    # The judged slug (div) takes over pow's section; the kept entries stay in TESTS.md in a section of their own.
    md = "## div — Division\n" + ENTRY.format(id="CALC-003", t="c", f="d_test.go", n="TestCALC003_C") + "\n## keep — Kept\n"
    return md + "".join(ENTRY.format(id=i, t="k", f=f, n=n) for i, f, n in entries)


PY_BASE = (
    "import pytest\n\n\n"
    "def test_CALC001_first():\n    assert 1\n\n\n"
    '@pytest.mark.parametrize("v", [1])\n'
    "def test_CALC002_removed(v):\n    assert v\n\n\n"
    '@pytest.mark.parametrize("w", [2])\n'
    "def test_CALC004_next(w):\n    assert w\n"
)
# The neighbour (CALC-002, decorated) is removed; the next test, decorated otherwise, takes its place.
PY_ACC = (
    "import pytest\n\n\n"
    "def test_CALC001_first():\n    assert 1\n\n\n"
    '@pytest.mark.parametrize("w", [2])\n'
    "def test_CALC004_next(w):\n    assert w\n"
)

HELM_BASE = (
    "suite: one\ntemplates:\n  - a.yaml\ntests:\n"
    '  - it: "[CALC-001] first"\n    asserts:\n      - isKind:\n          of: Deployment\n'
    "---\n"
    "suite: two\ntemplates:\n  - b.yaml\ntests:\n"
    '  - it: "[CALC-002] removed"\n    asserts:\n      - isKind:\n          of: Service\n'
)
# The document that follows CALC-001's (separator and suite header included) is removed.
HELM_ACC = HELM_BASE[: HELM_BASE.index("---\n")]


def run(repo, mutants, base, acc, name, entries, names):
    # pow (the base commit's slug) owns the file under test, so the acceptance commit may edit it (--replaces pow).
    pow_md = "## pow — Power\n" + "".join(
        ENTRY.format(id=i, t="p", f=name, n=n) for i, n in (("CALC-001", names[0]), ("CALC-002", names[1])))
    ok = [(n, "pass") for n in names if n != names[1]] + [("TestCALC003_C", "pass")]
    ko = [(n, "pass") for n in names if n != names[1]] + [("TestCALC003_C", "fail")]
    reports = {"tests/pass.xml": junit(*ok), "tests/slow.xml": junit(*ok), "tests/fail.xml": junit(*ko)}
    sha = char_setup(repo, md=kept_md(entries), base_md=pow_md, files={name: base, "p_test.go": "package x\n"},
                     acc_files={name: acc, **reports})
    make_patch(repo, mutants, "CALC-003", {"src/state": "fail\n"})
    return char(repo, sha, mutants, "--replaces", "pow")


def test_PRV015_pytest_decorated_neighbour_removed_does_not_modify_kept_test(repo, mutants):
    name = "test_p.py"
    rc, out = run(repo, mutants, PY_BASE, PY_ACC, name, [
        ("CALC-001", name, "test_CALC001_first"), ("CALC-004", name, "test_CALC004_next")],
        ["test_CALC001_first", "test_CALC002_removed", "test_CALC004_next"])
    assert not [ln for ln in out.splitlines() if "KEPT TEST MODIFIED" in ln and "CALC-001" in ln], out


def test_PRV015_helm_unittest_following_document_removed_does_not_modify_kept_test(repo, mutants):
    name = "p_suite_test.yaml"
    rc, out = run(repo, mutants, HELM_BASE, HELM_ACC, name, [("CALC-001", name, "[CALC-001] first")],
                   ["[CALC-001] first", "[CALC-002] removed"])
    assert not [ln for ln in out.splitlines() if "KEPT TEST MODIFIED" in ln and "CALC-001" in ln], out
