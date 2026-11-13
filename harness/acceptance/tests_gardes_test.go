package acceptance

import (
	"strings"
	"testing"
)

// Slugs test-garde (FRM-002), test-garde-voisin (PRV-015), test-garde-dernier
// (PRV-016) and test-bruno (FRM-008): reading the tests of a file.

// keptMD: the judged slug (div) takes over pow's section; the kept entries stay in TESTS.md in a section of their own.
func keptMD(entries ...[3]string) string {
	md := "## div — Division\n" + entry("CALC-003", "c", "d_test.go", "TestCALC003_C") + "\n## keep — Kept\n"
	for _, e := range entries {
		md += entry(e[0], "k", e[1], e[2])
	}
	return md
}

var keptPow = keptMD([3]string{"CALC-001", "p_test.go", "TestCALC001_A"}, [3]string{"CALC-002", "p_test.go", "TestCALC002_B"})

func keptModified(out, id string) []string {
	var hits []string
	for _, ln := range grep(out, "KEPT TEST MODIFIED") {
		if strings.Contains(ln, id) {
			hits = append(hits, ln)
		}
	}
	return hits
}

// CALC-001 (kept, not listed) and CALC-002 (listed in --ac-changed) both have a line modified.
func TestFRM002_KeptTestModifiedIsReported(t *testing.T) {
	r := newRepo(t)
	modified := "package x\nfunc TestCALC001_A(t *testing.T) { _ = 1 }\nfunc TestCALC002_B(t *testing.T) { _ = 2 }\n"
	sha := r.charSetup(charOpts{md: keptPow, accFiles: files{"p_test.go": modified}})
	r.makePatch("CALC-003", killing)
	_, out := r.char(sha, charRun{}, "--replaces", "pow", "--ac-changed", "CALC-002")
	if hits := keptModified(out, "CALC-001"); len(hits) == 0 || !strings.Contains(hits[0], "p_test.go") {
		t.Errorf("the modified kept test is not reported:\n%s", out)
	}
	if hits := keptModified(out, "CALC-002"); len(hits) != 0 {
		t.Errorf("the test of the changed criterion is reported: %q", hits)
	}
	lastIs(t, out, "FAIL")
}

// CALC-001 (kept, not listed) is the last test of p_test.go; the acceptance commit appends CALC-002 after a blank line.
func TestFRM002_TestAppendedAfterKeptTestIsNotReported(t *testing.T) {
	r := newRepo(t)
	base := "package x\nfunc TestCALC001_A(t *testing.T) {}\n"
	sha := r.charSetup(charOpts{md: keptPow, files: files{"p_test.go": base},
		accFiles: files{"p_test.go": base + "\nfunc TestCALC002_B(t *testing.T) {}\n"}})
	r.makePatch("CALC-003", killing)
	_, out := r.char(sha, charRun{}, "--replaces", "pow", "--ac-changed", "CALC-002")
	with(t, out, "KILLED CALC-003") // the gate went as far as the proofs
	without(t, out, "KEPT TEST MODIFIED")
}

// CALC-001 (kept, not listed) is carried by two tests, both left intact; only CALC-002 (listed) is modified.
func TestFRM002_KeptIdentifierWithTwoTestsIsNotReported(t *testing.T) {
	r := newRepo(t)
	base := "package x\nfunc TestCALC001_A(t *testing.T) { _ = 1 }\n" +
		"func TestCALC001_Other(t *testing.T) { _ = 10 }\n" +
		"func TestCALC002_B(t *testing.T) { _ = 2 }\n"
	sha := r.charSetup(charOpts{md: keptPow, files: files{"p_test.go": base},
		accFiles: files{"p_test.go": strings.Replace(base, "{ _ = 2 }", "{ _ = 3 }", 1)}})
	r.makePatch("CALC-003", killing)
	_, out := r.char(sha, charRun{}, "--replaces", "pow", "--ac-changed", "CALC-002")
	with(t, out, "KILLED CALC-003") // the gate went as far as the proofs
	without(t, out, "KEPT TEST MODIFIED")
}

// neighbour judges an acceptance commit that rewrites the file name from base
// to acc. pow (the base commit's slug) owns the file, so the acceptance commit
// may edit it (--replaces pow). names are the tests of the base file; the
// second one is removed.
func neighbour(r *repo, base, acc, name string, names []string, kept ...[3]string) string {
	r.t.Helper()
	powMD := "## pow — Power\n" + entry("CALC-001", "p", name, names[0]) + entry("CALC-002", "p", name, names[1])
	var left []tc
	for _, n := range names {
		if n != names[1] {
			left = append(left, tc{n, "pass"})
		}
	}
	ok, ko := junit(append(left, tc{"TestCALC003_C", "pass"})...), junit(append(left, tc{"TestCALC003_C", "fail"})...)
	sha := r.charSetup(charOpts{md: keptMD(kept...), baseMD: powMD, files: files{name: base, "p_test.go": "package x\n"},
		accFiles: files{name: acc, "tests/pass.xml": ok, "tests/slow.xml": ok, "tests/fail.xml": ko}})
	r.makePatch("CALC-003", killing)
	_, out := r.char(sha, charRun{}, "--replaces", "pow")
	with(r.t, out, "KILLED CALC-003") // the gate went as far as the proofs
	return out
}

