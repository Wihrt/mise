"""Tests for tdd-gate. Run: uvx pytest ~/.claude/agents/bin/tests/test_tdd_gate.py"""
import importlib.machinery
import importlib.util
import json
import os
import subprocess
from pathlib import Path

import pytest

GATE = Path(__file__).resolve().parent.parent / "tdd-gate"
CMD = [os.environ["HARNESS"], "gate"]  # the harness binary under test (mise run test:tdd-gate builds it)
_loader = importlib.machinery.SourceFileLoader("tdd_gate", str(GATE))
_spec = importlib.util.spec_from_loader("tdd_gate", _loader)
tg = importlib.util.module_from_spec(_spec)
_loader.exec_module(tg)


# ------------------------------------------------------------------ unit tests
@pytest.mark.parametrize(
    "name,expected",
    [
        ("TestCALC001_Add", "CALC001"),
        ("TestCALC001_Add/sub_case", "CALC001"),
        ("test_CALC002_sub", "CALC002"),
        ("[ING-001] has a name", "ING001"),
        ("CALC003_sub_is_exact", "CALC003"),
        ("test_CALC_001_add", "CALC001"),
        ("TestSHA256Sum", None),  # legacy: digits followed by a letter
        ("TestHTTP2Handler", None),  # legacy: fewer than 3 digits
        ("test_S3_upload", None),
        ("TestMain", None),
    ],
)
def test_candidate(name, expected):
    assert tg.candidate(name) == expected


def test_section_ids_strips_trailing_colon():
    text = "## calc\n\n### CALC-001: title\n### CALC-002 other\n"
    assert tg.section_ids(text, "calc") == ["CALC-001", "CALC-002"]


def test_norm_both_sides():
    assert tg.norm("CALC-001") == "CALC001"
    assert tg.norm("TestCALC001") == "CALC001"
    assert tg.norm("test_CALC001") == "CALC001"


TESTS_MD = """## pow — Power
### CALC-001 — a
- **Critère** :
  - **Étant donné** x
  - **Quand** y
  - **Alors** z
- **Test** : `a_test.go` › `TestCALC001_A`
### CALC-002 — b
- **Critère** :
  - **Étant donné** x
  - **Quand** y
  - **Alors** z
- **Test** : `a_test.go` › `TestCALC002_B`

## div — Division
### CALC-003 — c
- **Critère** :
  - **Étant donné** x
  - **Quand** y
  - **Alors** z
- **Test** : `a_test.go` › `TestCALC003_C`
"""


def test_section_ids_multiple_sections():
    assert tg.section_ids(TESTS_MD, "pow") == ["CALC-001", "CALC-002"]
    assert tg.section_ids(TESTS_MD, "div") == ["CALC-003"]
    assert tg.section_ids(TESTS_MD, "nope") == []
    assert tg.all_ids(TESTS_MD) == {"CALC001", "CALC002", "CALC003"}


def test_incomplete_headings():
    md = TESTS_MD + "### CALC-004 — d\n- **Spec** : only spec\n"
    assert tg.incomplete_headings(TESTS_MD) == []
    assert tg.incomplete_headings(md) == ["### CALC-004 — d"]


def test_scan_tests_name_positions_and_legacy(tmp_path):
    (tmp_path / "a_test.go").write_text(
        "package x\n"
        "// TestCALC999_InComment mentions SHA-256 and ISO-8601\n"
        "func TestSHA256Sum(t *testing.T) {}\n"
        "func TestHTTP2Handler(t *testing.T) {}\n"
        "func TestCALC001_A(t *testing.T) { Foo() }\n"
        "func TestCALC002_B(t *testing.T) {}\n"
    )
    entries, pre = tg.scan_tests(tmp_path, ["a_test.go"])
    assert [e.cand for e in entries] == [None, None, "CALC001", "CALC002"]
    assert "Foo()" in entries[2].body and "Foo()" not in entries[3].body


def test_scan_tests_python_helm_tofu(tmp_path):
    (tmp_path / "t.py").write_text("def test_CALC001_add():\n    pass\ndef test_S3_upload():\n    pass\n")
    (tmp_path / "h.yaml").write_text("tests:\n  - it: \"[ING-002] x\"\n    asserts: []\n")
    (tmp_path / "m.tftest.hcl").write_text('run "CALC003_y" {\n}\n')
    entries, _ = tg.scan_tests(tmp_path, ["t.py", "h.yaml", "m.tftest.hcl"])
    assert [e.cand for e in entries] == ["CALC001", None, "ING002", "CALC003"]


JUNIT = """<testsuites><testsuite name="s">
<testcase name="TestCALC001_A"/>
<testcase name="TestCALC002_B"><failure message="x"/></testcase>
<testcase name="TestCALC003_C"><error message="x"/></testcase>
<testcase name="TestCALC004_D"><skipped/></testcase>
</testsuite></testsuites>"""


def test_load_junit_statuses(tmp_path):
    p = tmp_path / "j.xml"
    p.write_text(JUNIT)
    r = tg.load_junit(p)
    assert r.state == "ok"
    assert r.cases == [
        ("TestCALC001_A", "pass"),
        ("TestCALC002_B", "fail"),
        ("TestCALC003_C", "fail"),
        ("TestCALC004_D", "skipped"),
    ]
    assert set(tg.results_by_id(r)) == {"CALC001", "CALC002", "CALC003", "CALC004"}


def test_load_junit_missing_empty_garbage(tmp_path):
    assert tg.load_junit(tmp_path / "none.xml").state == "missing"
    (tmp_path / "e.xml").write_text("")
    assert tg.load_junit(tmp_path / "e.xml").state == "empty"
    (tmp_path / "n.xml").write_text("<testsuites><testsuite/></testsuites>")
    assert tg.load_junit(tmp_path / "n.xml").state == "empty"
    (tmp_path / "g.xml").write_text("not xml <<")
    assert tg.load_junit(tmp_path / "g.xml").state == "unreadable"


def test_load_tofu_json(tmp_path):
    ev = [
        {"type": "test_run", "test_run": {"run": "CALC001_ok", "status": "pass"}},
        {"type": "test_run", "test_run": {"run": "CALC002_bad", "status": "fail"}},
        {"type": "test_run", "test_run": {"run": "CALC003_unk", "status": "error"}},
        {"type": "diagnostic", "diagnostic": {"summary": "Reference to undeclared output value",
                                              "detail": 'An output value with the name "missing" has not been declared.'}},
    ]
    p = tmp_path / "t.json"
    p.write_text("\n".join(json.dumps(e) for e in ev))
    r = tg.load_tofu_json(p)
    assert [c[1] for c in r.cases] == ["pass", "fail", "fail"]
    errs = tg.compile_errors(r.diag)
    assert "missing" in errs[0][1]


