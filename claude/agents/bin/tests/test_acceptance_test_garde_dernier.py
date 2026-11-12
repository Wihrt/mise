"""Acceptance tests, slug test-garde-dernier: PRV-016. Boundary: the harness gate command line."""
from test_tdd_gate import (  # noqa: F401  (fixtures are used by name)
    char, char_setup, junit, make_patch, mutants, repo,
)

ENTRY = "### {id} — {t}\n- **Critère** :\n  - **Étant donné** x\n  - **Quand** y\n  - **Alors** z\n- **Test** : `{f}` › `{n}`\n"


def kept_md(entries):
    # The judged slug (div) takes over pow's section; the kept entries stay in TESTS.md in a section of their own.
    md = "## div — Division\n" + ENTRY.format(id="CALC-003", t="c", f="d_test.go", n="TestCALC003_C") + "\n## keep — Kept\n"
    return md + "".join(ENTRY.format(id=i, t="k", f=f, n=n) for i, f, n in entries)


MAIN_BLOCK = 'if __name__ == "__main__":\n    sys.exit(pytest.main([__file__, "-v"]))\n'
PY_HEAD = "import sys\n\nimport pytest\n\n\n"
PY_KEPT = "def test_CALC001_first():\n    assert 1\n\n\n"
# The last test (CALC-002) is removed; the kept CALC-001 test becomes the last one, before the same final block.
PY_BASE = PY_HEAD + PY_KEPT + "def test_CALC002_last():\n    assert 2\n\n\n" + MAIN_BLOCK
PY_ACC = PY_HEAD + PY_KEPT + MAIN_BLOCK


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


def test_PRV016_kept_test_becoming_last_of_file_is_not_modified(repo, mutants):
    name = "test_p.py"
    rc, out = run(repo, mutants, PY_BASE, PY_ACC, name, [("CALC-001", name, "test_CALC001_first")],
                  ["test_CALC001_first", "test_CALC002_last"])
    assert not [ln for ln in out.splitlines() if "KEPT TEST MODIFIED" in ln and "CALC-001" in ln], out
