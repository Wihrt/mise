package acceptance

import (
	"strings"
	"testing"
)

func TestGreenPass(t *testing.T) {
	r := newRepo(t)
	test, feat := r.greenSetup(greenOpts{})
	code, out := r.runGreen(test, feat, greenRun{})
	with(t, out, "GREEN CALC-003")
	ends(t, code, out, 0, "PASS")
}

func TestGreenToolsOnlyMiseChangeOK(t *testing.T) {
	r := newRepo(t)
	test, feat := r.greenSetup(greenOpts{featMise: miseToml(miseOpts{tools: "foo = \"1\"\n"})})
	code, out := r.runGreen(test, feat, greenRun{})
	ends(t, code, out, 0, "PASS")
}

func TestGreenMiseTasksChanged(t *testing.T) {
	r := newRepo(t)
	test, feat := r.greenSetup(greenOpts{featMise: strings.Replace(miseToml(miseOpts{}), "exit 0", "true", 1)})
	code, out := r.runGreen(test, feat, greenRun{})
	with(t, out, "PROTECTED FILE MODIFIED: mise.toml (tasks)")
	ends(t, code, out, 1, "FAIL")
}

func TestGreenProtectedTestFileModified(t *testing.T) {
	r := newRepo(t)
	test, feat := r.greenSetup(greenOpts{featFiles: files{"a_test.go": goTests + "// reformat\n"}})
	code, out := r.runGreen(test, feat, greenRun{})
	with(t, out, "PROTECTED FILE MODIFIED: a_test.go")
	ends(t, code, out, 1, "FAIL")
}

func TestGreenTestsMDModified(t *testing.T) {
	r := newRepo(t)
	test, feat := r.greenSetup(greenOpts{featFiles: files{"TESTS.md": testsMD + "\n"}})
	code, out := r.runGreen(test, feat, greenRun{})
	with(t, out, "PROTECTED FILE MODIFIED: TESTS.md")
	ends(t, code, out, 1, "FAIL")
}

func TestGreenFailingTestcase(t *testing.T) {
	r := newRepo(t)
	test, feat := r.greenSetup(greenOpts{featReport: junit(tc{"TestCALC003_C", "fail"})})
	code, out := r.runGreen(test, feat, greenRun{})
	with(t, out, "NOT GREEN CALC-003")
	ends(t, code, out, 1, "FAIL")
}

func TestGreenSkippedTestcaseIsNotGreen(t *testing.T) {
	r := newRepo(t)
	test, feat := r.greenSetup(greenOpts{featReport: "<testsuites><testsuite><testcase name='TestCALC003_C'><skipped/></testcase></testsuite></testsuites>"})
	code, out := r.runGreen(test, feat, greenRun{})
	with(t, out, "NOT GREEN CALC-003")
	ends(t, code, out, 1, "FAIL")
}

func TestGreenRegressionOfEarlierSlug(t *testing.T) {
	r := newRepo(t)
	test, feat := r.greenSetup(greenOpts{featReport: junit(tc{"TestCALC001_A", "fail"}, tc{"TestCALC002_B", "pass"}, tc{"TestCALC003_C", "pass"})})
	code, out := r.runGreen(test, feat, greenRun{})
	with(t, out, "REGRESSION CALC-001", "GREEN CALC-003")
	ends(t, code, out, 1, "FAIL")
}

func TestGreenEarlierIDWithoutResult(t *testing.T) {
	r := newRepo(t)
	test, feat := r.greenSetup(greenOpts{featReport: junit(tc{"TestCALC001_A", "pass"}, tc{"TestCALC003_C", "pass"})})
	code, out := r.runGreen(test, feat, greenRun{})
	with(t, out, "NO RESULT CALC-002", "GREEN CALC-003")
	ends(t, code, out, 1, "FAIL")
}

func TestGreenCheckFails(t *testing.T) {
	r := newRepo(t)
	test, feat := r.greenSetup(greenOpts{checkRC: 3})
	code, out := r.runGreen(test, feat, greenRun{})
	// mise.toml [tasks.check] changed too, so both signals appear
	with(t, out, "CHECK FAILED")
	ends(t, code, out, 1, "FAIL")
}

func TestGreenIDWithoutTestAndTestWithoutID(t *testing.T) {
	r := newRepo(t)
	test := r.baseCommit(junit(tc{"TestCALC003_C", "fail"}), baseOpts{md: testsMD + entry("CALC-008", "ghost", "a_test.go", "TestCALC008_G")})
	code, out := r.runGreen(test, test, greenRun{})
	with(t, out, "ID WITHOUT TEST: CALC008")
	ends(t, code, out, 1, "FAIL")
	// a real test with an ID that TESTS.md does not list
	test = r.commit(files{"TESTS.md": testsMD, "a_test.go": goTests + "func TestCALC042_Z(t *testing.T) {}\n"}, "x")
	code, out = r.runGreen(test, test, greenRun{})
	with(t, out, "TEST WITHOUT ID: CALC042")
	without(t, out, "SHA256")
	ends(t, code, out, 1, "FAIL")
}