@pytest.mark.parametrize(
    "line,sym",
    [
        ("internal/calc/div_test.go:9:14: undefined: Div", "Div"),
        ("NameError: name 'mul' is not defined", "mul"),
        ("ImportError: cannot import name 'mul' from 'pkg.calc'", "mul"),
        ("E   ModuleNotFoundError: No module named 'pkg.newmod'", "newmod"),
    ],
)
def test_compile_errors_symbols(line, sym):
    (found, syms), = tg.compile_errors(line)
    assert sym in syms


# ----------------------------------------------------------- integration tests
def sh(repo, *cmd):
    return subprocess.run(cmd, cwd=repo, check=True, capture_output=True, text=True).stdout.strip()


def commit(repo, files, msg):
    if subprocess.run(["git", "rev-parse", "--verify", "-q", "HEAD"], cwd=repo, capture_output=True).returncode:
        # first commit: the judged task is wired in a commit hook unless set_wiring() says otherwise
        files = {**FIRST_WIRING.get(str(repo), {".pre-commit-config.yaml": precommit("test")}), **files}
    for name, content in files.items():
        p = repo / name
        p.parent.mkdir(parents=True, exist_ok=True)
        p.write_text(content)
    sh(repo, "git", "add", "-A")
    sh(repo, "git", "-c", "user.name=t", "-c", "user.email=t@t", "commit", "-q", "--no-verify", "-m", msg)
    return sh(repo, "git", "rev-parse", "HEAD")


def task_script(report, rc=None, extra=""):
    """Canned task: copy green.xml if the feat commit added it, else report.xml; exit 1 on any failure."""
    body = f"mkdir -p .tdd\n{extra}\nf={report}\n[ -f green.xml ] && f=green.xml\ncp $f .tdd/junit.xml\n"
    if rc is None:
        return body + "if grep -q '<failure\\|<error' $f; then exit 1; fi\nexit 0\n"
    return body + f"exit {rc}\n"


def mise_toml(check_rc=0, tools="", task="test", check_depends=None, test_depends=False):
    """test_depends: add the aggregate `[tasks.test] depends = ["test:*"]` (for a task such as test:render)."""
    return (
        f"[tools]\n{tools}\n[tasks.\"{task}\"]\nrun = \"sh run_test.sh\"\n"
        f"{'[tasks.test]' + chr(10) + 'depends = [\"test:*\"]' + chr(10) if test_depends else ''}"
        f"[tasks.check]\n{f'depends = [\"{check_depends}\"]' + chr(10) if check_depends else ''}run = \"exit {check_rc}\"\n"
    )


def junit(*cases):
    body = ""
    for name, st in cases:
        child = {"pass": "", "fail": "<failure message='x'/>", "error": "<error message='x'/>"}[st]
        body += f"<testcase name='{name}'>{child}</testcase>"
    return f"<testsuites><testsuite name='s'>{body}</testsuite></testsuites>"


GO_TESTS = (
    "package x\n"
    "func TestSHA256Sum(t *testing.T) {}\n"
    "func TestCALC001_A(t *testing.T) { A() }\n"
    "func TestCALC002_B(t *testing.T) { Div() }\n"
    "func TestCALC003_C(t *testing.T) {}\n"
)


def precommit(*tasks):
    """A .pre-commit-config.yaml whose local hooks cite `mise run <task>` for each task."""
    hooks = "".join(
        f"      - id: h{i}\n        name: h{i}\n        entry: mise run {t}\n        language: system\n        pass_filenames: false\n"
        for i, t in enumerate(tasks)
    )
    return f"repos:\n  - repo: local\n    hooks:\n{hooks}"


FIRST_WIRING = {}  # str(repo) -> wiring files of the repo's first commit (default: a hook citing `mise run test`)


def set_wiring(repo, files):
    """Choose the wiring files the repo's first commit carries; call before the setup. Files not given are absent."""
    FIRST_WIRING[str(repo)] = files


@pytest.fixture
def repo(tmp_path, monkeypatch):
    monkeypatch.setenv("MISE_TRUSTED_CONFIG_PATHS", str(tmp_path))
    monkeypatch.setenv("MISE_YES", "1")
    r = tmp_path / "repo"
    r.mkdir()
    sh(r, "git", "init", "-q", "-b", "main")
    return r


def gate(repo, *args):
    p = subprocess.run([*CMD, *args, "--repo", str(repo)], capture_output=True, text=True)
    return p.returncode, p.stdout


def replay(repo, base, head, proofs, tasks, *extra):
    """Run `tdd-gate replay` on two commits of repo; tasks is a comma list."""
    return gate(repo, "replay", "--base", base, "--head", head, "--proofs", str(proofs), "--task", tasks, *extra)


def base_commit(repo, report, rc=None, md=TESTS_MD, extra="", mise=None):
    return commit(
        repo,
        {
            "TESTS.md": md,
            "a_test.go": GO_TESTS,
            "mise.toml": mise or mise_toml(),
            "run_test.sh": task_script("report.xml", rc, extra),
            "report.xml": report,
        },
        "test(div)",
    )


def test_red_pass_all_fail(repo):
    sha = base_commit(repo, junit(("TestCALC001_A", "pass"), ("TestCALC002_B", "fail"), ("TestCALC003_C", "fail")))
    # section pow = CALC-001, CALC-002 -> CALC001 passes: not red
    rc, out = gate(repo, "red", "--slug", "pow", "--test", sha, "--task", "test")
    assert rc == 1 and "NOT RED CALC-001" in out and out.splitlines()[-1] == "FAIL"
    # section div = CALC-003 -> red; the other sections' state is ignored
    rc, out = gate(repo, "red", "--slug", "div", "--test", sha, "--task", "test")
    assert rc == 0 and "RED CALC-003" in out and out.splitlines()[-1] == "PASS"


def test_red_missing_test(repo):
    md = TESTS_MD + "### CALC-009 — z\n- **Critère** :\n  - **Étant donné** x\n  - **Quand** y\n  - **Alors** z\n- **Test** : `a_test.go` › `TestCALC009_Z`\n"  # in section div
    sha = base_commit(repo, junit(("TestCALC003_C", "fail")), md=md)
    rc, out = gate(repo, "red", "--slug", "div", "--test", sha, "--task", "test")
    assert rc == 1 and "MISSING CALC-009" in out


def test_red_task_passed(repo):
    sha = base_commit(repo, junit(("TestCALC003_C", "pass")), rc=0)
    rc, out = gate(repo, "red", "--slug", "div", "--test", sha, "--task", "test")
    assert rc == 1 and "TASK PASSED, NOT RED" in out and "NOT RED CALC-003" in out


