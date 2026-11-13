package tdd

import (
	"reflect"
	"slices"
	"strconv"
	"strings"
	"testing"
)

func entry(id, title, test string) string {
	return "### " + id + " — " + title + "\n- **Critère** :\n\t- **Étant donné** x\n\t- **Quand** y\n\t- **Alors** z\n" +
		"- **Test** : `a_test.go` › `" + test + "`\n"
}

var testsMD = strings.ReplaceAll("## pow — Power\n"+entry("CALC-001", "a", "TestCALC001_A")+entry("CALC-002", "b", "TestCALC002_B")+
	"\n## div — Division\n"+entry("CALC-003", "c", "TestCALC003_C"), "\t", "  ")

const spec = "# s\n## Critères d'acceptation\n### CALC-003 — c\n- **Alors** x\n" +
	"### CALC-001 — a\n- **Alors** x\n### CALC-002 — b\n- **Alors** x\n## Annexe\n### CALC-009 — not a criterion\n"

func keys(m map[string]bool) []string {
	var k []string
	for s := range m {
		k = append(k, s)
	}
	slices.Sort(k)
	return k
}

func TestCandidate(t *testing.T) {
	tests := map[string]string{
		"TestCALC001_Add":          "CALC001",
		"TestCALC001_Add/sub_case": "CALC001",
		"test_CALC002_sub":         "CALC002",
		"[ING-001] has a name":     "ING001",
		"CALC003_sub_is_exact":     "CALC003",
		"test_CALC_001_add":        "CALC001",
		"TestSHA256Sum":            "", // legacy: digits followed by a letter
		"TestHTTP2Handler":         "", // legacy: fewer than 3 digits
		"test_S3_upload":           "",
		"TestMain":                 "",
		"TestParseHelper":          "",
	}
	for name, want := range tests {
		if got := Candidate(name); got != want {
			t.Errorf("Candidate(%q) = %q, want %q", name, got, want)
		}
	}
}

func TestNorm(t *testing.T) {
	for _, s := range []string{"CALC-001", "TestCALC001", "test_CALC001"} {
		if got := Norm(s); got != "CALC001" {
			t.Errorf("Norm(%q) = %q", s, got)
		}
	}
}

func TestSectionIDs(t *testing.T) {
	if got := SectionIDs("## calc\n\n### CALC-001: title\n### CALC-002 other\n", "calc"); !slices.Equal(got, []string{"CALC-001", "CALC-002"}) {
		t.Errorf("trailing colon: %v", got)
	}
	if got := SectionIDs(testsMD, "pow"); !slices.Equal(got, []string{"CALC-001", "CALC-002"}) {
		t.Errorf("pow: %v", got)
	}
	if got := SectionIDs(testsMD, "div"); !slices.Equal(got, []string{"CALC-003"}) {
		t.Errorf("div: %v", got)
	}
	if got := SectionIDs(testsMD, "nope"); len(got) != 0 {
		t.Errorf("nope: %v", got)
	}
	if got := keys(AllIDs(testsMD)); !slices.Equal(got, []string{"CALC001", "CALC002", "CALC003"}) {
		t.Errorf("AllIDs: %v", got)
	}
}

func TestIncompleteHeadings(t *testing.T) {
	if got := IncompleteHeadings(testsMD); len(got) != 0 {
		t.Errorf("complete file: %v", got)
	}
	if got := IncompleteHeadings(testsMD + "### CALC-004 — d\n- **Spec** : only spec\n"); !slices.Equal(got, []string{"### CALC-004 — d"}) {
		t.Errorf("missing test line: %v", got)
	}
	if got := IncompleteHeadings("### ING-002 — t\n- **Critère** : `s.md` › ING-002\n- **Test** : `a` › `b`\n"); len(got) != 0 {
		t.Errorf("Critère is accepted: %v", got)
	}
}

func TestIncompleteEntries(t *testing.T) {
	md := "## s — S\n### A-001 — a\n- **Étant donné** x\n- **Quand** y\n### A-002 — b\n- **Étant donné** x\n- **Quand** y\n- **Alors** z\n"
	if got := IncompleteEntries(md, "s"); !slices.Equal(got, []string{"A-001"}) {
		t.Errorf("IncompleteEntries = %v", got)
	}
}

func cands(entries []Entry) []string {
	var c []string
	for _, e := range entries {
		c = append(c, e.Cand)
	}
	return c
}

