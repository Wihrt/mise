"""Acceptance tests, slug accept-lost-multiple: RPL-004. Boundary: the tdd-gate command line."""
from test_tdd_gate import (  # noqa: F401  (fixtures are used by name)
    commit, make_patch, mise_toml, mutants, replay, repo, sh,
)

GUARD = "grep -q fail src/one src/two && exit 1\nexit 0\n"


def test_RPL004_each_accepted_loss_is_named(repo, mutants):
    base = commit(repo, {"run_test.sh": GUARD, "mise.toml": mise_toml(), ".gitignore": ".tdd\n",
                         "src/one": "ok\n", "src/two": "ok\n"}, "base")
    head = commit(repo, {"run_test.sh": "exit 0\n"}, "head")
    sh(repo, "git", "checkout", "-q", base)
    make_patch(repo, mutants, "m-one", {"src/one": "fail\n"})
    make_patch(repo, mutants, "m-two", {"src/two": "fail\n"})
    sh(repo, "git", "checkout", "-q", "main")
    rc, out = replay(repo, base, head, mutants, "test",
                     "--accept-lost", "m-one", "--accept-lost", "m-two")
    assert "LOST ACCEPTED m-one" in out and "LOST ACCEPTED m-two" in out, out
    assert rc == 0 and out.splitlines()[-1] == "PASS", out
    rc, out = replay(repo, base, head, mutants, "test", "--accept-lost", "m-one")
    assert "LOST m-two" in out and "LOST ACCEPTED m-two" not in out, out
    assert rc != 0 and out.splitlines()[-1] == "FAIL", out
