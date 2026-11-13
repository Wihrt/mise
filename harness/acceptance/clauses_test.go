package acceptance

import (
	"strings"
	"testing"
)

// Slugs clauses (FRM-006), preuve-sensible (PRV-001, PRV-002) and
// preuve-specifique (PRV-003 to PRV-005): one mutant per clause of the Alors,
// and a neutral change.

// The spec gives CALC-003 an Alors in two clauses; the TESTS.md entry carries it on a single line.
const twoClausesSpec = "# s\n## Critères d'acceptation\n### CALC-003 — c\n- **Étant donné** x\n- **Quand** y\n" +
	"- **Alors**\n  - first clause\n  - second clause\n### CALC-001 — a\n### CALC-002 — b\n"

// oneLine is md with CALC-003's entry carrying its Alors on one line (the last "z" Alors line of md is CALC-003's).
func oneLine(md string) string {
	const alors = "  - **Alors** z\n"
	i := strings.LastIndex(md, alors)
	return md[:i] + "  - **Alors** first clause ; second clause\n" + md[i+len(alors):]
}

func TestFRM006_RedEntryWithFewerClausesThanSpecFails(t *testing.T) {
	r := newRepo(t)
	r.commit(files{"run_test.sh": taskScript("tests/report.xml", nil, ""), "spec.md": twoClausesSpec}, "base")
	sha := r.accCommit(junit(tc{"TestCALC003_C", "fail"}), files{"TESTS.md": oneLine(testsMD)})
	_, out := r.gate("red", "--slug", "div", "--test", sha, "--task", "test", "--spec", "spec.md", "--ac", "CALC-003")
	with(t, out, "CLAUSES CALC-003: 1 in TESTS.md, 2 in spec")
	lastIs(t, out, "FAIL")
}

func TestFRM006_CharacterizeEntryWithFewerClausesThanSpecFails(t *testing.T) {
	r := newRepo(t)
	sha := r.charSetup(charOpts{md: oneLine(stabMD), files: files{"spec.md": twoClausesSpec}})
	_, out := r.char(sha, charRun{})
	with(t, out, "CLAUSES CALC-003: 1 in TESTS.md, 2 in spec")
	lastIs(t, out, "FAIL")
}

// twoClauseMD gives CALC-003 an Alors in two clauses (a line followed by a list).
var twoClauseMD = strings.Replace(stabMD, "  - **Alors** z\n- **Test** : `d_test.go`",
	"  - **Alors**\n    - clause une\n    - clause deux\n- **Test** : `d_test.go`", 1)

var (
	// harmless makes no test fail.
	harmless = files{"src/note.txt": "neutral\n"}
	comment  = files{"src/note.txt": "# comment\n"}
)

func TestPRV001_MissingMutantForClause2(t *testing.T) {
	r := newRepo(t)
	sha := r.charSetup(charOpts{md: twoClauseMD})
	r.makePatch("CALC-003", killing)
	r.makePatch("CALC-003.neutral", harmless)
	_, out := r.char(sha, charRun{})
	with(t, out, "MUTANT MISSING CALC-003 clause 2")
	lastIs(t, out, "FAIL")
}

func TestPRV002_ClauseWithoutDefendingTestSurvives(t *testing.T) {
	r := newRepo(t)
	sha := r.charSetup(charOpts{md: twoClauseMD})
	r.makePatch("CALC-003", killing)
	r.makePatch("CALC-003.2", comment)
	r.makePatch("CALC-003.neutral", harmless)
	_, out := r.char(sha, charRun{})
	with(t, out, "KILLED CALC-003 clause 1", "SURVIVED CALC-003 clause 2")
	lastIs(t, out, "FAIL")
}

func TestPRV003_MissingNeutralChange(t *testing.T) {
	r := newRepo(t)
	sha := r.charSetup(charOpts{})
	r.makePatch("CALC-003", killing)
	_, out := r.char(sha, charRun{neutral: neutral{none: true}})
	with(t, out, "NEUTRAL MISSING CALC-003")
	lastIs(t, out, "FAIL")
}

func TestPRV003_NeutralChangeNotApplyingIsInvalid(t *testing.T) {
	r := newRepo(t)
	sha := r.charSetup(charOpts{})
	r.makePatch("CALC-003", killing)
	write(t, r.mutants+"/CALC-003.neutral.patch",
		"diff --git a/src/absent.txt b/src/absent.txt\n--- a/src/absent.txt\n+++ b/src/absent.txt\n@@ -1 +1 @@\n-x\n+y\n")
	_, out := r.char(sha, charRun{})
	with(t, out, "NEUTRAL INVALID CALC-003")
	without(t, out, "NEUTRAL OK CALC-003")
	lastIs(t, out, "FAIL")
}

// The neutral change renames an element the test asserts: the test of CALC-003 fails under it.
func TestPRV004_TestBreakingUnderNeutralChangeIsBrittle(t *testing.T) {
	r := newRepo(t)
	sha := r.charSetup(charOpts{})
	r.makePatch("CALC-003", killing)
	_, out := r.char(sha, charRun{neutral: neutral{change: killing}})
	brittle := grep(out, "BRITTLE CALC-003")
	if len(brittle) == 0 || !strings.Contains(brittle[0], "TestCALC003_C") {
		t.Errorf("no BRITTLE line naming the test:\n%s", out)
	}
	lastIs(t, out, "FAIL")
}

func TestPRV005_SensitiveAndSpecificTestPasses(t *testing.T) {
	r := newRepo(t)
	sha := r.charSetup(charOpts{md: twoClauseMD})
	r.makePatch("CALC-003", killing)
	r.makePatch("CALC-003.2", killing)
	_, out := r.char(sha, charRun{neutral: neutral{change: harmless}})
	without(t, out, "MUTANT MISSING")
	with(t, out, "KILLED CALC-003 clause 1", "KILLED CALC-003 clause 2", "NEUTRAL OK CALC-003")
	lastIs(t, out, "PASS")
}

// Slug neutre-sans-resultat: the neutral change applies but sets a state for which the task writes no report.
func TestPRV010_NeutralChangeWithoutResultIsInvalid(t *testing.T) {
	r := newRepo(t)
	sha := r.charSetup(charOpts{})
	r.makePatch("CALC-003", killing)
	code, out := r.char(sha, charRun{neutral: neutral{change: files{"src/state": "none\n"}}})
	with(t, out, "NEUTRAL INVALID CALC-003")
	without(t, out, "NEUTRAL OK CALC-003")
	ends(t, code, out, 1, "FAIL")
}

// Slug revue-finale: a mutant whose patch starts at `--- a/<file>`.
func TestPRV011_MutantWithoutDiffGitHeaderIsJudged(t *testing.T) {
	r := newRepo(t)
	sha := r.charSetup(charOpts{})
	r.patch("CALC-003", killing, true)
	text := read(t, r.mutants+"/CALC-003.patch")
	if !strings.HasPrefix(text, "--- a/src/state\n") || strings.Contains(text, "diff --git") || strings.Contains(text, "\nindex ") {
		t.Fatalf("the patch still has its git header:\n%s", text)
	}
	_, out := r.char(sha, charRun{})
	with(t, out, "KILLED CALC-003")
}
