"""Acceptance tests, slug mutant-mise: PRV-008. Boundary: the harness gate command line."""
import re

from test_tdd_gate import (  # noqa: F401  (fixtures are used by name)
    STAB_MD, char, char_setup, gate, make_patch, mutants, precommit, repo, set_wiring,
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


def judge(repo, mutants, mutated_mise=None, files=None, patch=None):
    set_wiring(repo, {".pre-commit-config.yaml": precommit(TASK)})
    md = STAB_MD.replace("Commande : `mise run test`", f"Commande : `mise run {TASK}`")
    sha = char_setup(repo, md=md, mise_task=TASK, mise=MISE, files=files)
    make_patch(repo, mutants, "CALC-003", patch or {"mise.toml": mutated_mise})
    return char(repo, sha, mutants, task=TASK)


def test_PRV008_mutant_of_code_task_is_judged(repo, mutants):
    rc, out = judge(repo, mutants, MISE.replace(BOOT, body("echo mutated")))
    assert re.search(r"^(KILLED|SURVIVED) CALC-003", out, re.M), out


def test_PRV008_mutant_of_test_task_is_refused(repo, mutants):
    rc, out = judge(repo, mutants, MISE.replace(JUDGE, body("sh run_test.sh && true")))
    assert re.search(r"^MUTANT INVALID CALC-003.*touches test task test:bootstrap", out, re.M), out


# Another file whose mutated line number is, in mise.toml, a line of the body of test:bootstrap.
OTHER = "".join(f"line {i}\n" for i in range(1, 31))


def test_PRV008_mutant_of_other_file_is_judged(repo, mutants):
    lines = MISE.splitlines()
    first = lines.index('[tasks."test:bootstrap"]') + 3  # 1-based number of the first body line
    last = first + JUDGE.count("\n") - 1
    n = first + 5  # mutated line: number inside the body of test:bootstrap
    assert first <= n <= last and n <= len(OTHER.splitlines()), (first, last, n)
    mutated = OTHER.replace(f"line {n}\n", f"mutated {n}\n")
    rc, out = judge(repo, mutants, files={"src/code.txt": OTHER}, patch={"src/code.txt": mutated})
    assert re.search(r"^(KILLED|SURVIVED) CALC-003", out, re.M), out
    assert "MUTANT INVALID" not in out, out
