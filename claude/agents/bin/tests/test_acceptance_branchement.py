"""Acceptance tests, slug branchement: FRM-004. Boundary: the tdd-gate command line."""
import pytest

from test_tdd_gate import (  # noqa: F401  (fixtures are used by name)
    STAB_MD, TESTS_MD, gate, green_setup, char_setup, make_patch, mutants, precommit, repo, set_wiring,
)

TASK = "test:render"
COMMAND = f"Branche : `feat/div` · Commande : `mise run {TASK}`\n"
GITLAB_CI = f"render:\n  script:\n    - mise run {TASK}\n"


def with_command(md):
    return md.replace("## div — Division\n", "## div — Division\n" + COMMAND)


def run_green(repo, check_depends=None):
    t, f = green_setup(repo, md=with_command(TESTS_MD), task=TASK, check_depends=check_depends)
    return gate(repo, "green", "--slug", "div", "--test", t, "--feat", f, "--task", TASK, "--proof-exempt")


def run_char(repo, mutants):
    sha = char_setup(repo, md=with_command(STAB_MD), mise_task=TASK)
    make_patch(repo, mutants, "CALC-003", {"src/state": "fail\n"})
    return gate(repo, "characterize", "--slug", "div", "--test", sha, "--task", TASK, "--spec", "spec.md",
                "--ac", "CALC-003", "--mutants", str(mutants))


# The task is cited by no commit hook nor CI job (a hook cites another task only).
def unwired(repo):
    set_wiring(repo, {".pre-commit-config.yaml": precommit("check")})


def test_FRM004_green_task_not_wired(repo):
    unwired(repo)
    rc, out = run_green(repo)
    assert "NOT WIRED test:render" in out and out.splitlines()[-1] == "FAIL", out


def test_FRM004_characterize_task_not_wired(repo, mutants):
    unwired(repo)
    rc, out = run_char(repo, mutants)
    assert "NOT WIRED test:render" in out and out.splitlines()[-1] == "FAIL", out


# Once the task is cited in .pre-commit-config.yaml or .gitlab-ci.yml, the NOT WIRED line is gone.
WIRINGS = {
    ".pre-commit-config.yaml": {".pre-commit-config.yaml": precommit(TASK)},
    ".gitlab-ci.yml": {".pre-commit-config.yaml": precommit("check"), ".gitlab-ci.yml": GITLAB_CI},
}


@pytest.mark.parametrize("name", list(WIRINGS))
def test_FRM004_green_task_wired(repo, name):
    set_wiring(repo, WIRINGS[name])
    rc, out = run_green(repo)
    assert "GREEN CALC-003" in out and "NOT WIRED" not in out, out


def test_FRM004_characterize_task_wired(repo, mutants):
    set_wiring(repo, WIRINGS[".pre-commit-config.yaml"])
    rc, out = run_char(repo, mutants)
    assert "KILLED CALC-003" in out and "NOT WIRED" not in out, out


# The hook cites only `mise run check`, and test:render is a dependency of check.
def test_FRM004_green_task_wired_by_dependency(repo):
    unwired(repo)
    rc, out = run_green(repo, check_depends=TASK)
    assert "GREEN CALC-003" in out and "NOT WIRED" not in out, out
