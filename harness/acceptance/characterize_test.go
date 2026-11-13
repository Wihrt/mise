package acceptance

import (
	"path/filepath"
	"strings"
	"testing"
)

var (
	killing = files{"src/state": "fail\n"}
	// editedP is pTests with the test of CALC-001 edited.
	editedP = strings.Replace(pTests, "TestCALC001_A(t *testing.T) {}", "TestCALC001_A(t *testing.T) { _ = 1 }", 1)
)

func clean(t *testing.T, r *repo) {
	t.Helper()
	if st := r.status(); st != "" {
		t.Errorf("the tree is not clean:\n%s", st)
	}
}

func TestCharacterizeKilled(t *testing.T) {
	r := newRepo(t)
	sha := r.charSetup(charOpts{})
	r.makePatch("CALC-003", killing)
	code, out := r.char(sha, charRun{})
	with(t, out, "KILLED CALC-003 clause 1: TestCALC003_C")
	ends(t, code, out, 0, "PASS")
	clean(t, r)
}

func TestCharacterizeSurvived(t *testing.T) {
	r := newRepo(t)
	sha := r.charSetup(charOpts{})
	r.makePatch("CALC-003", files{"src/note.txt": "# comment\n"})
	code, out := r.char(sha, charRun{})
	with(t, out, "SURVIVED CALC-003 clause 1")
	ends(t, code, out, 1, "FAIL")
	clean(t, r)
}

func TestCharacterizeMissingPatch(t *testing.T) {
	r := newRepo(t)
	code, out := r.char(r.charSetup(charOpts{}), charRun{})
	with(t, out, "MUTANT MISSING CALC-003 clause 1")
	ends(t, code, out, 1, "FAIL")
}

func TestCharacterizePatchNotApplying(t *testing.T) {
	r := newRepo(t)
	sha := r.charSetup(charOpts{})
	write(t, filepath.Join(r.mutants, "CALC-003.patch"),
		"diff --git a/src/note.txt b/src/note.txt\n--- a/src/note.txt\n+++ b/src/note.txt\n@@ -1 +1 @@\n-not the real content\n+changed\n")
	code, out := r.char(sha, charRun{})
	with(t, out, "MUTANT INVALID CALC-003 clause 1")
	ends(t, code, out, 1, "FAIL")
	clean(t, r)
}

func TestCharacterizePatchTouchesTestPath(t *testing.T) {
	r := newRepo(t)
	sha := r.charSetup(charOpts{})
	r.makePatch("CALC-003", files{"src/state": "fail\n", "src/x_test.go": "package x\n"})
	code, out := r.char(sha, charRun{})
	with(t, out, "MUTANT INVALID CALC-003 clause 1: touches test path src/x_test.go")
	without(t, out, "KILLED")
	ends(t, code, out, 1, "FAIL")
	clean(t, r)
}

func TestCharacterizeNoResult(t *testing.T) {
	r := newRepo(t)
	sha := r.charSetup(charOpts{})
	r.makePatch("CALC-003", files{"src/state": "none\n"})
	code, out := r.char(sha, charRun{})
	with(t, out, "MUTANT INVALID CALC-003 clause 1: no result")
	ends(t, code, out, 1, "FAIL")
	clean(t, r)
}

func TestCharacterizeBaselineRed(t *testing.T) {
	r := newRepo(t)
	sha := r.charSetup(charOpts{state: "fail"})
	r.makePatch("CALC-003", files{"src/note.txt": "x\n"})
	code, out := r.char(sha, charRun{})
	with(t, out, "BASELINE NOT GREEN CALC-003: TestCALC003_C fail")
	without(t, strings.ReplaceAll(out, "MUTANT INVALID", ""), "MUTANT")
	ends(t, code, out, 1, "FAIL")
}

func TestCharacterizeRestoresTreeWithNewFile(t *testing.T) {
	r := newRepo(t)
	sha := r.charSetup(charOpts{})
	r.makePatch("CALC-003", files{"src/state": "fail\n", "src/new.txt": "new\n"})
	code, out := r.char(sha, charRun{})
	with(t, out, "KILLED CALC-003 clause 1")
	ends(t, code, out, 0, "PASS")
	clean(t, r)
	if exists(filepath.Join(r.dir, "src/new.txt")) {
		t.Error("the file the mutant created is still there")
	}
}

func TestCharacterizeTimeoutRestoresTree(t *testing.T) {
	r := newRepo(t)
	sha := r.charSetup(charOpts{})
	r.makePatch("CALC-003", files{"src/state": "slow\n", "src/new.txt": "new\n"})
	code, out := r.char(sha, charRun{}, "--timeout", "1")
	with(t, out, "MUTANT CALC-003 clause 1: patch applied", "TIMEOUT after 1s")
	ends(t, code, out, 2, "FAIL")
	clean(t, r)
	if exists(filepath.Join(r.dir, "src/new.txt")) {
		t.Error("the file the mutant created is still there")
	}
}

