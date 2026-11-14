package acceptance

import (
	"strings"
	"testing"
)

// Slugs preuves-nouveau (PRV-007), revue-finale (PRV-012),
// preuves-optionnelles (PRV-017) and sans-code-minimal (PRV-020): the proofs
// of the green gate.

// newFeature makes a red acceptance commit (CALC-003 fails), then a feat commit making it green.
func newFeature(r *repo) (string, string) {
	r.t.Helper()
	test := r.charSetup(charOpts{state: "fail"})
	return test, r.commit(files{"src/state": "pass\n"}, "feat(div)")
}

func TestPRV007_MissingProofsOptionFails(t *testing.T) {
	r := newRepo(t)
	test, feat := newFeature(r)
	code, out := r.runGreen(test, feat, greenRun{noExempt: true})
	with(t, out, "PROOF MISSING")
	ends(t, code, out, 1, "FAIL")
}

func TestPRV007_SurvivingMutantFails(t *testing.T) {
	r := newRepo(t)
	test, feat := newFeature(r)
	r.makePatch("CALC-003", comment)
	code, out := r.runGreen(test, feat, greenRun{noExempt: true, proofs: true})
	with(t, out, "SURVIVED")
	ends(t, code, out, 1, "FAIL")
}

func TestPRV007_KilledMutantAndNeutralChangePass(t *testing.T) {
	r := newRepo(t)
	test, feat := newFeature(r)
	r.makePatch("CALC-003", killing)
	_, out := r.runGreen(test, feat, greenRun{noExempt: true, proofs: true})
	with(t, out, "KILLED CALC-003", "NEUTRAL OK CALC-003")
	without(t, out, "SURVIVED")
	lastIs(t, out, "PASS")
}

func TestPRV007_ProofExemptAppliesNoMutant(t *testing.T) {
	r := newRepo(t)
	test, feat := newFeature(r)
	code, out := r.runGreen(test, feat, greenRun{})
	without(t, out, "KILLED", "SURVIVED")
	ends(t, code, out, 0, "PASS")
}

const (
	prod = "src/calc.txt"
	// neededTask passes only when the production file holds a line "needed".
	neededTask = "mkdir -p .tdd\n" +
		"if grep -q '^needed$' src/calc.txt; then cp tests/pass.xml .tdd/junit.xml; exit 0; fi\n" +
		"cp tests/fail.xml .tdd/junit.xml\nexit 1\n"
	prodBase = "header\nmiddle\n"
)

func TestPRV012_ProofsNotAppliedOnModifiedTree(t *testing.T) {
	r := newRepo(t)
	test, feat := newFeature(r)
	r.makePatch("CALC-003", killing)
	r.write("src/note.txt", "uncommitted edit\n")
	_, out := r.runGreen(test, feat, greenRun{noExempt: true, proofs: true})
	with(t, out, "TREE NOT CLEAN")
	without(t, out, "MUTANT")
	lastIs(t, out, "FAIL")
	if got := read(t, r.dir+"/src/note.txt"); got != "uncommitted edit\n" {
		t.Errorf("the uncommitted edit is gone: %q", got)
	}
}

// feature makes an acceptance commit, a feat commit writing content in the
// production file, and a mutant killing CALC-003.
func feature(r *repo, content string) (string, string) {
	r.t.Helper()
	test := r.charSetup(charOpts{files: files{prod: prodBase}, task: neededTask})
	feat := r.commit(files{prod: content}, "feat(div)")
	r.makePatch("CALC-003", files{prod: strings.Replace(content, "needed\n", "", 1)})
	return test, feat
}

func TestPRV017_NeutralChangeIsSkippedOnRequest(t *testing.T) {
	r := newRepo(t)
	test, feat := feature(r, "header\nneeded\nmiddle\n")
	code, out := r.runGreen(test, feat, greenRun{noExempt: true, proofs: true, neutral: neutral{none: true}, options: []string{"--no-neutral"}})
	with(t, out, "KILLED CALC-003", "NEUTRAL SKIPPED")
	without(t, out, "NEUTRAL MISSING")
	ends(t, code, out, 0, "PASS")
	code, out = r.runGreen(test, feat, greenRun{noExempt: true, proofs: true, neutral: neutral{none: true}})
	with(t, out, "NEUTRAL MISSING CALC-003")
	ends(t, code, out, 1, "FAIL")
}

// Line 4 ("extra") is a block no test needs: it is no longer judged.
func TestPRV020_UnneededBlockIsNoLongerJudged(t *testing.T) {
	r := newRepo(t)
	test, feat := feature(r, "header\nneeded\nmiddle\nextra\n")
	code, out := r.runGreen(test, feat, greenRun{noExempt: true, proofs: true, neutral: neutral{none: true}, options: []string{"--no-neutral"}})
	with(t, out, "KILLED CALC-003")
	without(t, out, "UNNEEDED", "MINIMAL")
	ends(t, code, out, 0, "PASS")
}

func TestPRV020_RetiredOptionsAreRefused(t *testing.T) {
	for _, option := range [][]string{{"--no-minimal"}, {"--since", "HEAD"}, {"--unneeded-ok", prod + ":4-4"}, {"--spec", "spec.md"}} {
		r := newRepo(t)
		test, feat := feature(r, "header\nneeded\nmiddle\nextra\n")
		code, stderr := r.refusedGreen(test, feat, greenRun{noExempt: true, proofs: true, neutral: neutral{none: true}, options: append([]string{"--no-neutral"}, option...)})
		if code != 2 {
			t.Errorf("%s: exit code = %d, want 2\n%s", option[0], code, stderr)
		}
		if named := "flag provided but not defined: " + option[0][1:]; !strings.Contains(stderr, named) {
			t.Errorf("%s: stderr does not hold %q\n%s", option[0], named, stderr)
		}
	}
}