def test_red_compile_error_excused_when_symbol_used(repo):
    report = junit(("TestMain", "fail"))
    sha = base_commit(repo, report, extra="echo 'x_test.go:9:14: undefined: Div'")
    rc, out = gate(repo, "red", "--slug", "pow", "--test", sha, "--task", "test")
    # CALC-001 uses A(), not Div: not excused. CALC-002 uses Div: excused.
    assert "COMPILE OR COLLECT ERROR: x_test.go:9:14: undefined: Div" in out
    assert "RED CALC-002" in out and "NOT RED CALC-001" in out and rc == 1


def test_red_compile_error_all_ids_use_symbol(repo):
    tests = GO_TESTS.replace("A()", "Div()")
    sha = commit(repo, {"TESTS.md": TESTS_MD, "a_test.go": tests, "mise.toml": mise_toml(),
                        "run_test.sh": "echo 'undefined: Div'\nexit 1\n"}, "test")
    rc, out = gate(repo, "red", "--slug", "pow", "--test", sha, "--task", "test")
    assert rc == 0 and "excused by compile/collect error" in out


def test_red_missing_report_without_symbol_fails(repo):
    sha = commit(repo, {"TESTS.md": TESTS_MD, "a_test.go": GO_TESTS, "mise.toml": mise_toml(),
                        "run_test.sh": "exit 1\n"}, "test")
    rc, out = gate(repo, "red", "--slug", "div", "--test", sha, "--task", "test")
    assert rc == 1 and "REPORT" in out and "missing" in out and "NOT RED CALC-003" in out


def test_red_stale_report_is_not_trusted(repo):
    sha = commit(repo, {"TESTS.md": TESTS_MD, "a_test.go": GO_TESTS, "mise.toml": mise_toml(),
                        "run_test.sh": "exit 1\n"}, "test")
    (repo / ".tdd").mkdir()
    (repo / ".tdd/junit.xml").write_text(junit(("TestCALC003_C", "fail")))
    rc, out = gate(repo, "red", "--slug", "div", "--test", sha, "--task", "test")
    assert rc == 1


def test_no_ids_and_usage_errors(repo):
    sha = base_commit(repo, junit())
    rc, out = gate(repo, "red", "--slug", "ghost", "--test", sha, "--task", "test")
    assert rc == 2 and "NO IDS FOUND" in out and out.splitlines()[-1] == "FAIL"
    p = subprocess.run([*CMD, "green", "--slug", "x"], capture_output=True, text=True)
    assert p.returncode == 2


def green_setup(repo, feat_files=None, feat_report=None, check_rc=0, feat_mise=None, md=TESTS_MD, task="test", check_depends=None, test_depends=False):
    test_sha = base_commit(repo, junit(("TestCALC003_C", "fail")), md=md, mise=mise_toml(task=task, check_depends=check_depends, test_depends=test_depends))
    files = dict(feat_files or {})
    files["green.xml"] = feat_report or junit(("TestCALC001_A", "pass"), ("TestCALC002_B", "pass"), ("TestCALC003_C", "pass"))
    if feat_mise:
        files["mise.toml"] = feat_mise
    elif check_rc:
        files["mise.toml"] = mise_toml(check_rc=check_rc)
    files.setdefault("impl.go", "package x\n")
    return test_sha, commit(repo, files, "feat(div)")


def run_green(repo, t, f, exempt=True, proofs=None, neutral=True, ac="CALC-003", unneeded_ok=(), options=()):
    """Run green. By default --proof-exempt; exempt=False drops it; proofs=<dir> adds --proofs <dir>, the directory
    being prepared as char does (valid neutral changes for each criterion of ac, see prepare_proofs);
    unneeded_ok: blocks (<file>:<start>-<end>) passed each as --unneeded-ok; options: further options, passed as is."""
    extra = list(options)
    for block in unneeded_ok:
        extra += ["--unneeded-ok", block]
    if proofs is not None:
        prepare_proofs(repo, proofs, neutral, ac)
        extra += ["--proofs", str(proofs)]
    if exempt:
        extra.append("--proof-exempt")
    return gate(repo, "green", "--slug", "div", "--test", t, "--feat", f, "--task", "test", *extra)


def test_green_pass(repo):
    t, f = green_setup(repo)
    rc, out = run_green(repo, t, f)
    assert rc == 0 and "GREEN CALC-003" in out and out.splitlines()[-1] == "PASS"


def test_green_tools_only_mise_change_ok(repo):
    t, f = green_setup(repo, feat_mise=mise_toml(tools='foo = "1"\n'))
    rc, out = run_green(repo, t, f)
    assert rc == 0, out


def test_green_mise_tasks_changed(repo):
    t, f = green_setup(repo, feat_mise=mise_toml().replace("exit 0", "true"))
    rc, out = run_green(repo, t, f)
    assert rc == 1 and "PROTECTED FILE MODIFIED: mise.toml (tasks)" in out


def test_green_protected_test_file_modified(repo):
    t, f = green_setup(repo, feat_files={"a_test.go": GO_TESTS + "// reformat\n"})
    rc, out = run_green(repo, t, f)
    assert rc == 1 and "PROTECTED FILE MODIFIED: a_test.go" in out


def test_green_tests_md_modified(repo):
    t, f = green_setup(repo, feat_files={"TESTS.md": TESTS_MD + "\n"})
    rc, out = run_green(repo, t, f)
    assert rc == 1 and "PROTECTED FILE MODIFIED: TESTS.md" in out


def test_green_failing_testcase(repo):
    t, f = green_setup(repo, feat_report=junit(("TestCALC003_C", "fail")))
    rc, out = run_green(repo, t, f)
    assert rc == 1 and "NOT GREEN CALC-003" in out


def test_green_skipped_testcase_is_not_green(repo):
    rep = "<testsuites><testsuite><testcase name='TestCALC003_C'><skipped/></testcase></testsuite></testsuites>"
    t, f = green_setup(repo, feat_report=rep)
    rc, out = run_green(repo, t, f)
    assert rc == 1 and "NOT GREEN CALC-003" in out


def test_green_regression_of_earlier_slug(repo):
    rep = junit(("TestCALC001_A", "fail"), ("TestCALC002_B", "pass"), ("TestCALC003_C", "pass"))
    t, f = green_setup(repo, feat_report=rep)
    rc, out = run_green(repo, t, f)
    assert rc == 1 and "REGRESSION CALC-001" in out and "GREEN CALC-003" in out
    assert out.splitlines()[-1] == "FAIL"


def test_green_earlier_id_without_result(repo):
    t, f = green_setup(repo, feat_report=junit(("TestCALC001_A", "pass"), ("TestCALC003_C", "pass")))
    rc, out = run_green(repo, t, f)
    assert rc == 1 and "NO RESULT CALC-002" in out and "GREEN CALC-003" in out


def test_green_check_fails(repo):
    t, f = green_setup(repo, check_rc=3)
    rc, out = run_green(repo, t, f)
    assert rc == 1
    # mise.toml [tasks.check] changed too, so both signals appear
    assert "CHECK FAILED" in out


