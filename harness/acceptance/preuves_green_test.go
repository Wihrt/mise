package acceptance

import (
	"strings"
	"testing"
)

// Slugs preuves-nouveau (PRV-007), code-minimal (PRV-009), revue-finale
// (PRV-012, PRV-013), spec-hors-code (PRV-014) and preuves-optionnelles
// (PRV-017, PRV-018): the proofs of the green gate.

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

// twoBlocks adds two separate blocks: line 2 ("needed") and line 4 ("extra"), line 3 unchanged between them.
func twoBlocks(r *repo) (string, string) {
	r.t.Helper()
	test := r.charSetup(charOpts{files: files{prod: prodBase}, task: neededTask})
	feat := r.commit(files{prod: "header\nneeded\nmiddle\nextra\n"}, "feat(div)")
	r.makePatch("CALC-003", files{prod: "header\nmiddle\nextra\n"})
	return test, feat
}

func TestPRV009_UnneededBlockIsRefused(t *testing.T) {
	r := newRepo(t)
	test, feat := twoBlocks(r)
	_, out := r.runGreen(test, feat, greenRun{noExempt: true, proofs: true})
	unneeded := grep(out, "UNNEEDED")
	if len(unneeded) == 0 || !strings.HasPrefix(unneeded[0], "UNNEEDED "+prod+":4-4") {
		t.Errorf("the unneeded block is not refused:\n%s", out)
	}
	without(t, strings.Join(unneeded, "\n"), prod+":2-2")
	lastIs(t, out, "FAIL")
}

func TestPRV009_UnneededOKAcceptsTheBlock(t *testing.T) {
	r := newRepo(t)
	test, feat := twoBlocks(r)
	_, out := r.runGreen(test, feat, greenRun{noExempt: true, proofs: true, unneededOK: []string{prod + ":4-4"}})
	unneeded := grep(out, "UNNEEDED")
	if len(unneeded) != 1 || unneeded[0] != "UNNEEDED ACCEPTED "+prod+":4-4" {
		t.Errorf("UNNEEDED lines = %q", unneeded)
	}
	lastIs(t, out, "PASS")
}

// The base has an extra line "old" (line 3) that the feat commit deletes; the
// added blocks are line 2 ("needed") and line 5 ("extra"); a test file is
// added too and the task never reads it.
func TestPRV009_DeletionAndTestFileAreNotJudged(t *testing.T) {
	r := newRepo(t)
	test := r.charSetup(charOpts{files: files{prod: "header\nmiddle\nold\ntail\n"}, task: neededTask})
	feat := r.commit(files{prod: "header\nneeded\nmiddle\ntail\nextra\n", "tests/test_extra.py": "def test_x():\n    pass\n"}, "feat(div)")
	r.makePatch("CALC-003", files{prod: "header\nmiddle\ntail\nextra\n"})
	block := prod + ":5-5"
	_, out := r.runGreen(test, feat, greenRun{noExempt: true, proofs: true, unneededOK: []string{block}})
	if unneeded := grep(out, "UNNEEDED"); len(unneeded) != 1 || unneeded[0] != "UNNEEDED ACCEPTED "+block {
		t.Errorf("UNNEEDED lines = %q", unneeded)
	}
	lastIs(t, out, "PASS")
}

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

func TestPRV013_UnneededBlockAddedBetweenTwoAcceptanceCommits(t *testing.T) {
	r := newRepo(t)
	first := r.charSetup(charOpts{files: files{prod: prodBase}, task: neededTask})
	r.commit(files{prod: "header\nmiddle\nextra\n"}, "chore: code no test asks for")
	second := r.commit(files{"tests/test_extra.py": "def test_x():\n    pass\n"}, "test(div): acceptance")
	feat := r.commit(files{prod: "header\nneeded\nmiddle\nextra\n"}, "feat(div)")
	r.makePatch("CALC-003", files{prod: "header\nmiddle\nextra\n"})
	r.prepareProofs(neutral{}, "CALC-003")
	_, out := r.gate("green", "--slug", "div", "--test", second, "--since", first, "--feat", feat, "--task", "test", "--proofs", r.mutants)
	if unneeded := grep(out, "UNNEEDED "+prod+":4-4"); len(unneeded) == 0 || !strings.HasPrefix(unneeded[0], "UNNEEDED ") {
		t.Errorf("the block added before the second acceptance commit is not refused:\n%s", out)
	}
	lastIs(t, out, "FAIL")
}

func TestPRV014_SpecLinesAddedBetweenTwoAcceptanceCommitsAreNotUnneeded(t *testing.T) {
	const design = "docs/specs/design.md"
	r := newRepo(t)
	first := r.charSetup(charOpts{files: files{prod: prodBase, design: "# Spec\n"}, task: neededTask})
	r.commit(files{design: "# Spec\n\n### X-001\nnew criterion line\n"}, "docs: spec change")
	second := r.commit(files{"tests/test_extra.py": "def test_x():\n    pass\n"}, "test(div): acceptance")
	feat := r.commit(files{prod: "header\nneeded\nmiddle\n"}, "feat(div)")
	r.makePatch("CALC-003", files{prod: prodBase})
	r.prepareProofs(neutral{}, "CALC-003")
	_, out := r.gate("green", "--slug", "div", "--test", second, "--since", first, "--feat", feat, "--task", "test",
		"--proofs", r.mutants, "--spec", design)
	for _, ln := range grep(out, design) {
		if strings.HasPrefix(ln, "UNNEEDED") {
			t.Errorf("a spec line is judged as code: %s", ln)
		}
	}
	lastIs(t, out, "PASS")
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

// Line 4 ("extra") is a block no test needs.
func TestPRV018_MinimalCodeIsSkippedOnRequest(t *testing.T) {
	r := newRepo(t)
	test, feat := feature(r, "header\nneeded\nmiddle\nextra\n")
	code, out := r.runGreen(test, feat, greenRun{noExempt: true, proofs: true, options: []string{"--no-minimal"}})
	with(t, out, "MINIMAL SKIPPED")
	without(t, out, "UNNEEDED")
	ends(t, code, out, 0, "PASS")
	code, out = r.runGreen(test, feat, greenRun{noExempt: true, proofs: true})
	with(t, out, "UNNEEDED "+prod+":4-4")
	ends(t, code, out, 1, "FAIL")
}
