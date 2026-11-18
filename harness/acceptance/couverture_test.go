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