def test_green_id_without_test_and_test_without_id(repo):
    md = TESTS_MD + "### CALC-008 — ghost\n- **Critère** :\n  - **Étant donné** x\n  - **Quand** y\n  - **Alors** z\n- **Test** : `a_test.go` › `TestCALC008_G`\n"
    t = base_commit(repo, junit(("TestCALC003_C", "fail")), md=md)
    rc, out = run_green(repo, t, t)
    assert rc == 1 and "ID WITHOUT TEST: CALC008" in out
    # a real test with an ID that TESTS.md does not list
    t2 = commit(repo, {"TESTS.md": TESTS_MD, "a_test.go": GO_TESTS + "func TestCALC042_Z(t *testing.T) {}\n"}, "x")
    rc, out = run_green(repo, t2, t2)
    assert rc == 1 and "TEST WITHOUT ID: CALC042" in out and "SHA256" not in out


def test_green_incomplete_section(repo):
    md = TESTS_MD + "### CALC-010 — half\n- **Spec** : x\n"
    t = base_commit(repo, junit(("TestCALC003_C", "pass")), md=md, rc=0)
    rc, out = run_green(repo, t, t)
    assert rc == 1 and "INCOMPLETE: ### CALC-010 — half" in out


def test_tofu_json_format(repo):
    ev = "\n".join(json.dumps(e) for e in [
        {"type": "test_run", "test_run": {"run": "CALC003_c", "status": "fail"}},
    ])
    sha = commit(repo, {
        "TESTS.md": TESTS_MD.replace("a_test.go", "m.tftest.hcl"),
        "m.tftest.hcl": 'run "CALC001_a" {\n}\nrun "CALC002_b" {\n}\nrun "CALC003_c" {\n}\n',
        "mise.toml": mise_toml(),
        "run_test.sh": f"mkdir -p .tdd\ncat > .tdd/tofu.json <<'E'\n{ev}\nE\nexit 1\n",
    }, "test")
    rc, out = gate(repo, "red", "--slug", "div", "--test", sha, "--task", "test", "--format", "tofu-json")
    rc2, out2 = gate(repo, "red", "--slug", "div", "--test", sha, "--task", "test")
    assert rc2 == 0 and "tofu.json" in out2
    assert rc == 0 and "RED CALC-003" in out


@pytest.mark.parametrize("mode", ["red", "green"])
def test_default_timeout_is_540(mode):
    argv = [mode, "--slug", "s", "--test", "abc", "--task", "t"]
    if mode == "green":
        argv += ["--feat", "def"]
    assert tg.build_parser().parse_args(argv).timeout == 540


# ------------------------------------------------------------ acceptance mode
SPEC = ("# s\n## Critères d'acceptation\n### CALC-003 — c\n- **Alors** x\n"
        "### CALC-001 — a\n- **Alors** x\n### CALC-002 — b\n- **Alors** x\n## Annexe\n### CALC-009 — not a criterion\n")
CRIT = ["CALC-003", "CALC-001", "CALC-002"]


def acc_commit(repo, report, extra=None):
    """Acceptance commit holding test paths only; spec and task script live in a preceding base commit."""
    if not (repo / "run_test.sh").exists():
        commit(repo, {"run_test.sh": task_script("tests/report.xml"), "spec.md": SPEC}, "base")
    return commit(
        repo,
        {"TESTS.md": TESTS_MD, "a_test.go": GO_TESTS, "mise.toml": mise_toml(),
         "tests/report.xml": report, **(extra or {})},
        "test(div): acceptance",
    )


def test_spec_criteria_parsing():
    assert tg.spec_criteria(SPEC) == CRIT
    assert tg.spec_criteria(SPEC.replace("Critères d'acceptation", "Acceptance criteria")) == CRIT
    assert tg.spec_criteria("# s\n### CALC-003 — c\n") == []


def test_incomplete_headings_accepts_critere():
    assert tg.incomplete_headings("### ING-002 — t\n- **Critère** : `s.md` › ING-002\n- **Test** : `a` › `b`\n") == []


def test_red_acceptance_pass(repo):
    sha = acc_commit(repo, junit(("TestCALC003_C", "fail")))
    rc, out = gate(repo, "red", "--slug", "div", "--test", sha, "--task", "test", "--spec", "spec.md", "--ac", "CALC-003")
    assert rc == 0 and "CRITERIA: CALC-003" in out and "RED CALC-003" in out


def test_red_ac_not_in_spec(repo):
    sha = acc_commit(repo, junit(("TestCALC003_C", "fail")))
    rc, out = gate(repo, "red", "--slug", "div", "--test", sha, "--task", "test", "--spec", "spec.md", "--ac", "CALC-003,CALC-004")
    assert rc == 1 and "NOT IN SPEC CALC-004" in out


def test_red_ac_section_mismatch(repo):
    sha = acc_commit(repo, junit(("TestCALC001_A", "fail"), ("TestCALC002_B", "fail"), ("TestCALC003_C", "fail")))
    rc, out = gate(repo, "red", "--slug", "pow", "--test", sha, "--task", "test", "--spec", "spec.md", "--ac", "CALC-001")
    assert rc == 1 and "NOT IN --ac CALC-002" in out
    rc, out = gate(repo, "red", "--slug", "div", "--test", sha, "--task", "test", "--spec", "spec.md", "--ac", "CALC-003,CALC-001")
    assert rc == 1 and "NOT IN SECTION CALC-001" in out


def test_red_acceptance_refuses_compile_excuse(repo):
    tests = GO_TESTS.replace("A()", "Div()")
    commit(repo, {"run_test.sh": "echo 'undefined: Div'\nexit 1\n", "spec.md": SPEC}, "base")
    sha = commit(repo, {"TESTS.md": TESTS_MD, "a_test.go": tests, "mise.toml": mise_toml()}, "test")
    args = ("red", "--slug", "pow", "--test", sha, "--task", "test", "--spec", "spec.md", "--ac", "CALC-001,CALC-002")
    rc, out = gate(repo, *args)
    assert rc == 1 and "NOT RED CALC-001: compile/collect error excuse refused in acceptance mode" in out
    rc, out = gate(repo, *args, "--allow-compile-red")
    assert rc == 0 and "excused by compile/collect error" in out


def test_red_acceptance_rejects_production_file(repo):
    sha = acc_commit(repo, junit(("TestCALC003_C", "fail")), {"pkg/calc.go": "package x\n"})
    args = ("red", "--slug", "div", "--test", sha, "--task", "test", "--spec", "spec.md", "--ac", "CALC-003")
    rc, out = gate(repo, *args)
    assert rc == 1 and "NON-TEST FILE IN ACCEPTANCE COMMIT: pkg/calc.go" in out
    ok = acc_commit(repo, junit(("TestCALC003_C", "fail")),
                    {"tests/x_test.go": "package x\n", "chart/tests/acceptance/a.yaml": "a: 1\n",
                     "cmd/testdata/p.json": "{}\n", ".gitignore": ".tdd\n", "conftest.py": "\n",
                     "m.tftest.hcl": "\n", "test_a.py": "\n"})
    rc, out = gate(repo, *args[:4], ok, *args[5:])
    assert rc == 0 and "NON-TEST FILE" not in out, out


