"""Acceptance tests, slug revue-finale: FRM-007, PRV-011, PRV-012, PRV-013, RPL-003. Boundary: the harness gate command line."""
from test_tdd_gate import (  # noqa: F401  (fixtures are used by name)
    STAB_MD, TESTS_MD, char, char_setup, commit, gate, green_setup, make_patch, mise_toml, mutants, prepare_proofs,
    replay, repo, run_green, sh,
)

# ------------------------------------------------------------------ FRM-007
TASK = "test:render"
COMMAND = f"Branche : `feat/div` · Commande : `mise run {TASK}`\n"


def with_command(md):
    return md.replace("## div — Division\n", "## div — Division\n" + COMMAND)


# The hook runs `mise run test` (default wiring); [tasks.test] aggregates test:* by pattern.
def test_FRM007_green_task_reached_by_dependency_pattern(repo):
    t, f = green_setup(repo, md=with_command(TESTS_MD), task=TASK, test_depends=True)
    rc, out = gate(repo, "green", "--slug", "div", "--test", t, "--feat", f, "--task", TASK, "--proof-exempt")
    assert "NOT WIRED" not in out, out


def test_FRM007_characterize_task_reached_by_dependency_pattern(repo, mutants):
    sha = char_setup(repo, md=with_command(STAB_MD), mise_task=TASK, mise=mise_toml(task=TASK, test_depends=True))
    make_patch(repo, mutants, "CALC-003", {"src/state": "fail\n"})
    rc, out = char(repo, sha, mutants, task=TASK)
    assert "NOT WIRED" not in out, out


# ------------------------------------------------------------------ PRV-011
def test_PRV011_mutant_without_diff_git_header_is_judged(repo, mutants):
    sha = char_setup(repo)
    make_patch(repo, mutants, "CALC-003", {"src/state": "fail\n"}, bare=True)
    text = (mutants / "CALC-003.patch").read_text()
    assert text.startswith("--- a/src/state\n") and "diff --git" not in text and "\nindex " not in text
    rc, out = char(repo, sha, mutants)
    assert "KILLED CALC-003" in out, out


# ------------------------------------------------------------------ PRV-012
def test_PRV012_proofs_not_applied_on_modified_tree(repo, mutants):
    t = char_setup(repo, state="fail")
    f = commit(repo, {"src/state": "pass\n"}, "feat(div)")
    make_patch(repo, mutants, "CALC-003", {"src/state": "fail\n"})
    (repo / "src/note.txt").write_text("uncommitted edit\n")
    rc, out = run_green(repo, t, f, exempt=False, proofs=mutants)
    lines = out.splitlines()
    assert "TREE NOT CLEAN" in out, out
    assert not [ln for ln in lines if "MUTANT" in ln], out
    assert lines[-1] == "FAIL", out
    assert (repo / "src/note.txt").read_text() == "uncommitted edit\n"


# ------------------------------------------------------------------ PRV-013
PROD = "src/calc.txt"
NEEDED_TASK = (
    "mkdir -p .tdd\n"
    "if grep -q '^needed$' src/calc.txt; then cp tests/pass.xml .tdd/junit.xml; exit 0; fi\n"
    "cp tests/fail.xml .tdd/junit.xml\nexit 1\n"
)


def test_PRV013_unneeded_block_added_between_two_acceptance_commits(repo, mutants):
    first = char_setup(repo, files={PROD: "header\nmiddle\n"}, task=NEEDED_TASK)
    commit(repo, {PROD: "header\nmiddle\nextra\n"}, "chore: code no test asks for")
    second = commit(repo, {"tests/test_extra.py": "def test_x():\n    pass\n"}, "test(div): acceptance")
    feat = commit(repo, {PROD: "header\nneeded\nmiddle\nextra\n"}, "feat(div)")
    make_patch(repo, mutants, "CALC-003", {PROD: "header\nmiddle\nextra\n"})
    prepare_proofs(repo, mutants)
    # --since does not exist yet: red by usage error accepted here
    rc, out = gate(repo, "green", "--slug", "div", "--test", second, "--since", first, "--feat", feat,
                   "--task", "test", "--proofs", str(mutants))
    assert any(ln.startswith(f"UNNEEDED {PROD}:4-4") for ln in out.splitlines()), out
    assert out.splitlines()[-1] == "FAIL", out


# ------------------------------------------------------------------ RPL-003
def test_RPL003_failing_task_without_mutant_stops_the_replay(repo, mutants):
    first = commit(repo, {"run_test.sh": "exit 1\n", "mise.toml": mise_toml(), ".gitignore": ".tdd\n",
                          "src/state": "ok\n", "src/note.txt": "note\n"}, "first fails")
    make_patch(repo, mutants, "m-state", {"src/state": "fail\n"})
    second = commit(repo, {"run_test.sh": "exit 0\n"}, "second passes")
    rc, out = replay(repo, first, second, mutants, "test")
    assert "BASELINE NOT GREEN base" in out and "KEPT" not in out and out.splitlines()[-1] == "FAIL", out


def test_RPL003_swapped_base_and_head_names_head(repo, mutants):
    first = commit(repo, {"run_test.sh": "exit 1\n", "mise.toml": mise_toml(), ".gitignore": ".tdd\n",
                          "src/state": "ok\n", "src/note.txt": "note\n"}, "first fails")
    make_patch(repo, mutants, "m-state", {"src/state": "fail\n"})
    second = commit(repo, {"run_test.sh": "exit 0\n"}, "second passes")
    rc, out = replay(repo, second, first, mutants, "test")
    assert "BASELINE NOT GREEN head" in out and "KEPT" not in out and out.splitlines()[-1] == "FAIL", out
