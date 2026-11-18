package acceptance

import (
	"path/filepath"
	"regexp"
	"strconv"
	"testing"
)

// The Go file under measure, and its blocks as the Go toolchain writes them in
// a coverage profile: the entry of Run (line 5), the body of the if (lines
// 6-7), the last return (line 8) and, when the file has Noop, the empty body of
// Noop, a block without statement that always has the count 0.
const (
	covModule = "go.mod"
	covApp    = "internal/app/app.go"
	covGoMod  = "module example.test/app\n\ngo 1.22\n"
	covRun    = "package app\n\n// Run labels n.\nfunc Run(n int) string {\n\tif n > 0 {\n\t\treturn \"positive\"\n\t}\n\treturn \"other\"\n}\n"
	covNoop   = "\n// Noop does nothing.\nfunc Noop() {}\n"
	covReport = ".tdd/cover/go.txt"
)

// covFiles is the repository content; withNoop adds the empty function.
func covFiles(withNoop bool) files {
	src := covRun
	if withNoop {
		src += covNoop
	}
	return files{covModule: covGoMod, covApp: src}
}

// profile is the Go coverage profile of that file: the counts of the entry, of
// the if body and of the last return of Run, then the empty block of Noop.
func profile(withNoop bool, entry, body, tail string) string {
	const path = "example.test/app/internal/app/app.go:"
	p := "mode: set\n" +
		path + "5.2,5.11 1 " + entry + "\n" +
		path + "6.3,7.1 1 " + body + "\n" +
		path + "8.2,8.16 1 " + tail + "\n"
	if withNoop {
		p += path + "12.14,12.14 0 0\n"
	}
	return p
}

const (
	hit  = "1"
	miss = "0"
)

func TestCOV001_UncoveredGoBlockFails(t *testing.T) {
	r := covRepo(t, covFiles(false), files{covReport: profile(false, hit, miss, hit)}, 0)
	code, out := r.coverage()
	with(t, out, "UNCOVERED internal/app/app.go:6-7")
	ends(t, code, out, 1, "FAIL")
}

func TestCOV001_AllCoveredPasses(t *testing.T) {
	r := covRepo(t, covFiles(false), files{covReport: profile(false, hit, hit, hit)}, 0)
	code, out := r.coverage()
	without(t, out, "UNCOVERED")
	with(t, out, "ERROR RETURNS IGNORED 0")
	ends(t, code, out, 0, "PASS")
}

func TestCOV001_EmptyBlockIsNotJudged(t *testing.T) {
	r := covRepo(t, covFiles(true), files{covReport: profile(true, hit, hit, hit)}, 0)
	code, out := r.coverage()
	without(t, out, "UNCOVERED")
	ends(t, code, out, 0, "PASS")
}

// The Go file of a marker method and its profile as the toolchain writes it
// (go test -coverprofile, go1.27.4): the only function has an empty body, so
// the only block has no statement, starts and ends at the same position.
func TestCOV012_FileOfEmptyFunctionsIsMeasured(t *testing.T) {
	const event = "internal/app/event.go"
	r := covRepo(t, files{
		covModule: covGoMod,
		event:     "package app\n\n// Placed is an event.\ntype Placed struct{}\n\nfunc (Placed) isEvent() {}\n",
	}, files{covReport: "mode: set\nexample.test/app/internal/app/event.go:6.26,6.26 0 0\n"}, 0)
	code, out := r.coverage()
	without(t, out, "NOT MEASURED", "UNCOVERED")
	ends(t, code, out, 0, "PASS")
}

func TestCOV003_ReportsComplementEachOther(t *testing.T) {
	r := covRepo(t, covFiles(false), files{
		".tdd/cover/a.txt": profile(false, hit, miss, miss),
		".tdd/cover/b.txt": profile(false, miss, hit, miss),
	}, 0)
	code, out := r.coverage()
	with(t, out, "UNCOVERED internal/app/app.go:8-8")
	without(t, out, "UNCOVERED internal/app/app.go:5-5", "UNCOVERED internal/app/app.go:6-7")
	ends(t, code, out, 1, "FAIL")
}