func TestScan(t *testing.T) {
	entries, _ := Scan("a_test.go", "package x\n"+
		"// TestCALC999_InComment mentions SHA-256 and ISO-8601\n"+
		"func TestSHA256Sum(t *testing.T) {}\n"+
		"func TestHTTP2Handler(t *testing.T) {}\n"+
		"func TestCALC001_A(t *testing.T) { Foo() }\n"+
		"func TestCALC002_B(t *testing.T) {}\n")
	if got := cands(entries); !slices.Equal(got, []string{"", "", "CALC001", "CALC002"}) {
		t.Fatalf("go: %v", got)
	}
	if !strings.Contains(entries[2].Body, "Foo()") || strings.Contains(entries[3].Body, "Foo()") {
		t.Errorf("bodies: %q / %q", entries[2].Body, entries[3].Body)
	}
	if entries[2].Line != 5 {
		t.Errorf("line = %d, want 5", entries[2].Line)
	}
	files := map[string]string{
		"t.py":         "def test_CALC001_add():\n    pass\ndef test_S3_upload():\n    pass\n",
		"h.yaml":       "tests:\n  - it: \"[ING-002] x\"\n    asserts: []\n",
		"m.tftest.hcl": "run \"CALC003_y\" {\n}\n",
		"x.bats":       "@test \"BAT-001 does\" {\n}\n",
		"api.bru":      "tests {\n  test(\"[API-004] status is 200\", function() {\n  });\n}\n",
	}
	want := map[string][]string{"t.py": {"CALC001", ""}, "h.yaml": {"ING002"}, "m.tftest.hcl": {"CALC003"}, "x.bats": {"BAT001"}, "api.bru": {"API004"}}
	for f, content := range files {
		e, _ := Scan(f, content)
		if got := cands(e); !slices.Equal(got, want[f]) {
			t.Errorf("%s: %v, want %v", f, got, want[f])
		}
	}
}

func TestScanBodyStopsBeforeNextDecoratorAndTrailer(t *testing.T) {
	entries, pre := Scan("t.py", "import x\n\ndef test_A001_a():\n    pass\n\n@pytest.mark.x\ndef test_A002_b():\n    pass\n\nif __name__ == '__main__':\n    main()\n")
	if pre != "import x\n" {
		t.Errorf("preamble = %q", pre)
	}
	if entries[0].Body != "def test_A001_a():\n    pass" || entries[1].Body != "def test_A002_b():\n    pass" {
		t.Errorf("bodies: %q / %q", entries[0].Body, entries[1].Body)
	}
}

const junitXML = `<testsuites><testsuite name="s">
<testcase name="TestCALC001_A"/>
<testcase name="TestCALC002_B"><failure message="x"/></testcase>
<testcase name="TestCALC003_C"><error message="x"/></testcase>
<testcase name="TestCALC004_D"><skipped/></testcase>
</testsuite></testsuites>`

func TestParseJUnit(t *testing.T) {
	r := ParseJUnit(junitXML)
	want := []Case{{"TestCALC001_A", Pass}, {"TestCALC002_B", Fail}, {"TestCALC003_C", Fail}, {"TestCALC004_D", Skipped}}
	if r.State != ReportOK || !slices.Equal(r.Cases, want) {
		t.Errorf("report = %+v", r)
	}
	if got := len(ResultsByID(r)); got != 4 {
		t.Errorf("ResultsByID has %d ids, want 4", got)
	}
	states := map[string]string{
		"":                                      ReportEmpty,
		"<testsuites><testsuite/></testsuites>": ReportEmpty,
		"not xml <<":                            ReportUnreadable,
		"plain text":                            ReportUnreadable,
	}
	for data, want := range states {
		if got := ParseJUnit(data).State; got != want {
			t.Errorf("ParseJUnit(%q).State = %q, want %q", data, got, want)
		}
	}
}

func TestParseTofuJSON(t *testing.T) {
	r := ParseTofuJSON(`{"type":"test_run","test_run":{"run":"CALC001_ok","status":"pass"}}
{"type":"test_run","test_run":{"run":"CALC002_bad","status":"fail"}}
{"type":"test_run","test_run":{"run":"CALC003_unk","status":"error"}}
{"type":"diagnostic","diagnostic":{"summary":"Reference to undeclared output value","detail":"An output value with the name \"missing\" has not been declared."}}`)
	var got []string
	for _, c := range r.Cases {
		got = append(got, c.Status)
	}
	if !slices.Equal(got, []string{Pass, Fail, Fail}) {
		t.Errorf("statuses = %v", got)
	}
	errs := CompileErrors(r.Diag)
	if len(errs) == 0 || !slices.Contains(errs[0].Symbols, "missing") {
		t.Errorf("compile errors = %+v", errs)
	}
	if got := ParseTofuJSON("garbage\n").State; got != ReportUnreadable {
		t.Errorf("garbage: %q", got)
	}
}