// The neighbour (CALC-002, decorated) is removed; the next test, decorated otherwise, takes its place.
func TestPRV015_PytestDecoratedNeighbourRemovedDoesNotModifyKeptTest(t *testing.T) {
	const name = "test_p.py"
	first := "import pytest\n\n\ndef test_CALC001_first():\n    assert 1\n\n\n"
	next := "@pytest.mark.parametrize(\"w\", [2])\ndef test_CALC004_next(w):\n    assert w\n"
	base := first + "@pytest.mark.parametrize(\"v\", [1])\ndef test_CALC002_removed(v):\n    assert v\n\n\n" + next
	out := neighbour(newRepo(t), base, first+next, name, []string{"test_CALC001_first", "test_CALC002_removed", "test_CALC004_next"},
		[3]string{"CALC-001", name, "test_CALC001_first"}, [3]string{"CALC-004", name, "test_CALC004_next"})
	if hits := keptModified(out, "CALC-001"); len(hits) != 0 {
		t.Errorf("the kept test is said modified: %q", hits)
	}
}

// The document that follows CALC-001's (separator and suite header included) is removed.
func TestPRV015_HelmUnittestFollowingDocumentRemovedDoesNotModifyKeptTest(t *testing.T) {
	const name = "p_suite_test.yaml"
	one := "suite: one\ntemplates:\n  - a.yaml\ntests:\n" +
		"  - it: \"[CALC-001] first\"\n    asserts:\n      - isKind:\n          of: Deployment\n"
	two := "---\nsuite: two\ntemplates:\n  - b.yaml\ntests:\n" +
		"  - it: \"[CALC-002] removed\"\n    asserts:\n      - isKind:\n          of: Service\n"
	out := neighbour(newRepo(t), one+two, one, name, []string{"[CALC-001] first", "[CALC-002] removed"},
		[3]string{"CALC-001", name, "[CALC-001] first"})
	if hits := keptModified(out, "CALC-001"); len(hits) != 0 {
		t.Errorf("the kept test is said modified: %q", hits)
	}
}

// The last test (CALC-002) is removed; the kept CALC-001 test becomes the last one, before the same final block.
func TestPRV016_KeptTestBecomingLastOfFileIsNotModified(t *testing.T) {
	const name = "test_p.py"
	head := "import sys\n\nimport pytest\n\n\ndef test_CALC001_first():\n    assert 1\n\n\n"
	trailer := "if __name__ == \"__main__\":\n    sys.exit(pytest.main([__file__, \"-v\"]))\n"
	base := head + "def test_CALC002_last():\n    assert 2\n\n\n" + trailer
	out := neighbour(newRepo(t), base, head+trailer, name, []string{"test_CALC001_first", "test_CALC002_last"},
		[3]string{"CALC-001", name, "test_CALC001_first"})
	if hits := keptModified(out, "CALC-001"); len(hits) != 0 {
		t.Errorf("the kept test is said modified: %q", hits)
	}
}

// The Go file keeps the tests of the other slug only: CALC-003 lives in the Bruno file alone.
func TestFRM008_BrunoTestCarriesItsID(t *testing.T) {
	r := newRepo(t)
	const title = "[CALC-003] the quotient is returned"
	md := strings.Replace(testsMD, "`a_test.go` › `TestCALC003_C`", "`tests/api/div.bru` › `"+title+"`", 1)
	if md == testsMD {
		t.Fatal("the entry of CALC-003 was not rewritten")
	}
	bru := "meta {\n  name: div\n  type: http\n  seq: 1\n}\n\n" +
		"get {\n  url: http://127.0.0.1:8123/div\n}\n\n" +
		"tests {\n  test(\"" + title + "\", function() {\n    expect(res.getBody().quotient).to.equal(2);\n  });\n}\n"
	r.baseCommit(junit(tc{"TestCALC001_A", "pass"}, tc{"TestCALC002_B", "pass"}, tc{title, "fail"}), baseOpts{md: md})
	sha := r.commit(files{"tests/api/div.bru": bru, "a_test.go": pTests}, "test(div): acceptance")
	code, out := r.gate("red", "--slug", "div", "--test", sha, "--task", "test")
	with(t, out, "RED CALC-003")
	without(t, out, "MISSING CALC-003")
	ends(t, code, out, 0, "PASS")
}