func TestCOV007_ReportNotWrittenFails(t *testing.T) {
	r := covRepo(t, covFiles(false), files{covReport: ""}, 0)
	r.write(covReport, profile(false, hit, hit, hit))
	code, out := r.coverage()
	with(t, out, "COVERAGE missing", covReport)
	ends(t, code, out, 1, "FAIL")
}

func TestCOV007_FailedTaskFails(t *testing.T) {
	r := covRepo(t, covFiles(false), files{covReport: profile(false, hit, hit, hit)}, 1)
	code, out := r.coverage()
	with(t, out, "TASK FAILED")
	ends(t, code, out, 1, "FAIL")
}

// The Python file of the real trial of 16/11 (coverage.py 7.17.3, branch mode)
// and its LCOV record as the tool wrote it. Lines 19 to 24 are never run (line
// 22 is not measured) and line 19 carries branches counted "-"; line 16 ran and
// carries a branch never taken.
const (
	covPy     = "src/spike/server.py"
	covPyLcov = ".tdd/cover/py.lcov"
	covPySrc  = "import os\n" +
		"import signal\n" +
		"import threading\n" +
		"from http.server import BaseHTTPRequestHandler, HTTPServer\n" +
		"\n" +
		"\n" +
		"def classify(n):\n" +
		"    if n > 0:\n" +
		"        return \"positive\"\n" +
		"    else:\n" +
		"        return \"non-positive\"\n" +
		"\n" +
		"\n" +
		"class Handler(BaseHTTPRequestHandler):\n" +
		"    def do_GET(self):\n" +
		"        if self.path == \"/hello\":\n" +
		"            body = classify(1).encode()\n" +
		"            self.send_response(200)\n" +
		"        elif self.path == \"/never\":\n" +
		"            body = b\"never called\"\n" +
		"            self.send_response(200)\n" +
		"        else:\n" +
		"            body = b\"nf\"\n" +
		"            self.send_response(404)\n" +
		"        self.end_headers()\n" +
		"        self.wfile.write(body)\n" +
		"\n" +
		"    def log_message(self, *a):\n" +
		"        pass\n" +
		"\n" +
		"\n" +
		"def main():\n" +
		"    srv = HTTPServer((\"0.0.0.0\", int(os.environ.get(\"PORT\", \"8000\"))), Handler)\n" +
		"    if os.environ.get(\"GRACEFUL\") == \"1\":\n" +
		"        # shutdown() must run from another thread than serve_forever()\n" +
		"        signal.signal(signal.SIGTERM,\n" +
		"                      lambda s, f: threading.Thread(target=srv.shutdown).start())\n" +
		"    print(\"listening\", flush=True)\n" +
		"    srv.serve_forever()\n" +
		"\n" +
		"\n" +
		"if __name__ == \"__main__\":\n" +
		"    main()\n"
	covPyRec = `SF:src/spike/server.py
DA:1,1
DA:2,1
DA:3,1
DA:4,1
DA:7,1
DA:8,1
DA:9,1
DA:11,0
DA:14,1
DA:15,1
DA:16,1
DA:17,1
DA:18,1
DA:19,0
DA:20,0
DA:21,0
DA:23,0
DA:24,0
DA:25,1
DA:26,1
DA:28,1
DA:29,1
DA:32,1
DA:33,1
DA:34,1
DA:36,0
DA:38,1
DA:39,1
DA:42,1
DA:43,1
LF:30
LH:23
FN:7,11,classify
FNDA:1,classify
FN:15,26,Handler.do_GET
FNDA:1,Handler.do_GET
FN:28,29,Handler.log_message
FNDA:1,Handler.log_message
FN:32,39,main
FNDA:1,main
FNF:4
FNH:4
BRDA:8,0,jump to line 9,1
BRDA:8,0,jump to line 11,0
BRDA:16,0,jump to line 17,1
BRDA:16,0,jump to line 19,0
BRDA:19,0,jump to line 20,-
BRDA:19,0,jump to line 23,-
BRDA:34,0,jump to line 36,0
BRDA:34,0,jump to line 38,1
BRDA:42,0,jump to line 43,1
BRDA:42,0,exit the module,0
BRF:10
BRH:4
end_of_record
`
)

