package acceptance

import (
	"strings"
	"testing"
)

// Slug preuve-necessaire: PRV-006. X-001 is CALC-003 and X-002 is CALC-004, both in the judged section div.
func TestPRV006_CriterionCoveredByAnotherIsRedundant(t *testing.T) {
	r := newRepo(t)
	both := []tc{{"TestCALC001_A", "pass"}, {"TestCALC002_B", "pass"}}
	report := func(c3, c4 string) string {
		return junit(append(append([]tc{}, both...), tc{"TestCALC003_C", c3}, tc{"TestCALC004_D", c4})...)
	}
	sha := r.charSetup(charOpts{
		md:    stabMD + entry("CALC-004", "d", "d_test.go", "TestCALC004_D"),
		task:  "mkdir -p .tdd\ns=$(cat src/state)\ncp tests/$s.xml .tdd/junit.xml\ncase \"$s\" in pass) exit 0;; *) exit 1;; esac\n",
		files: files{"spec.md": strings.Replace(spec, "### CALC-001", "### CALC-004 — d\n- **Alors** x\n### CALC-001", 1)},
		accFiles: files{
			"d_test.go":       dTests + "func TestCALC004_D(t *testing.T) {}\n",
			"tests/pass.xml":  report("pass", "pass"),
			"tests/fail.xml":  report("fail", "pass"),
			"tests/both.xml":  report("fail", "fail"),
			"tests/other.xml": report("pass", "fail"),
		},
	})
	// mutant of X-002 fails the tests of X-001 and X-002; mutant of X-001 fails only the test of X-001
	r.makePatch("CALC-003", files{"src/state": "fail\n"})
	r.makePatch("CALC-004", files{"src/state": "both\n"})
	code, out := r.char(sha, charRun{ac: "CALC-003,CALC-004"})
	redundant := grep(out, "REDUNDANT CALC-004")
	if len(redundant) == 0 {
		t.Errorf("no REDUNDANT CALC-004 line:\n%s", out)
	}
	for _, ln := range redundant {
		if !strings.Contains(ln, "CALC-003") {
			t.Errorf("the covering criterion is not named: %s", ln)
		}
	}
	without(t, out, "REDUNDANT CALC-003", "CLAUSES")
	if code == 0 {
		t.Error("exit 0, want a failure")
	}
	lastIs(t, out, "FAIL")
}