def test_red_spec_ac_usage(repo):
    sha = acc_commit(repo, junit(("TestCALC003_C", "fail")))
    base = ("red", "--slug", "div", "--test", sha, "--task", "test")
    assert gate(repo, *base, "--ac", "CALC-003")[0] == 2
    assert gate(repo, *base, "--spec", "spec.md")[0] == 2


def test_run_mise_sets_goflags_count1(repo, monkeypatch):
    commit(repo, {"mise.toml": "[tasks.t]\nrun = 'echo \"$GOFLAGS\" > goflags.txt'\n"}, "x")
    monkeypatch.delenv("GOFLAGS", raising=False)
    assert tg.run_mise(repo, "t", 60)[0] == 0
    assert (repo / "goflags.txt").read_text().split() == ["-count=1"]
    monkeypatch.setenv("GOFLAGS", "-mod=mod")
    assert tg.run_mise(repo, "t", 60)[0] == 0
    assert (repo / "goflags.txt").read_text().split() == ["-mod=mod", "-count=1"]
    monkeypatch.setenv("GOFLAGS", "-count=1 -mod=mod")
    assert tg.run_mise(repo, "t", 60)[0] == 0
    assert (repo / "goflags.txt").read_text().split() == ["-count=1", "-mod=mod"]


# ------------------------------------------------------------ stability check
def test_red_spec_missing_file(repo):
    sha = acc_commit(repo, junit(("TestCALC003_C", "fail")))
    rc, out = gate(repo, "red", "--slug", "div", "--test", sha, "--task", "test", "--spec", "nope.md", "--ac", "CALC-003")
    assert rc == 2 and "not found" in out


def test_section_test_files():
    assert tg.section_test_files(TESTS_MD) == {"pow": {"a_test.go"}, "div": {"a_test.go"}}


STAB_MD = TESTS_MD.replace("`a_test.go` › `TestCALC003_C`", "`d_test.go` › `TestCALC003_C`").replace("`a_test.go`", "`p_test.go`")
P_TESTS = "package x\nfunc TestCALC001_A(t *testing.T) {}\nfunc TestCALC002_B(t *testing.T) {}\n"
D_TESTS = "package x\nfunc TestCALC003_C(t *testing.T) {}\n"
GREEN_ALL = junit(("TestCALC001_A", "pass"), ("TestCALC002_B", "pass"), ("TestCALC003_C", "pass"))


def stability_setup(repo, test_edits_p=True, feat_files=None):
    commit(repo, {"TESTS.md": STAB_MD, "p_test.go": P_TESTS, "d_test.go": D_TESTS, "mise.toml": mise_toml(),
                  "run_test.sh": task_script("report.xml"), "report.xml": junit(("TestCALC001_A", "pass"))}, "base")
    files = {"report.xml": junit(("TestCALC003_C", "fail"))}
    if test_edits_p:
        files["p_test.go"] = P_TESTS.replace("TestCALC001_A(t *testing.T) {}", "TestCALC001_A(t *testing.T) { _ = 1 }")
    t = commit(repo, files, "test(div): acceptance")
    f = commit(repo, {"green.xml": GREEN_ALL, "impl.go": "package x\n", **(feat_files or {})}, "feat(div)")
    return t, f


def test_green_stability_blocks_earlier_test(repo):
    t, f = stability_setup(repo)
    rc, out = run_green(repo, t, f)
    assert rc == 1 and "EARLIER ACCEPTANCE TEST MODIFIED: p_test.go" in out


def test_green_stability_includes_acceptance_commit(repo):
    t, f = stability_setup(repo)
    rc, out = gate(repo, "green", "--slug", "div", "--test", t, "--feat", f, "--task", "test", "--ac-changed", "CALC-001", "--proof-exempt")
    assert rc == 0 and "CRITERION CHANGE ALLOWED: p_test.go (CALC-001)" in out


def test_green_stability_feat_edit(repo):
    t, f = stability_setup(repo, test_edits_p=False, feat_files={"p_test.go": P_TESTS + "// x\n"})
    rc, out = run_green(repo, t, f)
    assert rc == 1 and "EARLIER ACCEPTANCE TEST MODIFIED: p_test.go" in out


def test_green_unit_test_without_id_ignored(repo):
    assert tg.candidate("TestParseHelper") is None
    t, f = stability_setup(repo, test_edits_p=False,
                           feat_files={"helper_test.go": "package x\nfunc TestParseHelper(t *testing.T) {}\n"})
    rc, out = run_green(repo, t, f)
    assert rc == 0, out


# ------------------------------------------------------------ characterize mode
CHAR_TASK = (
    "mkdir -p .tdd\ns=$(cat src/state)\ncp tests/$s.xml .tdd/junit.xml\n"
    "[ \"$s\" = slow ] && sleep 3\n[ \"$s\" = fail ] && exit 1\nexit 0\n"
)
CHAR_REPORTS = {
    "tests/pass.xml": GREEN_ALL,
    "tests/slow.xml": GREEN_ALL,
    "tests/fail.xml": junit(("TestCALC001_A", "pass"), ("TestCALC002_B", "pass"), ("TestCALC003_C", "fail")),
    "tests/none.xml": junit(("TestCALC001_A", "pass"), ("TestCALC002_B", "pass")),
}


def spec_for(md):
    """SPEC with, for each criterion whose entry in md has a sub-list Alors, as many clauses as that entry has
    (a one-line Alors keeps SPEC's single line)."""
    spec = SPEC
    for id_ in ("CALC-001", "CALC-002", "CALC-003"):
        marker = f"### {id_} — "
        if marker not in md:
            continue
        lines = md.split(marker, 1)[1].split("\n")
        n = 1
        for i, line in enumerate(lines):
            if line.strip() == "- **Alors**":
                n = 0
                while lines[i + 1 + n].startswith("    - "):
                    n += 1
                break
        if n != 1:
            head, sep, tail = spec.partition(marker)
            tail = tail.replace("- **Alors** x\n", "- **Alors**\n" + "".join(f"  - clause {k}\n" for k in range(n)), 1)
            spec = head + sep + tail
    return spec