func TestCharacterizeDirtyTree(t *testing.T) {
	r := newRepo(t)
	sha := r.charSetup(charOpts{})
	r.makePatch("CALC-003", killing)
	r.write("junk.txt", "x\n")
	code, out := r.char(sha, charRun{})
	with(t, out, "TREE NOT CLEAN")
	without(t, out, "KILLED")
	ends(t, code, out, 1, "FAIL")
}

func TestCharacterizeHeadNotTest(t *testing.T) {
	r := newRepo(t)
	sha := r.charSetup(charOpts{})
	r.makePatch("CALC-003", killing)
	r.commit(files{"src/note.txt": "later\n"}, "feat")
	code, out := r.char(sha, charRun{})
	with(t, out, "HEAD IS NOT --test")
	without(t, out, "KILLED")
	ends(t, code, out, 1, "FAIL")
}

// fullBase is a base commit that already holds pow and div, tests included.
func (r *repo) fullBase(extra files) {
	r.t.Helper()
	r.commit(merge(files{"run_test.sh": charTask, "spec.md": spec, "src/state": "pass\n", "src/note.txt": "note\n", ".gitignore": ".tdd\n",
		"TESTS.md": stabMD, "p_test.go": pTests, "d_test.go": dTests, "mise.toml": miseToml(miseOpts{})}, extra), "base")
}

func TestCharacterizeACChangedAllowsEarlierTest(t *testing.T) {
	r := newRepo(t)
	r.fullBase(nil)
	sha := r.commit(merge(files{"p_test.go": editedP}, charReports), "test(div): acceptance")
	r.makePatch("CALC-003", killing)
	code, out := r.char(sha, charRun{})
	with(t, out, "EARLIER ACCEPTANCE TEST MODIFIED: p_test.go")
	ends(t, code, out, 1, "FAIL")
	code, out = r.char(sha, charRun{}, "--ac-changed", "CALC-001")
	with(t, out, "CRITERION CHANGE ALLOWED: p_test.go (CALC-001)")
	ends(t, code, out, 0, "PASS")
}

// divReports are the reports of an acceptance commit that keeps div only.
var divReports = merge(charReports, files{
	"tests/pass.xml": junit(tc{"TestCALC003_C", "pass"}),
	"tests/fail.xml": junit(tc{"TestCALC003_C", "fail"}),
})

// divTakesOver makes the acceptance commit of div drop pow's section; keep leaves pow's test file on disk.
func (r *repo) divTakesOver(keep bool) string {
	r.t.Helper()
	r.fullBase(nil)
	if !keep {
		r.remove("p_test.go")
	}
	sha := r.commit(merge(files{"TESTS.md": divOnly}, divReports), "test(div): acceptance")
	r.makePatch("CALC-003", killing)
	return sha
}

func TestCharacterizeReplaces(t *testing.T) {
	r := newRepo(t)
	sha := r.divTakesOver(false)
	code, out := r.char(sha, charRun{})
	with(t, out, "EARLIER ACCEPTANCE TEST MODIFIED: p_test.go")
	ends(t, code, out, 1, "FAIL")
	code, out = r.char(sha, charRun{}, "--replaces", "pow")
	with(t, out, "REPLACED pow: p_test.go", "REPLACED pow: CALC-001", "KILLED CALC-003")
	ends(t, code, out, 0, "PASS")
}

func TestOrphanTestFileAfterReplace(t *testing.T) {
	r := newRepo(t)
	code, out := r.char(r.divTakesOver(true), charRun{}, "--replaces", "pow")
	with(t, out, "ORPHAN TEST p_test.go:2 CALC-001", "ORPHAN TEST p_test.go:3 CALC-002")
	ends(t, code, out, 1, "FAIL")
}

func TestNoOrphanWhenFileDeleted(t *testing.T) {
	r := newRepo(t)
	code, out := r.char(r.divTakesOver(false), charRun{}, "--replaces", "pow")
	without(t, out, "ORPHAN")
	ends(t, code, out, 0, "PASS")
}

func TestCharacterizeOutOfScopeBaseline(t *testing.T) {
	r := newRepo(t)
	ext := "\n## ext — External\nBranche : `feat/ext` · Commande : pipeline MR, job `check`\n" + entry("EXT-001", "e", "e_test.go", "TestEXT001_E")
	powMD := stabMD[:strings.Index(stabMD, "## div")]
	r.commit(files{"run_test.sh": charTask, "spec.md": spec, "src/state": "pass\n", "src/note.txt": "note\n",
		".gitignore": ".tdd\n", "TESTS.md": powMD + ext, "p_test.go": pTests, "mise.toml": miseToml(miseOpts{}),
		"e_test.go": "package x\nfunc TestEXT001_E(t *testing.T) {}\n"}, "base")
	sha := r.commit(merge(files{"TESTS.md": strings.Replace(stabMD, "\n## div", ext+"\n## div", 1), "d_test.go": dTests}, charReports), "test(div): acceptance")
	r.makePatch("CALC-003", killing)
	code, out := r.char(sha, charRun{})
	with(t, out, "OUT OF SCOPE EXT-001: pipeline MR, job `check`", "KILLED CALC-003")
	without(t, out, "BASELINE NOT GREEN EXT-001")
	ends(t, code, out, 0, "PASS")
}