func TestCOV002_UncoveredPythonLinesAndBranch(t *testing.T) {
	r := covRepo(t, files{covPy: covPySrc}, files{covPyLcov: covPyRec}, 0)
	code, out := r.coverage()
	with(t, out, "UNCOVERED src/spike/server.py:19-24", "UNCOVERED BRANCH src/spike/server.py:16")
	without(t, out, "UNCOVERED BRANCH src/spike/server.py:19")
	ends(t, code, out, 1, "FAIL")
}

// The pragmas sit on lines 3 and 4; every measured line and branch is covered.
const (
	covPragmaPy  = "app/util.py"
	covPragmaSrc = "def f(n):\n    if n:\n        return 1  # pragma: no cover\n    for x in n:  # pragma: no branch\n        pass\n    return 2\n"
	covPragmaRec = "SF:app/util.py\nDA:1,1\nDA:2,1\nDA:3,1\nDA:4,1\nDA:5,1\nDA:6,1\nBRDA:2,0,jump to line 3,1\nBRDA:2,0,jump to line 4,1\nBRDA:4,0,jump to line 5,1\nBRDA:4,0,jump to line 6,1\nend_of_record\n"
)

func TestCOV008_PragmaIsRefused(t *testing.T) {
	r := covRepo(t, files{covPragmaPy: covPragmaSrc}, files{covPyLcov: covPragmaRec}, 0)
	code, out := r.coverage()
	with(t, out, "PRAGMA app/util.py:3", "PRAGMA app/util.py:4")
	without(t, out, "UNCOVERED")
	ends(t, code, out, 1, "FAIL")
}

// The Go file of the real output of go1.27.4 (go build -cover): Run and Code,
// whose default clause (line 17) is the real case of COV-005.
const (
	covCodeSrc = "package app\n" +
		"\n" +
		"// Run labels n.\n" +
		"func Run(n int) string {\n" +
		"\tif n > 0 {\n" +
		"\t\treturn \"positive\"\n" +
		"\t}\n" +
		"\treturn \"other\"\n" +
		"}\n" +
		"\n" +
		"// Code maps a name to an exit code.\n" +
		"func Code(name string) int {\n" +
		"\tswitch name {\n" +
		"\tcase \"ok\":\n" +
		"\t\treturn 0\n" +
		"\tdefault:\n" +
		"\t\treturn 127\n" +
		"\t}\n" +
		"}\n"
	covCodeMD = "## internal/app/app.go\n\n- `return 127` : chemin de secours\n"
)

// codeProfile is the profile when Run(0) and Code("ok") ran, with the counts of
// the if body (lines 6-7) and of the default clause (line 17).
func codeProfile(body, deflt string) string {
	const path = "example.test/app/internal/app/app.go:"
	return "mode: set\n" +
		path + "5.2,5.11 1 1\n" +
		path + "6.3,7.1 1 " + body + "\n" +
		path + "8.2,8.16 1 1\n" +
		path + "13.2,13.14 1 1\n" +
		path + "15.3,15.11 1 1\n" +
		path + "17.3,17.13 1 " + deflt + "\n"
}

func codeFiles(extra files) files {
	return merge(files{covModule: covGoMod, covApp: covCodeSrc}, extra)
}

func TestCOV005_WrittenExceptionIsAccepted(t *testing.T) {
	r := covRepo(t, codeFiles(files{"COVERAGE.md": covCodeMD}), files{covReport: codeProfile(hit, miss)}, 0)
	code, out := r.coverage()
	with(t, out, "ACCEPTED internal/app/app.go:17-17: chemin de secours")
	without(t, out, "UNCOVERED")
	ends(t, code, out, 0, "PASS")
}

func TestCOV005_StaleExceptionFails(t *testing.T) {
	r := covRepo(t, codeFiles(files{"COVERAGE.md": covCodeMD}), files{covReport: codeProfile(hit, hit)}, 0)
	code, out := r.coverage()
	with(t, out, "STALE EXCEPTION internal/app/app.go: return 127")
	ends(t, code, out, 1, "FAIL")
}

// Tracked files that no report names.
const (
	covUnmeasuredPy = "src/tool/main.py"
	covTestPy       = "tests/test_app.py"
	covPortsGo      = "internal/app/ports.go"
	covYaml         = "ci/pipeline.yaml"
)