def char_setup(repo, state="pass", md=None, task=None, mise_task="test", mise=None, files=None, acc_files=None, base_md=None):
    """Base commit (pow's tests, task, spec, src/state), then an acceptance commit whose tests are already green.
    acc_files: files added to (or replacing those of) the acceptance commit, e.g. a second judged criterion.
    base_md: the TESTS.md of the base commit (default: pow's section of STAB_MD)."""
    pow_md = base_md or STAB_MD[: STAB_MD.index("## div")]
    commit(repo, {"run_test.sh": task or CHAR_TASK, "spec.md": spec_for(md or STAB_MD), "src/state": state + "\n", "src/note.txt": "note\n",
                  ".gitignore": ".tdd\n", "TESTS.md": pow_md, "p_test.go": P_TESTS, "mise.toml": mise or mise_toml(task=mise_task), **(files or {})}, "base")
    return commit(repo, {"TESTS.md": md or STAB_MD, "d_test.go": D_TESTS, **CHAR_REPORTS, **(acc_files or {})}, "test(div): acceptance")


def make_patch(repo, mutants, id_, files, bare=False):
    """Write <mutants>/<id_>.patch (git diff of `files` against HEAD), leaving the tree clean.
    bare: drop the `diff --git` and `index` header lines (the patch starts at `--- a/<file>`)."""
    for name, content in files.items():
        p = repo / name
        p.parent.mkdir(parents=True, exist_ok=True)
        p.write_text(content)
    sh(repo, "git", "add", "-A")
    diff = subprocess.run(["git", "diff", "--cached"], cwd=repo, check=True, capture_output=True, text=True).stdout
    sh(repo, "git", "reset", "-q", "--hard")
    sh(repo, "git", "clean", "-fdq")
    if bare:
        diff = "".join(ln for ln in diff.splitlines(keepends=True) if not ln.startswith(("diff --git", "index ")))
    mutants.mkdir(parents=True, exist_ok=True)
    (mutants / f"{id_}.patch").write_text(diff)


DEFAULT_NEUTRAL = (
    "diff --git a/src/note.txt b/src/note.txt\n--- a/src/note.txt\n+++ b/src/note.txt\n@@ -1 +1 @@\n-note\n+neutral\n"
)


def prepare_proofs(repo, mutants, neutral=True, ac="CALC-003"):
    """Complete the proofs directory with a neutral change for each criterion of ac (comma list), unless one is
    already there; neutral=False leaves them out, neutral={file: content} replaces CALC-003's by that change."""
    if isinstance(neutral, dict):
        make_patch(repo, mutants, "CALC-003.neutral", neutral)
    elif neutral:
        mutants.mkdir(parents=True, exist_ok=True)
        for id_ in ac.split(","):
            if not (mutants / f"{id_}.neutral.patch").exists():
                (mutants / f"{id_}.neutral.patch").write_text(DEFAULT_NEUTRAL)


def char(repo, sha, mutants, *extra, neutral=True, task="test", ac="CALC-003"):
    """Run characterize on CALC-003 (ac: comma list of the judged criteria, each getting its default neutral change). The proofs directory gets a valid CALC-003.neutral.patch unless one is already
    there; neutral=False leaves it out, neutral={file: content} replaces it by that change;
    task is the judged mise task."""
    prepare_proofs(repo, mutants, neutral, ac)
    return gate(repo, "characterize", "--slug", "div", "--test", sha, "--task", task, "--spec", "spec.md",
                "--ac", ac, "--mutants", str(mutants), *extra)


@pytest.fixture
def mutants(repo):
    return repo.parent / "mutants"


def test_characterize_killed(repo, mutants):
    sha = char_setup(repo)
    make_patch(repo, mutants, "CALC-003", {"src/state": "fail\n"})
    rc, out = char(repo, sha, mutants)
    assert rc == 0 and "KILLED CALC-003 clause 1: TestCALC003_C" in out and out.splitlines()[-1] == "PASS", out
    assert sh(repo, "git", "status", "--short") == ""


def test_characterize_survived(repo, mutants):
    sha = char_setup(repo)
    make_patch(repo, mutants, "CALC-003", {"src/note.txt": "# comment\n"})
    rc, out = char(repo, sha, mutants)
    assert rc == 1 and "SURVIVED CALC-003 clause 1" in out and out.splitlines()[-1] == "FAIL", out
    assert sh(repo, "git", "status", "--short") == ""


def test_characterize_missing_patch(repo, mutants):
    sha = char_setup(repo)
    rc, out = char(repo, sha, mutants)
    assert rc == 1 and "MUTANT MISSING CALC-003 clause 1" in out and out.splitlines()[-1] == "FAIL", out


def test_characterize_patch_not_applying(repo, mutants):
    sha = char_setup(repo)
    mutants.mkdir()
    (mutants / "CALC-003.patch").write_text(
        "diff --git a/src/note.txt b/src/note.txt\n--- a/src/note.txt\n+++ b/src/note.txt\n"
        "@@ -1 +1 @@\n-not the real content\n+changed\n")
    rc, out = char(repo, sha, mutants)
    assert rc == 1 and "MUTANT INVALID CALC-003 clause 1" in out, out
    assert sh(repo, "git", "status", "--short") == ""


def test_characterize_patch_touches_test_path(repo, mutants):
    sha = char_setup(repo)
    make_patch(repo, mutants, "CALC-003", {"src/state": "fail\n", "src/x_test.go": "package x\n"})
    rc, out = char(repo, sha, mutants)
    assert rc == 1 and "MUTANT INVALID CALC-003 clause 1: touches test path src/x_test.go" in out, out
    assert "KILLED" not in out and sh(repo, "git", "status", "--short") == ""


def test_characterize_no_result(repo, mutants):
    sha = char_setup(repo)
    make_patch(repo, mutants, "CALC-003", {"src/state": "none\n"})
    rc, out = char(repo, sha, mutants)
    assert rc == 1 and "MUTANT INVALID CALC-003 clause 1: no result" in out, out
    assert sh(repo, "git", "status", "--short") == ""


def test_characterize_baseline_red(repo, mutants):
    sha = char_setup(repo, state="fail")
    make_patch(repo, mutants, "CALC-003", {"src/note.txt": "x\n"})
    rc, out = char(repo, sha, mutants)
    assert rc == 1 and "BASELINE NOT GREEN CALC-003: TestCALC003_C fail" in out, out
    assert "MUTANT" not in out.replace("MUTANT INVALID", "")


def test_characterize_restores_tree_with_new_file(repo, mutants):
    sha = char_setup(repo)
    make_patch(repo, mutants, "CALC-003", {"src/state": "fail\n", "src/new.txt": "new\n"})
    rc, out = char(repo, sha, mutants)
    assert rc == 0 and "KILLED CALC-003 clause 1" in out, out
    assert sh(repo, "git", "status", "--short") == "" and not (repo / "src/new.txt").exists()