func TestCompileErrorsSymbols(t *testing.T) {
	tests := map[string]string{
		"internal/calc/div_test.go:9:14: undefined: Div":        "Div",
		"NameError: name 'mul' is not defined":                  "mul",
		"ImportError: cannot import name 'mul' from 'pkg.calc'": "mul",
		"E   ModuleNotFoundError: No module named 'pkg.newmod'": "newmod",
	}
	for line, sym := range tests {
		errs := CompileErrors(line)
		if len(errs) != 1 || !slices.Contains(errs[0].Symbols, sym) {
			t.Errorf("CompileErrors(%q) = %+v, want symbol %q", line, errs, sym)
		}
	}
}

func TestSpecCriteria(t *testing.T) {
	want := []string{"CALC-003", "CALC-001", "CALC-002"}
	if got := SpecCriteria(spec); !slices.Equal(got, want) {
		t.Errorf("french heading: %v", got)
	}
	if got := SpecCriteria(strings.Replace(spec, "Critères d'acceptation", "Acceptance criteria", 1)); !slices.Equal(got, want) {
		t.Errorf("english heading: %v", got)
	}
	if got := SpecCriteria("# s\n### CALC-003 — c\n"); len(got) != 0 {
		t.Errorf("no heading: %v", got)
	}
}

func TestSectionTests(t *testing.T) {
	want := map[string]map[string]bool{"pow": {"a_test.go": true}, "div": {"a_test.go": true}}
	if got := SectionTests(testsMD); !reflect.DeepEqual(got, want) {
		t.Errorf("SectionTests = %v", got)
	}
}

func TestReplacedIDs(t *testing.T) {
	stab := strings.ReplaceAll(strings.Replace(testsMD, "`a_test.go` › `TestCALC003_C`", "`d_test.go` › `TestCALC003_C`", 1), "`a_test.go`", "`p_test.go`")
	divOnly := "## div — Division\n### CALC-003 — c\n- **Test** : `d_test.go` › `TestCALC003_C`\n"
	powPartial := "## pow — Power\n### CALC-002 — b\n- **Test** : `p_test.go` › `TestCALC002_B`\n\n" + divOnly
	if got := keys(ReplacedIDs(stab, divOnly, []string{"pow"})); !slices.Equal(got, []string{"CALC-001", "CALC-002"}) {
		t.Errorf("section dropped: %v", got)
	}
	if got := keys(ReplacedIDs(stab, powPartial, []string{"pow"})); !slices.Equal(got, []string{"CALC-001"}) {
		t.Errorf("section kept in part: %v", got)
	}
	if got := ReplacedIDs(stab, divOnly, []string{"div"}); len(got) != 0 {
		t.Errorf("section kept: %v", got)
	}
}

func TestClauseCount(t *testing.T) {
	tests := map[string]int{
		"- **Alors** z": 1,
		"  - **Alors**\n    - a\n    - b\n  - **Test** : x": 2,
		"- **Quand** y":        0,
		"- **Alors** :\n  - a": 1,
		"- **Alors**\n  - a\n  suite\n  - b\n  - c\n- **Test**": 3,
	}
	for entry, want := range tests {
		if got := ClauseCount(entry); got != want {
			t.Errorf("ClauseCount(%q) = %d, want %d", entry, got, want)
		}
	}
	if got := ClauseCount(EntryText(spec, "CALC-001")); got != 1 {
		t.Errorf("entry of the spec: %d", got)
	}
}

func TestOutOfScope(t *testing.T) {
	md := "## a — A\nCommande : `mise run test`\n### A-001 — x\n## b — B\nCommande : `mise run test:e2e`\n### B-001 — y\n### B-002 — z\n"
	got := OutOfScope(md, "test", map[string]bool{"B002": true})
	if len(got) != 1 || got["B001"] != "`mise run test:e2e`" {
		t.Errorf("OutOfScope = %v", got)
	}
	if got := OutOfScope(md, "test:e2e", nil); len(got) != 1 || got["A001"] == "" {
		t.Errorf("other task: %v", got)
	}
}

func TestWiredCommand(t *testing.T) {
	md := "## a — A\nCommande : `mise run test:render`\n### A-001 — x\n## b — B\n### B-001 — y\n"
	if got := WiredCommand(md, "a"); got != "test:render" {
		t.Errorf("a: %q", got)
	}
	if got := WiredCommand(md, "b"); got != "" {
		t.Errorf("b: %q", got)
	}
}

