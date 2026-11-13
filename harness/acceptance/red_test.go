package acceptance

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestRedPassAllFail(t *testing.T) {
	r := newRepo(t)
	sha := r.baseCommit(junit(tc{"TestCALC001_A", "pass"}, tc{"TestCALC002_B", "fail"}, tc{"TestCALC003_C", "fail"}), baseOpts{})
	// section pow = CALC-001, CALC-002 -> CALC001 passes: not red
	code, out := r.gate("red", "--slug", "pow", "--test", sha, "--task", "test")
	with(t, out, "NOT RED CALC-001")
	ends(t, code, out, 1, "FAIL")
	// section div = CALC-003 -> red; the other sections' state is ignored
	code, out = r.gate("red", "--slug", "div", "--test", sha, "--task", "test")
	with(t, out, "RED CALC-003")
	ends(t, code, out, 0, "PASS")
}

func TestRedMissingTest(t *testing.T) {
	r := newRepo(t)
	md := testsMD + entry("CALC-009", "z", "a_test.go", "TestCALC009_Z") // in section div
	sha := r.baseCommit(junit(tc{"TestCALC003_C", "fail"}), baseOpts{md: md})
	code, out := r.gate("red", "--slug", "div", "--test", sha, "--task", "test")
	with(t, out, "MISSING CALC-009")
	ends(t, code, out, 1, "FAIL")
}

func TestRedTaskPassed(t *testing.T) {
	r := newRepo(t)
	sha := r.baseCommit(junit(tc{"TestCALC003_C", "pass"}), baseOpts{rc: intp(0)})
	code, out := r.gate("red", "--slug", "div", "--test", sha, "--task", "test")
	with(t, out, "TASK PASSED, NOT RED", "NOT RED CALC-003")
	ends(t, code, out, 1, "FAIL")
}

func TestRedCompileErrorExcusedWhenSymbolUsed(t *testing.T) {
	r := newRepo(t)
	sha := r.baseCommit(junit(tc{"TestMain", "fail"}), baseOpts{extra: "echo 'x_test.go:9:14: undefined: Div'"})
	code, out := r.gate("red", "--slug", "pow", "--test", sha, "--task", "test")
	// CALC-001 uses A(), not Div: not excused. CALC-002 uses Div: excused.
	with(t, out, "COMPILE OR COLLECT ERROR: x_test.go:9:14: undefined: Div", "RED CALC-002", "NOT RED CALC-001")
	ends(t, code, out, 1, "FAIL")
}

func TestRedCompileErrorAllIDsUseSymbol(t *testing.T) {
	r := newRepo(t)
	sha := r.commit(files{"TESTS.md": testsMD, "a_test.go": strings.Replace(goTests, "A()", "Div()", 1), "mise.toml": miseToml(miseOpts{}),
		"run_test.sh": "echo 'undefined: Div'\nexit 1\n"}, "test")
	code, out := r.gate("red", "--slug", "pow", "--test", sha, "--task", "test")
	with(t, out, "excused by compile/collect error")
	ends(t, code, out, 0, "PASS")
}

func noReport(r *repo) string {
	return r.commit(files{"TESTS.md": testsMD, "a_test.go": goTests, "mise.toml": miseToml(miseOpts{}), "run_test.sh": "exit 1\n"}, "test")
}

func TestRedMissingReportWithoutSymbolFails(t *testing.T) {
	r := newRepo(t)
	code, out := r.gate("red", "--slug", "div", "--test", noReport(r), "--task", "test")
	with(t, out, "REPORT", "missing", "NOT RED CALC-003")
	ends(t, code, out, 1, "FAIL")
}

func TestRedStaleReportIsNotTrusted(t *testing.T) {
	r := newRepo(t)
	sha := noReport(r)
	r.write(".tdd/junit.xml", junit(tc{"TestCALC003_C", "fail"}))
	code, out := r.gate("red", "--slug", "div", "--test", sha, "--task", "test")
	ends(t, code, out, 1, "FAIL")
}

func TestNoIDsAndUsageErrors(t *testing.T) {
	r := newRepo(t)
	sha := r.baseCommit(junit(), baseOpts{})
	code, out := r.gate("red", "--slug", "ghost", "--test", sha, "--task", "test")
	with(t, out, "NO IDS FOUND")
	ends(t, code, out, 2, "FAIL")
	if code, _ := run(t, r.dir, nil, harness, "gate", "green", "--slug", "x"); code != 2 {
		t.Errorf("missing options: exit %d, want 2", code)
	}
}

func TestTofuJSONFormat(t *testing.T) {
	r := newRepo(t)
	ev, err := json.Marshal(map[string]any{"type": "test_run", "test_run": map[string]string{"run": "CALC003_c", "status": "fail"}})
	if err != nil {
		t.Fatal(err)
	}
	sha := r.commit(files{
		"TESTS.md":     strings.ReplaceAll(testsMD, "a_test.go", "m.tftest.hcl"),
		"m.tftest.hcl": "run \"CALC001_a\" {\n}\nrun \"CALC002_b\" {\n}\nrun \"CALC003_c\" {\n}\n",
		"mise.toml":    miseToml(miseOpts{}),
		"run_test.sh":  "mkdir -p .tdd\ncat > .tdd/tofu.json <<'E'\n" + string(ev) + "\nE\nexit 1\n",
	}, "test")
	code, out := r.gate("red", "--slug", "div", "--test", sha, "--task", "test", "--format", "tofu-json")
	with(t, out, "RED CALC-003")
	ends(t, code, out, 0, "PASS")
	code, out = r.gate("red", "--slug", "div", "--test", sha, "--task", "test")
	with(t, out, "tofu.json")
	ends(t, code, out, 0, "PASS")
}