def test_characterize_timeout_restores_tree(repo, mutants):
    sha = char_setup(repo)
    make_patch(repo, mutants, "CALC-003", {"src/state": "slow\n", "src/new.txt": "new\n"})
    rc, out = char(repo, sha, mutants, "--timeout", "1")
    assert rc == 2 and "MUTANT CALC-003 clause 1: patch applied" in out and "TIMEOUT after 1s" in out, out
    assert sh(repo, "git", "status", "--short") == "" and not (repo / "src/new.txt").exists()


def test_characterize_dirty_tree(repo, mutants):
    sha = char_setup(repo)
    make_patch(repo, mutants, "CALC-003", {"src/state": "fail\n"})
    (repo / "junk.txt").write_text("x\n")
    rc, out = char(repo, sha, mutants)
    assert rc == 1 and "TREE NOT CLEAN" in out and "KILLED" not in out, out


def test_characterize_head_not_test(repo, mutants):
    sha = char_setup(repo)
    make_patch(repo, mutants, "CALC-003", {"src/state": "fail\n"})
    commit(repo, {"src/note.txt": "later\n"}, "feat")
    rc, out = char(repo, sha, mutants)
    assert rc == 1 and "HEAD IS NOT --test" in out and "KILLED" not in out, out


def test_characterize_ac_changed_parsed():
    argv = ["characterize", "--slug", "s", "--test", "x", "--task", "t", "--spec", "s.md", "--ac", "A-001", "--mutants", "m", "--ac-changed", "A-001,B-002"]
    assert tg.build_parser().parse_args(argv).ac_changed == ["A-001", "B-002"]


def test_characterize_ac_changed_allows_earlier_test(repo, mutants):
    commit(repo, {"run_test.sh": CHAR_TASK, "spec.md": SPEC, "src/state": "pass\n", "src/note.txt": "note\n", ".gitignore": ".tdd\n",
                  "TESTS.md": STAB_MD, "p_test.go": P_TESTS, "d_test.go": D_TESTS, "mise.toml": mise_toml()}, "base")
    sha = commit(repo, {"p_test.go": P_TESTS.replace("TestCALC001_A(t *testing.T) {}", "TestCALC001_A(t *testing.T) { _ = 1 }"), **CHAR_REPORTS}, "test(div): acceptance")
    make_patch(repo, mutants, "CALC-003", {"src/state": "fail\n"})
    rc, out = char(repo, sha, mutants)
    assert rc == 1 and "EARLIER ACCEPTANCE TEST MODIFIED: p_test.go" in out, out
    rc, out = char(repo, sha, mutants, "--ac-changed", "CALC-001")
    assert rc == 0 and "CRITERION CHANGE ALLOWED: p_test.go (CALC-001)" in out, out


# ------------------------------------------------------------------ --replaces
DIV_ONLY = "## div — Division\n### CALC-003 — c\n- **Critère** :\n  - **Étant donné** x\n  - **Quand** y\n  - **Alors** z\n- **Test** : `d_test.go` › `TestCALC003_C`\n"
POW_PARTIAL = ("## pow — Power\n### CALC-002 — b\n- **Critère** :\n  - **Étant donné** x\n  - **Quand** y\n  - **Alors** z\n- **Test** : `p_test.go` › `TestCALC002_B`\n\n"
               + DIV_ONLY)


def replaces_setup(repo, partial=False, keep_pow=False):
    """Base: pow (p_test.go) and div. Test commit: div acceptance that drops pow's section (and file)."""
    commit(repo, {"TESTS.md": STAB_MD, "p_test.go": P_TESTS, "d_test.go": D_TESTS, "mise.toml": mise_toml(),
                  "run_test.sh": task_script("report.xml"), "report.xml": junit(("TestCALC001_A", "pass"))}, "base")
    files = {"report.xml": junit(("TestCALC003_C", "fail"))}
    if partial:
        files["TESTS.md"] = POW_PARTIAL
        files["p_test.go"] = "package x\nfunc TestCALC002_B(t *testing.T) {}\n"
        green = junit(("TestCALC002_B", "fail"), ("TestCALC003_C", "pass"))
    else:
        files["TESTS.md"] = DIV_ONLY
        if not keep_pow:
            (repo / "p_test.go").unlink()
        green = junit(("TestCALC003_C", "pass"))
    t = commit(repo, files, "test(div): acceptance")
    f = commit(repo, {"green.xml": green, "impl.go": "package x\n"}, "feat(div)")
    return t, f


def green_args(t, f, *extra):
    return ("green", "--slug", "div", "--test", t, "--feat", f, "--task", "test", "--proof-exempt", *extra)


def test_replaced_ids():
    after = DIV_ONLY
    assert tg.replaced_ids(STAB_MD, after, ["pow"]) == {"CALC-001", "CALC-002"}
    assert tg.replaced_ids(STAB_MD, POW_PARTIAL, ["pow"]) == {"CALC-001"}
    assert tg.replaced_ids(STAB_MD, after, ["div"]) == set()


def test_replaces_allows_and_lists(repo):
    t, f = replaces_setup(repo)
    rc, out = gate(repo, *green_args(t, f, "--replaces", "pow"))
    assert rc == 0, out
    for line in ("REPLACED pow: p_test.go", "REPLACED pow: CALC-001", "REPLACED pow: CALC-002"):
        assert line in out, out


def test_replaces_unlisted_still_refused(repo):
    t, f = replaces_setup(repo)
    rc, out = gate(repo, *green_args(t, f))
    assert rc == 1 and "EARLIER ACCEPTANCE TEST MODIFIED: p_test.go" in out and "REPLACED" not in out, out


def test_replaces_unknown_slug(repo):
    t, f = replaces_setup(repo)
    rc, out = gate(repo, *green_args(t, f, "--replaces", "nope"))
    assert rc == 2 and "nope" in out and out.splitlines()[-1] == "FAIL", out


def test_replaces_partial_section(repo):
    t, f = replaces_setup(repo, partial=True)
    rc, out = gate(repo, *green_args(t, f, "--replaces", "pow"))
    assert rc == 1 and "REGRESSION CALC-002" in out, out
    assert "REPLACED pow: CALC-001" in out and "REPLACED pow: CALC-002" not in out, out


def test_characterize_replaces(repo, mutants):
    commit(repo, {"run_test.sh": CHAR_TASK, "spec.md": SPEC, "src/state": "pass\n", "src/note.txt": "note\n", ".gitignore": ".tdd\n",
                  "TESTS.md": STAB_MD, "p_test.go": P_TESTS, "d_test.go": D_TESTS, "mise.toml": mise_toml()}, "base")
    (repo / "p_test.go").unlink()
    reports = {**CHAR_REPORTS, "tests/pass.xml": junit(("TestCALC003_C", "pass")),
               "tests/fail.xml": junit(("TestCALC003_C", "fail"))}
    sha = commit(repo, {"TESTS.md": DIV_ONLY, **reports}, "test(div): acceptance")
    make_patch(repo, mutants, "CALC-003", {"src/state": "fail\n"})
    rc, out = char(repo, sha, mutants)
    assert rc == 1 and "EARLIER ACCEPTANCE TEST MODIFIED: p_test.go" in out, out
    rc, out = char(repo, sha, mutants, "--replaces", "pow")
    assert rc == 0 and "REPLACED pow: p_test.go" in out and "REPLACED pow: CALC-001" in out, out
    assert "KILLED CALC-003" in out