func TestIsTestPath(t *testing.T) {
	yes := []string{"TESTS.md", "mise.toml", "pkg/a_test.go", "test_x.py", "tests/data.json", "m/x.tftest.hcl", "a/testdata/f", "conftest.py"}
	no := []string{"main.go", "src/test.py", "pkg/latest/a.go", "README.md"}
	for _, p := range yes {
		if !IsTestPath(p) {
			t.Errorf("IsTestPath(%q) = false", p)
		}
	}
	for _, p := range no {
		if IsTestPath(p) {
			t.Errorf("IsTestPath(%q) = true", p)
		}
	}
}

func TestMisePatchProblem(t *testing.T) {
	old := []string{"[tools]", `go = "1"`, "[tasks.test]", `run = "go test"`, `[tasks."test:e2e"]`, `run = "x"`}
	patch := func(start int, body string) string {
		return "diff --git a/mise.toml b/mise.toml\n--- a/mise.toml\n+++ b/mise.toml\n@@ -" + strconv.Itoa(start) + ",1 +1,1 @@\n" + body
	}
	if got := MisePatchProblem(old, patch(4, "-run = \"go test\"\n+run = \"true\"\n"), "test"); got != "touches test task test" {
		t.Errorf("judged task changed: %q", got)
	}
	if got := MisePatchProblem(old, patch(6, "-run = \"x\"\n+run = \"y\"\n"), "test"); got != "" {
		t.Errorf("other task changed: %q", got)
	}
	if got := MisePatchProblem(old, patch(6, "-run = \"x\"\n+run = \"y\"\n"), "test:e2e"); got != "touches test task test:e2e" {
		t.Errorf("quoted task changed: %q", got)
	}
	if got := MisePatchProblem(old, "diff --git a/x.go b/x.go\n@@ -4 +4 @@\n-a\n+b\n", "test"); got != "" {
		t.Errorf("other file: %q", got)
	}
}

func TestRedundant(t *testing.T) {
	kills := []Kills{
		{"A001", []map[string]bool{{"A001": true, "B001": true}, {"A001": true, "B001": true, "C001": true}}},
		{"B001", []map[string]bool{{"B001": true}}},
	}
	if got := Redundant(kills); !reflect.DeepEqual(got, [][2]string{{"A001", "B001"}}) {
		t.Errorf("Redundant = %v", got)
	}
}

func TestAddedBlocks(t *testing.T) {
	diff := "diff --git a/src/a.go b/src/a.go\n--- a/src/a.go\n+++ b/src/a.go\n@@ -3,0 +4,2 @@ func\n+x\n+y\n@@ -9 +11 @@\n-a\n+b\n@@ -20,2 +21,0 @@\n-a\n-b\n" +
		"diff --git a/a_test.go b/a_test.go\n--- a/a_test.go\n+++ b/a_test.go\n@@ -1 +1,3 @@\n+z\n"
	want := []Block{{"src/a.go", 4, 5}, {"src/a.go", 11, 11}}
	if got := AddedBlocks(diff); !slices.Equal(got, want) {
		t.Errorf("AddedBlocks = %v", got)
	}
}

func TestMiseNonToolsAndWiring(t *testing.T) {
	if got := MiseNonTools("[tools]\ngo = \"1\"\n[tasks.test]\nrun = \"x\"\n"); !slices.Equal(got, []string{"[tasks.test]", `run = "x"`}) {
		t.Errorf("MiseNonTools = %v", got)
	}
	if !IsWiringFile(".github/workflows/ci.yml") || !IsWiringFile(".pre-commit-config.yaml") || IsWiringFile("ci.yml") {
		t.Error("IsWiringFile")
	}
	if got := MiseRunTasks("entry: mise run test:render\nrun: mise run  check && mise run lint"); !slices.Equal(got, []string{"test:render", "check", "lint"}) {
		t.Errorf("MiseRunTasks = %v", got)
	}
	if !Fnmatch("test:render", "test:*") || Fnmatch("lint:go", "test:*") || !Fnmatch("a", "[ab]") {
		t.Error("Fnmatch")
	}
}

func TestLinesAndContainsWord(t *testing.T) {
	if got := Lines("a\r\nb\rc\n\nd"); !slices.Equal(got, []string{"a", "b", "c", "", "d"}) {
		t.Errorf("Lines = %q", got)
	}
	if !ContainsWord("x := Div(1)", "Div") || ContainsWord("x := Divide(1)", "Div") || !ContainsWord("pkg.newmod", "pkg.newmod") {
		t.Error("ContainsWord")
	}
}