func TestGreenIncompleteSection(t *testing.T) {
	r := newRepo(t)
	test := r.baseCommit(junit(tc{"TestCALC003_C", "pass"}), baseOpts{md: testsMD + "### CALC-010 — half\n- **Spec** : x\n", rc: intp(0)})
	code, out := r.runGreen(test, test, greenRun{})
	with(t, out, "INCOMPLETE: ### CALC-010 — half")
	ends(t, code, out, 1, "FAIL")
}

// ------------------------------------------------------------ stability check

func TestGreenStabilityBlocksEarlierTest(t *testing.T) {
	r := newRepo(t)
	test, feat := r.stabilitySetup(true, nil)
	code, out := r.runGreen(test, feat, greenRun{})
	with(t, out, "EARLIER ACCEPTANCE TEST MODIFIED: p_test.go")
	ends(t, code, out, 1, "FAIL")
}

func TestGreenStabilityIncludesAcceptanceCommit(t *testing.T) {
	r := newRepo(t)
	test, feat := r.stabilitySetup(true, nil)
	code, out := r.gate("green", "--slug", "div", "--test", test, "--feat", feat, "--task", "test", "--ac-changed", "CALC-001", "--proof-exempt")
	with(t, out, "CRITERION CHANGE ALLOWED: p_test.go (CALC-001)")
	ends(t, code, out, 0, "PASS")
}

func TestGreenStabilityFeatEdit(t *testing.T) {
	r := newRepo(t)
	test, feat := r.stabilitySetup(false, files{"p_test.go": pTests + "// x\n"})
	code, out := r.runGreen(test, feat, greenRun{})
	with(t, out, "EARLIER ACCEPTANCE TEST MODIFIED: p_test.go")
	ends(t, code, out, 1, "FAIL")
}

func TestGreenUnitTestWithoutIDIgnored(t *testing.T) {
	r := newRepo(t)
	test, feat := r.stabilitySetup(false, files{"helper_test.go": "package x\nfunc TestParseHelper(t *testing.T) {}\n"})
	code, out := r.runGreen(test, feat, greenRun{})
	ends(t, code, out, 0, "PASS")
}

// ------------------------------------------------------------------ --replaces

// replacesSetup makes a base with pow (p_test.go) and div, then a div
// acceptance commit that drops pow's section (and file), then a feat commit.
func (r *repo) replacesSetup(partial, keepPow bool) (string, string) {
	r.t.Helper()
	r.commit(files{"TESTS.md": stabMD, "p_test.go": pTests, "d_test.go": dTests, "mise.toml": miseToml(miseOpts{}),
		"run_test.sh": taskScript("report.xml", nil, ""), "report.xml": junit(tc{"TestCALC001_A", "pass"})}, "base")
	f := files{"report.xml": junit(tc{"TestCALC003_C", "fail"})}
	green := junit(tc{"TestCALC003_C", "pass"})
	if partial {
		f["TESTS.md"] = powPartial
		f["p_test.go"] = "package x\nfunc TestCALC002_B(t *testing.T) {}\n"
		green = junit(tc{"TestCALC002_B", "fail"}, tc{"TestCALC003_C", "pass"})
	} else {
		f["TESTS.md"] = divOnly
		if !keepPow {
			r.remove("p_test.go")
		}
	}
	test := r.commit(f, "test(div): acceptance")
	return test, r.commit(files{"green.xml": green, "impl.go": "package x\n"}, "feat(div)")
}

func greenArgs(test, feat string, extra ...string) []string {
	return append([]string{"green", "--slug", "div", "--test", test, "--feat", feat, "--task", "test", "--proof-exempt"}, extra...)
}

func TestReplacesAllowsAndLists(t *testing.T) {
	r := newRepo(t)
	test, feat := r.replacesSetup(false, false)
	code, out := r.gate(greenArgs(test, feat, "--replaces", "pow")...)
	with(t, out, "REPLACED pow: p_test.go", "REPLACED pow: CALC-001", "REPLACED pow: CALC-002")
	ends(t, code, out, 0, "PASS")
}

func TestReplacesUnlistedStillRefused(t *testing.T) {
	r := newRepo(t)
	test, feat := r.replacesSetup(false, false)
	code, out := r.gate(greenArgs(test, feat)...)
	with(t, out, "EARLIER ACCEPTANCE TEST MODIFIED: p_test.go")
	without(t, out, "REPLACED")
	ends(t, code, out, 1, "FAIL")
}