var notJudged = regexp.MustCompile(`NOT JUDGED (\d+) files`)

func unmeasuredFiles(withYaml bool, extra files) files {
	f := merge(covFiles(false), files{
		covUnmeasuredPy: "def main():\n    return 1\n",
		covTestPy:       "def test_main():\n    assert True\n",
		covPortsGo:      "package app\n\n// Limit is the maximum.\nconst Limit = 3\n\n// Item is a thing.\ntype Item struct{ Name string }\n",
	}, extra)
	if withYaml {
		f[covYaml] = "steps:\n  - test\n"
	}
	return f
}

func judgedCount(t *testing.T, out string) int {
	t.Helper()
	m := notJudged.FindStringSubmatch(out)
	if m == nil {
		t.Fatalf("no NOT JUDGED line in:\n%s", out)
	}
	n, err := strconv.Atoi(m[1])
	if err != nil {
		t.Fatal(err)
	}
	return n
}

func TestCOV006_UnmeasuredFileFails(t *testing.T) {
	r := covRepo(t, unmeasuredFiles(true, nil), files{covReport: profile(false, hit, hit, hit)}, 0)
	code, out := r.coverage()
	with(t, out, "NOT MEASURED "+covUnmeasuredPy)
	without(t, out, "NOT MEASURED "+covTestPy, "NOT MEASURED "+covYaml, "NOT MEASURED "+covPortsGo)
	ends(t, code, out, 1, "FAIL")
	// The YAML file is counted: one more NOT JUDGED file than without it.
	plain := covRepo(t, unmeasuredFiles(false, nil), files{covReport: profile(false, hit, hit, hit)}, 0)
	_, outPlain := plain.coverage()
	if got, want := judgedCount(t, out), judgedCount(t, outPlain)+1; got != want {
		t.Errorf("NOT JUDGED counts %d files with the YAML file, want %d", got, want)
	}
}

func TestCOV006_WholeFileExceptionIsAccepted(t *testing.T) {
	md := "## " + covUnmeasuredPy + "\n\n- fichier entier : jamais importé\n"
	r := covRepo(t, unmeasuredFiles(true, files{"COVERAGE.md": md}), files{covReport: profile(false, hit, hit, hit)}, 0)
	code, out := r.coverage()
	with(t, out, "ACCEPTED "+covUnmeasuredPy+": jamais importé")
	ends(t, code, out, 0, "PASS")
}

func TestCOV009_BaselineWritesTheDebt(t *testing.T) {
	r := covRepo(t, codeFiles(nil), files{covReport: codeProfile(miss, miss)}, 0)
	code, out := r.coverage("--baseline")
	ends(t, code, out, 0, "PASS")
	md := read(t, filepath.Join(r.dir, "COVERAGE.md"))
	with(t, md, "## internal/app/app.go", "- `return \"positive\"` : dette initiale", "- `return 127` : dette initiale")
	code, out = r.coverage()
	with(t, out, "ACCEPTED internal/app/app.go:6-7", "ACCEPTED internal/app/app.go:17-17")
	without(t, out, "UNCOVERED")
	ends(t, code, out, 0, "PASS")
}

func TestCOV009_BaselineWritesNothingWhenTaskFails(t *testing.T) {
	r := covRepo(t, codeFiles(nil), files{covReport: codeProfile(miss, miss)}, 1)
	code, out := r.coverage("--baseline")
	with(t, out, "TASK FAILED")
	ends(t, code, out, 1, "FAIL")
	if exists(filepath.Join(r.dir, "COVERAGE.md")) {
		t.Error("COVERAGE.md was written although the task failed")
	}
}

// The Go file of the real output of go1.27.4 (go build -cover), whose profile
// is below when Load("go.mod", 0) ran: lines 12-13 and 15-16 never ran.
const covLoadSrc = "package app\n" +
	"\n" +
	"import (\n" +
	"\t\"errors\"\n" +
	"\t\"os\"\n" +
	")\n" +
	"\n" +
	"// Load reads a file and checks a return code.\n" +
	"func Load(path string, rc int) (string, error) {\n" +
	"\tdata, err := os.ReadFile(path)\n" +
	"\tif err != nil {\n" +
	"\t\treturn \"\", err\n" +
	"\t}\n" +
	"\tif rc != 0 {\n" +
	"\t\treturn \"\", errors.New(\"command failed\")\n" +
	"\t}\n" +
	"\treturn string(data), nil\n" +
	"}\n"

