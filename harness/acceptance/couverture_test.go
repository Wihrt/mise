package acceptance

import "testing"

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