// ------------------------------------------------------------ acceptance mode

func TestRedAcceptancePass(t *testing.T) {
	r := newRepo(t)
	sha := r.accCommit(junit(tc{"TestCALC003_C", "fail"}), nil)
	code, out := r.gate("red", "--slug", "div", "--test", sha, "--task", "test", "--spec", "spec.md", "--ac", "CALC-003")
	with(t, out, "CRITERIA: CALC-003", "RED CALC-003")
	ends(t, code, out, 0, "PASS")
}

func TestRedACNotInSpec(t *testing.T) {
	r := newRepo(t)
	sha := r.accCommit(junit(tc{"TestCALC003_C", "fail"}), nil)
	code, out := r.gate("red", "--slug", "div", "--test", sha, "--task", "test", "--spec", "spec.md", "--ac", "CALC-003,CALC-004")
	with(t, out, "NOT IN SPEC CALC-004")
	ends(t, code, out, 1, "FAIL")
}

func TestRedACSectionMismatch(t *testing.T) {
	r := newRepo(t)
	sha := r.accCommit(junit(tc{"TestCALC001_A", "fail"}, tc{"TestCALC002_B", "fail"}, tc{"TestCALC003_C", "fail"}), nil)
	code, out := r.gate("red", "--slug", "pow", "--test", sha, "--task", "test", "--spec", "spec.md", "--ac", "CALC-001")
	with(t, out, "NOT IN --ac CALC-002")
	ends(t, code, out, 1, "FAIL")
	code, out = r.gate("red", "--slug", "div", "--test", sha, "--task", "test", "--spec", "spec.md", "--ac", "CALC-003,CALC-001")
	with(t, out, "NOT IN SECTION CALC-001")
	ends(t, code, out, 1, "FAIL")
}

func TestRedAcceptanceRefusesCompileExcuse(t *testing.T) {
	r := newRepo(t)
	r.commit(files{"run_test.sh": "echo 'undefined: Div'\nexit 1\n", "spec.md": spec}, "base")
	sha := r.commit(files{"TESTS.md": testsMD, "a_test.go": strings.Replace(goTests, "A()", "Div()", 1), "mise.toml": miseToml(miseOpts{})}, "test")
	args := []string{"red", "--slug", "pow", "--test", sha, "--task", "test", "--spec", "spec.md", "--ac", "CALC-001,CALC-002"}
	code, out := r.gate(args...)
	with(t, out, "NOT RED CALC-001: compile/collect error excuse refused in acceptance mode")
	ends(t, code, out, 1, "FAIL")
	code, out = r.gate(append(args, "--allow-compile-red")...)
	with(t, out, "excused by compile/collect error")
	ends(t, code, out, 0, "PASS")
}

func TestRedAcceptanceRejectsProductionFile(t *testing.T) {
	r := newRepo(t)
	args := func(sha string) []string {
		return []string{"red", "--slug", "div", "--test", sha, "--task", "test", "--spec", "spec.md", "--ac", "CALC-003"}
	}
	sha := r.accCommit(junit(tc{"TestCALC003_C", "fail"}), files{"pkg/calc.go": "package x\n"})
	code, out := r.gate(args(sha)...)
	with(t, out, "NON-TEST FILE IN ACCEPTANCE COMMIT: pkg/calc.go")
	ends(t, code, out, 1, "FAIL")
	ok := r.accCommit(junit(tc{"TestCALC003_C", "fail"}), files{
		"tests/x_test.go": "package x\n", "chart/tests/acceptance/a.yaml": "a: 1\n",
		"cmd/testdata/p.json": "{}\n", ".gitignore": ".tdd\n", "conftest.py": "\n",
		"m.tftest.hcl": "\n", "test_a.py": "\n"})
	code, out = r.gate(args(ok)...)
	without(t, out, "NON-TEST FILE")
	ends(t, code, out, 0, "PASS")
}

func TestRedSpecACUsage(t *testing.T) {
	r := newRepo(t)
	sha := r.accCommit(junit(tc{"TestCALC003_C", "fail"}), nil)
	base := []string{"red", "--slug", "div", "--test", sha, "--task", "test"}
	if code, _ := r.gate(append(base, "--ac", "CALC-003")...); code != 2 {
		t.Errorf("--ac without --spec: exit %d, want 2", code)
	}
	if code, _ := r.gate(append(base, "--spec", "spec.md")...); code != 2 {
		t.Errorf("--spec without --ac: exit %d, want 2", code)
	}
}

func TestRedSpecMissingFile(t *testing.T) {
	r := newRepo(t)
	sha := r.accCommit(junit(tc{"TestCALC003_C", "fail"}), nil)
	code, out := r.gate("red", "--slug", "div", "--test", sha, "--task", "test", "--spec", "nope.md", "--ac", "CALC-003")
	with(t, out, "not found")
	ends(t, code, out, 2, "FAIL")
}
