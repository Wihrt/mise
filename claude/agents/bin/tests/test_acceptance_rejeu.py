"""Acceptance tests, slug rejeu: RPL-001, RPL-002. Boundary: the tdd-gate command line."""
from test_tdd_gate import (  # noqa: F401  (fixtures are used by name)
    commit, make_patch, mise_toml, mutants, replay, repo, sh,
)

GUARD = "grep -q fail src/state && exit 1\nexit 0\n"
NO_GUARD = "exit 0\n"


def stack(repo, head_task):
    """Two commits: the task fails when src/state says fail on the first; head_task is the task of the second."""
    base = commit(repo, {"run_test.sh": GUARD, "mise.toml": mise_toml(), ".gitignore": ".tdd\n",
                         "src/state": "ok\n", "src/note.txt": "note\n"}, "base")
    head = commit(repo, {"run_test.sh": head_task}, "head")
    return base, head


def test_RPL001_lost_protection_is_reported_then_accepted(repo, mutants):
    base, head = stack(repo, NO_GUARD)
    sh(repo, "git", "checkout", "-q", base)
    make_patch(repo, mutants, "m-state", {"src/state": "fail\n"})
    sh(repo, "git", "checkout", "-q", "main")
    rc, out = replay(repo, base, head, mutants, "test")
    assert rc != 0 and "LOST m-state" in out and "LOST ACCEPTED" not in out, out
    assert out.splitlines()[-1] == "FAIL", out
    rc, out = replay(repo, base, head, mutants, "test", "--accept-lost", "m-state")
    assert rc == 0 and "LOST ACCEPTED m-state" in out and out.splitlines()[-1] == "PASS", out


def test_RPL002_lossless_replay_passes_and_leaves_nothing(repo, mutants):
    base, head = stack(repo, GUARD + "# changed\n")
    sh(repo, "git", "checkout", "-q", base)
    make_patch(repo, mutants, "m-killed", {"src/state": "fail\n"})
    make_patch(repo, mutants, "m-alive", {"src/note.txt": "# comment\n"})
    sh(repo, "git", "checkout", "-q", "main")
    worktrees, status = sh(repo, "git", "worktree", "list"), sh(repo, "git", "status", "--short")
    rc, out = replay(repo, base, head, mutants, "test")
    assert rc == 0 and "KEPT m-killed" in out and "NEVER m-alive" in out, out
    assert out.splitlines()[-1] == "PASS", out
    assert sh(repo, "git", "worktree", "list") == worktrees
    assert sh(repo, "git", "status", "--short") == status