func TestReplacesUnknownSlug(t *testing.T) {
	r := newRepo(t)
	test, feat := r.replacesSetup(false, false)
	code, out := r.gate(greenArgs(test, feat, "--replaces", "nope")...)
	with(t, out, "nope")
	ends(t, code, out, 2, "FAIL")
}

func TestReplacesPartialSection(t *testing.T) {
	r := newRepo(t)
	test, feat := r.replacesSetup(true, false)
	code, out := r.gate(greenArgs(test, feat, "--replaces", "pow")...)
	with(t, out, "REGRESSION CALC-002", "REPLACED pow: CALC-001")
	without(t, out, "REPLACED pow: CALC-002")
	ends(t, code, out, 1, "FAIL")
}

func TestOrphanInGreen(t *testing.T) {
	r := newRepo(t)
	test, feat := r.replacesSetup(false, true)
	code, out := r.gate(greenArgs(test, feat, "--replaces", "pow")...)
	with(t, out, "ORPHAN TEST p_test.go:2 CALC-001")
	ends(t, code, out, 1, "FAIL")
}

// ------------------------------------------------------------ section scope (Commande line)

func extMD(command string) string {
	return testsMD + "\n## ext — External\nBranche : `feat/ext` · Commande : " + command + "\n" + entry("EXT-001", "e", "a_test.go", "TestEXT001_E")
}

var extTests = goTests + "func TestEXT001_E(t *testing.T) {}\n"

func (r *repo) extSetup(command string) (string, string) {
	r.t.Helper()
	test := r.commit(files{"TESTS.md": extMD(command), "a_test.go": extTests, "mise.toml": miseToml(miseOpts{}),
		"run_test.sh": taskScript("report.xml", nil, ""), "report.xml": junit(tc{"TestCALC003_C", "fail"})}, "test(div)")
	return test, r.commit(files{"green.xml": greenAll, "impl.go": "package x\n"}, "feat(div)")
}

func TestGreenOutOfScopeSection(t *testing.T) {
	r := newRepo(t)
	test, feat := r.extSetup("pipeline MR, job `check` (`mise install`)")
	code, out := r.runGreen(test, feat, greenRun{})
	if n := strings.Count(out, "OUT OF SCOPE EXT-001: pipeline MR, job `check` (`mise install`)"); n != 1 {
		t.Errorf("OUT OF SCOPE line printed %d times, want 1:\n%s", n, out)
	}
	without(t, out, "NO RESULT EXT-001")
	ends(t, code, out, 0, "PASS")
}

func TestGreenOutOfScopeIDWithoutTest(t *testing.T) {
	r := newRepo(t)
	ext := "\n## ext — External\nBranche : `feat/ext` · Commande : pipeline MR, job `check`\n" +
		"### EXT-001 — e\n- **Critère** :\n  - **Étant donné** x\n  - **Quand** y\n  - **Alors** z\n- **Test** : `.gitlab-ci.yml` › job `check`\n"
	test := r.commit(files{"TESTS.md": testsMD + ext, "a_test.go": goTests, "mise.toml": miseToml(miseOpts{}),
		"run_test.sh": taskScript("report.xml", nil, ""), "report.xml": junit(tc{"TestCALC003_C", "fail"})}, "test(div)")
	feat := r.commit(files{"green.xml": greenAll, "impl.go": "package x\n"}, "feat(div)")
	code, out := r.runGreen(test, feat, greenRun{})
	with(t, out, "OUT OF SCOPE EXT-001")
	without(t, out, "ID WITHOUT TEST")
	ends(t, code, out, 0, "PASS")
}

func TestGreenInScopeOtherTaskName(t *testing.T) {
	r := newRepo(t)
	test, feat := r.extSetup("`mise run test:other`")
	code, out := r.runGreen(test, feat, greenRun{})
	with(t, out, "OUT OF SCOPE EXT-001: `mise run test:other`")
	ends(t, code, out, 0, "PASS")
	test = r.commit(files{"TESTS.md": extMD("`mise run test`") + " ", "a_test.go": extTests + "// 2\n"}, "test(div) 2")
	feat = r.commit(files{"green.xml": junit(append(append([]tc{}, all3...), tc{"TestEXT001_E", "fail"})...), "impl2.go": "package x\n"}, "feat(div) 2")
	code, out = r.runGreen(test, feat, greenRun{})
	with(t, out, "REGRESSION EXT-001")
	without(t, out, "OUT OF SCOPE")
	ends(t, code, out, 1, "FAIL")
}
