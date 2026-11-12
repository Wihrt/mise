"""Acceptance tests, slug test-bruno: FRM-008. Boundary: the tdd-gate command line."""
from test_tdd_gate import (  # noqa: F401  (fixtures are used by name)
    TESTS_MD, base_commit, commit, gate, junit, repo,
)

GO_ENTRY = "`a_test.go` › `TestCALC003_C`"
assert GO_ENTRY in TESTS_MD
BRU_MD = TESTS_MD.replace(GO_ENTRY, "`tests/api/div.bru` › `[CALC-003] the quotient is returned`")
BRU = (
    "meta {\n  name: div\n  type: http\n  seq: 1\n}\n\n"
    "get {\n  url: http://127.0.0.1:8123/div\n}\n\n"
    "tests {\n"
    '  test("[CALC-003] the quotient is returned", function() {\n'
    "    expect(res.getBody().quotient).to.equal(2);\n"
    "  });\n"
    "}\n"
)
# The Go file keeps the tests of the other slug only: CALC-003 lives in the Bruno file alone.
GO_OTHERS = "package x\nfunc TestCALC001_A(t *testing.T) {}\nfunc TestCALC002_B(t *testing.T) {}\n"


def test_FRM008_bruno_test_carries_its_id(repo):
    report = junit(("TestCALC001_A", "pass"), ("TestCALC002_B", "pass"), ("[CALC-003] the quotient is returned", "fail"))
    base_commit(repo, report, md=BRU_MD)
    sha = commit(repo, {"tests/api/div.bru": BRU, "a_test.go": GO_OTHERS}, "test(div): acceptance")
    rc, out = gate(repo, "red", "--slug", "div", "--test", sha, "--task", "test")
    assert "RED CALC-003" in out and "MISSING CALC-003" not in out, out
    assert rc == 0 and out.splitlines()[-1] == "PASS", out
