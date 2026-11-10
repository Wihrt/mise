"""Acceptance tests, slug mutant-mise: PRV-008. Boundary: the tdd-gate command line."""
import re

from test_tdd_gate import (  # noqa: F401  (fixtures are used by name)
    STAB_MD, char_setup, gate, make_patch, mutants, precommit, repo, set_wiring,
)

TASK = "test:bootstrap"
COMMAND = f"Branche : `feat/div` · Commande : `mise run {TASK}`\n"
# The tested code is the body of [tasks."bootstrap:agent"]; the judged task is test:bootstrap.
# Both bodies are long multi-line strings: the mutated line is far below the table header, which a
# 3-line-context patch does not show.
def body(marker):
    pad = "".join(f"# step {i}\n" for i in range(1, 9))
    return f"{pad}{marker}\n# done\n# done\n# done\n"


BOOT = body("echo pass")
JUDGE = body("sh run_test.sh")
MISE = (
    f'[tasks."{TASK}"]\nrun = """\n{JUDGE}"""\n'
    f'[tasks.check]\nrun = "exit 0"\n'
    f'[tasks."bootstrap:agent"]\nrun = """\n{BOOT}"""\n'
)


def judge(repo, mutants, mutated_mise):
    set_wiring(repo, {".pre-commit-config.yaml": precommit(TASK)})
    md = STAB_MD.replace("Commande : `mise run test`", f"Commande : `mise run {TASK}`")
    sha = char_setup(repo, md=md, mise_task=TASK, mise=MISE)
    make_patch(repo, mutants, "CALC-003", {"mise.toml": mutated_mise})
    return gate(repo, "characterize", "--slug", "div", "--test", sha, "--task", TASK, "--spec", "spec.md",
                "--ac", "CALC-003", "--mutants", str(mutants))


def test_PRV008_mutant_of_code_task_is_judged(repo, mutants):
    rc, out = judge(repo, mutants, MISE.replace(BOOT, body("echo mutated")))
    assert re.search(r"^(KILLED|SURVIVED) CALC-003", out, re.M), out


def test_PRV008_mutant_of_test_task_is_refused(repo, mutants):
    rc, out = judge(repo, mutants, MISE.replace(JUDGE, body("sh run_test.sh && true")))
    assert re.search(r"^MUTANT INVALID CALC-003.*touches test task test:bootstrap", out, re.M), out