def orphan_char(repo, mutants, keep):
    commit(repo, {"run_test.sh": CHAR_TASK, "spec.md": SPEC, "src/state": "pass\n", "src/note.txt": "note\n", ".gitignore": ".tdd\n",
                  "TESTS.md": STAB_MD, "p_test.go": P_TESTS, "d_test.go": D_TESTS, "mise.toml": mise_toml()}, "base")
    if not keep:
        (repo / "p_test.go").unlink()
    reports = {**CHAR_REPORTS, "tests/pass.xml": junit(("TestCALC003_C", "pass")),
               "tests/fail.xml": junit(("TestCALC003_C", "fail"))}
    sha = commit(repo, {"TESTS.md": DIV_ONLY, **reports}, "test(div): acceptance")
    make_patch(repo, mutants, "CALC-003", {"src/state": "fail\n"})
    return char(repo, sha, mutants, "--replaces", "pow")


def test_orphan_test_file_after_replace(repo, mutants):
    rc, out = orphan_char(repo, mutants, keep=True)
    assert rc == 1, out
    assert "ORPHAN TEST p_test.go:2 CALC-001" in out and "ORPHAN TEST p_test.go:3 CALC-002" in out, out


def test_no_orphan_when_file_deleted(repo, mutants):
    rc, out = orphan_char(repo, mutants, keep=False)
    assert rc == 0 and "ORPHAN" not in out, out


def test_orphan_in_green(repo):
    t, f = replaces_setup(repo, keep_pow=True)
    rc, out = gate(repo, *green_args(t, f, "--replaces", "pow"))
    assert rc == 1 and "ORPHAN TEST p_test.go:2 CALC-001" in out, out


# ------------------------------------------------------------ section scope (Commande line)
def ext_md(command):
    return TESTS_MD + (
        f"\n## ext — External\nBranche : `feat/ext` · Commande : {command}\n"
        "### EXT-001 — e\n- **Critère** :\n  - **Étant donné** x\n  - **Quand** y\n  - **Alors** z\n- **Test** : `a_test.go` › `TestEXT001_E`\n"
    )


EXT_TESTS = GO_TESTS + "func TestEXT001_E(t *testing.T) {}\n"
ALL3 = (("TestCALC001_A", "pass"), ("TestCALC002_B", "pass"), ("TestCALC003_C", "pass"))


def ext_setup(repo, command, ext_status=None):
    cases = ALL3 + ((("TestEXT001_E", ext_status),) if ext_status else ())
    t = commit(repo, {"TESTS.md": ext_md(command), "a_test.go": EXT_TESTS, "mise.toml": mise_toml(),
                      "run_test.sh": task_script("report.xml"), "report.xml": junit(("TestCALC003_C", "fail"))},
               "test(div)")
    f = commit(repo, {"green.xml": junit(*cases), "impl.go": "package x\n"}, "feat(div)")
    return t, f


def test_green_out_of_scope_section(repo):
    t, f = ext_setup(repo, "pipeline MR, job `check` (`mise install`)")
    rc, out = run_green(repo, t, f)
    assert rc == 0, out
    assert out.count("OUT OF SCOPE EXT-001: pipeline MR, job `check` (`mise install`)") == 1
    assert "NO RESULT EXT-001" not in out


def test_green_out_of_scope_id_without_test(repo):
    ext = ("\n## ext — External\nBranche : `feat/ext` · Commande : pipeline MR, job `check`\n"
           "### EXT-001 — e\n- **Critère** :\n  - **Étant donné** x\n  - **Quand** y\n  - **Alors** z\n- **Test** : `.gitlab-ci.yml` › job `check`\n")
    t = commit(repo, {"TESTS.md": TESTS_MD + ext, "a_test.go": GO_TESTS, "mise.toml": mise_toml(),
                      "run_test.sh": task_script("report.xml"), "report.xml": junit(("TestCALC003_C", "fail"))},
               "test(div)")
    f = commit(repo, {"green.xml": junit(*ALL3), "impl.go": "package x\n"}, "feat(div)")
    rc, out = run_green(repo, t, f)
    assert rc == 0, out
    assert "ID WITHOUT TEST" not in out
    assert "OUT OF SCOPE EXT-001" in out


def test_green_in_scope_other_task_name(repo):
    t, f = ext_setup(repo, "`mise run test:other`")
    rc, out = run_green(repo, t, f)
    assert rc == 0 and "OUT OF SCOPE EXT-001: `mise run test:other`" in out, out
    t2, f2 = ext_setup_failing(repo)
    rc, out = run_green(repo, t2, f2)
    assert rc == 1 and "REGRESSION EXT-001" in out and "OUT OF SCOPE" not in out, out


def ext_setup_failing(repo):
    t = commit(repo, {"TESTS.md": ext_md("`mise run test`") + " ", "a_test.go": EXT_TESTS + "// 2\n"}, "test(div) 2")
    f = commit(repo, {"green.xml": junit(*ALL3, ("TestEXT001_E", "fail")), "impl2.go": "package x\n"}, "feat(div) 2")
    return t, f


def test_characterize_out_of_scope_baseline(repo, mutants):
    ext = "\n## ext — External\nBranche : `feat/ext` · Commande : pipeline MR, job `check`\n" \
          "### EXT-001 — e\n- **Critère** :\n  - **Étant donné** x\n  - **Quand** y\n  - **Alors** z\n- **Test** : `e_test.go` › `TestEXT001_E`\n"
    pow_md = STAB_MD[: STAB_MD.index("## div")]
    commit(repo, {"run_test.sh": CHAR_TASK, "spec.md": SPEC, "src/state": "pass\n", "src/note.txt": "note\n",
                  ".gitignore": ".tdd\n", "TESTS.md": pow_md + ext, "p_test.go": P_TESTS, "mise.toml": mise_toml(),
                  "e_test.go": "package x\nfunc TestEXT001_E(t *testing.T) {}\n"}, "base")
    sha = commit(repo, {"TESTS.md": STAB_MD.replace("\n## div", ext + "\n## div"), "d_test.go": D_TESTS,
                        **CHAR_REPORTS}, "test(div): acceptance")
    make_patch(repo, mutants, "CALC-003", {"src/state": "fail\n"})
    rc, out = char(repo, sha, mutants)
    assert rc == 0 and "OUT OF SCOPE EXT-001: pipeline MR, job `check`" in out, out
    assert "BASELINE NOT GREEN EXT-001" not in out and "KILLED CALC-003" in out
