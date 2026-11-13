package acceptance

import (
	"strings"
	"testing"
)

// Slug forme: FRM-001, FRM-003, FRM-005.

// FRM-001: section div (the judged slug) only cites the spec; section pow keeps the old format too.
const oldFormatMD = "## pow — Power\n" +
	"### CALC-001 — a\n- **Critère** : docs/specs/x.md › CALC-001\n- **Test** : `a_test.go` › `TestCALC001_A`\n" +
	"### CALC-002 — b\n- **Critère** : docs/specs/x.md › CALC-002\n- **Test** : `a_test.go` › `TestCALC002_B`\n" +
	"\n## div — Division\n" +
	"### CALC-003 — c\n- **Critère** : docs/specs/x.md › CALC-003\n- **Test** : `a_test.go` › `TestCALC003_C`\n"

func checkIncomplete(t *testing.T, out string) {
	t.Helper()
	with(t, out, "INCOMPLETE CALC-003")
	lastIs(t, out, "FAIL")
	for _, ln := range grep(out, "INCOMPLETE") {
		if strings.Contains(ln, "CALC-001") || strings.Contains(ln, "CALC-002") {
			t.Errorf("the other section is reported: %s", ln)
		}
	}
}

func TestFRM001_RedIncompleteEntry(t *testing.T) {
	r := newRepo(t)
	sha := r.baseCommit(junit(tc{"TestCALC003_C", "fail"}), baseOpts{md: oldFormatMD})
	_, out := r.gate("red", "--slug", "div", "--test", sha, "--task", "test")
	checkIncomplete(t, out)
}

func TestFRM001_GreenIncompleteEntry(t *testing.T) {
	r := newRepo(t)
	test, feat := r.greenSetup(greenOpts{md: oldFormatMD})
	_, out := r.runGreen(test, feat, greenRun{})
	checkIncomplete(t, out)
}

func TestFRM001_CharacterizeIncompleteEntry(t *testing.T) {
	r := newRepo(t)
	md := strings.Replace(stabMD, "- **Critère** :\n  - **Étant donné** x\n  - **Quand** y\n  - **Alors** z\n- **Test** : `d_test.go`",
		"- **Critère** : docs/specs/x.md › CALC-003\n- **Test** : `d_test.go`", 1)
	if md == stabMD {
		t.Fatal("the entry of CALC-003 was not rewritten")
	}
	_, out := r.char(r.charSetup(charOpts{md: md}), charRun{})
	checkIncomplete(t, out)
}

// FRM-003: a task that exits 0 without writing a report.
func TestFRM003_CharacterizeReportMissing(t *testing.T) {
	r := newRepo(t)
	_, out := r.char(r.charSetup(charOpts{task: "exit 0\n"}), charRun{})
	missing := grep(out, "REPORT missing")
	if len(missing) == 0 || !strings.Contains(missing[0], "test") || !strings.Contains(missing[0], ".tdd/junit.xml") {
		t.Errorf("no REPORT missing line naming the task and the report:\n%s", out)
	}
	lastIs(t, out, "FAIL")
}

// FRM-005: the spec is in the repo with an uncommitted modification.
func dirtySpec(r *repo) { r.write("spec.md", spec+"\nuncommitted edit\n") }

func TestFRM005_RedSpecNotCommitted(t *testing.T) {
	r := newRepo(t)
	sha := r.charSetup(charOpts{state: "fail"}) // spec committed, acceptance commit holds test paths only, CALC-003 red
	args := []string{"red", "--slug", "div", "--test", sha, "--task", "test", "--spec", "spec.md", "--ac", "CALC-003"}
	_, out := r.gate(args...) // control: with the spec committed the gate passes
	without(t, out, "SPEC NOT COMMITTED")
	lastIs(t, out, "PASS")
	dirtySpec(r)
	_, out = r.gate(args...)
	with(t, out, "SPEC NOT COMMITTED")
	lastIs(t, out, "FAIL")
}

func TestFRM005_CharacterizeSpecNotCommitted(t *testing.T) {
	r := newRepo(t)
	sha := r.charSetup(charOpts{})
	dirtySpec(r)
	_, out := r.char(sha, charRun{})
	with(t, out, "SPEC NOT COMMITTED")
	lastIs(t, out, "FAIL")
}