const covLoadProfile = "mode: set\n" +
	"example.test/app/internal/app/app.go:10.2,11.16 2 1\n" +
	"example.test/app/internal/app/app.go:12.3,13.1 1 0\n" +
	"example.test/app/internal/app/app.go:14.2,14.13 1 1\n" +
	"example.test/app/internal/app/app.go:15.3,16.1 1 0\n" +
	"example.test/app/internal/app/app.go:17.2,17.26 1 1\n"

func TestCOV004_ErrorReturnIsIgnoredAndCounted(t *testing.T) {
	r := covRepo(t, files{covModule: covGoMod, covApp: covLoadSrc}, files{covReport: covLoadProfile}, 0)
	code, out := r.coverage()
	without(t, out, "UNCOVERED internal/app/app.go:12-13")
	with(t, out, "ERROR RETURNS IGNORED 1", "UNCOVERED internal/app/app.go:15-16")
	ends(t, code, out, 1, "FAIL")
}

// Two blocks never run, each the whole body of an if on err made of one return:
// lines 9-10 return the tested error, lines 13-14 swallow it. Positions as the
// Go toolchain writes them for this file.
const covSwallowSrc = "package app\n" +
	"\n" +
	"import \"os\"\n" +
	"\n" +
	"// Load reads a file.\n" +
	"func Load(path string) (string, error) {\n" +
	"\tdata, err := os.ReadFile(path)\n" +
	"\tif err != nil {\n" +
	"\t\treturn \"\", err\n" +
	"\t}\n" +
	"\t_, err = os.Stat(path)\n" +
	"\tif err != nil {\n" +
	"\t\treturn \"\", nil\n" +
	"\t}\n" +
	"\treturn string(data), nil\n" +
	"}\n"

const covSwallowProfile = "mode: set\n" +
	"example.test/app/internal/app/app.go:7.2,8.16 2 1\n" +
	"example.test/app/internal/app/app.go:9.3,10.1 1 0\n" +
	"example.test/app/internal/app/app.go:11.2,12.16 2 1\n" +
	"example.test/app/internal/app/app.go:13.3,14.1 1 0\n" +
	"example.test/app/internal/app/app.go:15.2,15.26 1 1\n"

func TestCOV010_ReturnSwallowingTheErrorIsJudged(t *testing.T) {
	r := covRepo(t, files{covModule: covGoMod, covApp: covSwallowSrc}, files{covReport: covSwallowProfile}, 0)
	code, out := r.coverage()
	without(t, out, "UNCOVERED internal/app/app.go:9-10")
	with(t, out, "ERROR RETURNS IGNORED 1", "UNCOVERED internal/app/app.go:13-14")
	ends(t, code, out, 1, "FAIL")
}

// Three pragma spellings coverage.py recognizes, on lines 3, 4 and 5; every
// measured line and branch is covered.
const (
	covSpellPy  = "app/spell.py"
	covSpellSrc = "def f(n):\n    if n:\n        a = 1  # pragma no cover\n        b = 2  # PRAGMA: NO COVER\n    for x in n:  #  pragma:no branch\n        pass\n    return 2\n"
	covSpellRec = "SF:app/spell.py\nDA:1,1\nDA:2,1\nDA:3,1\nDA:4,1\nDA:5,1\nDA:6,1\nDA:7,1\nBRDA:2,0,jump to line 3,1\nBRDA:2,0,jump to line 5,1\nBRDA:5,0,jump to line 6,1\nBRDA:5,0,jump to line 7,1\nend_of_record\n"
)

func TestCOV011_RecognizedPragmaSpellingsAreRefused(t *testing.T) {
	r := covRepo(t, files{covSpellPy: covSpellSrc}, files{covPyLcov: covSpellRec}, 0)
	code, out := r.coverage()
	with(t, out, "PRAGMA app/spell.py:3", "PRAGMA app/spell.py:4", "PRAGMA app/spell.py:5")
	ends(t, code, out, 1, "FAIL")
}
