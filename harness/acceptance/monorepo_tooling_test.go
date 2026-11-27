package acceptance

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// Slug monorepo-tooling: MNR-004. The fragments of claude/skills/repo-tooling/templates are read from the
// repository and assembled into a throwaway monorepo: monorepo.toml at the root, the Go fragment in
// services/a, the Python fragment in services/b. `mise run check` is run at the root against MUTATION_BASE.

const (
	monoGoUnit = unitBase + "\nfunc TestDouble(t *testing.T) {\n\tif Double(2) != 6 {\n\t\tt.Fatal(\"double\")\n\t}\n}\n"
	monoGoBare = unitBase + "\nfunc TestDouble(t *testing.T) {\n\tDouble(2)\n}\n"
	monoPyUnit = "from demo.calc import double, name\n\n\ndef test_name():\n    assert name() == \"calc\"\n" +
		"\n\ndef test_double():\n    assert double(2) == 6\n"
	monoPyBare = "from demo.calc import double, name\n\n\ndef test_name():\n    assert name() == \"calc\"\n" +
		"\n\ndef test_double():\n    double(2)\n"
)

func monoTemplate(t *testing.T, name string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("..", "..", "claude", "skills", "repo-tooling", "templates", name))
	if err != nil {
		t.Fatalf("fragment %s: %v", name, err)
	}
	return string(b)
}

// monoSection returns the table of a template that starts at the given header line, up to the next table.
func monoSection(t *testing.T, tmpl, header string) string {
	t.Helper()
	var out []string
	in := false
	for _, l := range strings.Split(monoTemplate(t, tmpl), "\n") {
		if strings.HasPrefix(l, "[") {
			in = l == header
		}
		if in {
			out = append(out, l)
		}
	}
	if len(out) == 0 {
		t.Fatalf("%s has no %s table", tmpl, header)
	}
	return strings.Join(out, "\n") + "\n"
}

// monoCommented returns the commented example table of templates/mise.toml that starts at header,
// uncommented, up to the next commented table: the repo-owned test:<ns> task of a service.
func monoCommented(t *testing.T, header string) string {
	t.Helper()
	var out []string
	in := false
	for _, l := range strings.Split(monoTemplate(t, "mise.toml"), "\n") {
		if strings.HasPrefix(l, "# [") {
			in = l == "# "+header
		}
		if in {
			out = append(out, strings.TrimPrefix(l, "# "))
		}
	}
	if len(out) == 0 {
		t.Fatalf("mise.toml has no commented %s example", header)
	}
	return strings.Join(out, "\n") + "\n"
}

// monoCommon is what a service takes of templates/mise.toml: its lint (with the typos check), test and check tasks. Nothing else is
// written by hand: the service is assembled from the templates only.
func monoCommon(t *testing.T) string {
	t.Helper()
	return monoSection(t, "mise.toml", "[tasks.lint]") + "\n" +
		monoSection(t, "mise.toml", `[tasks."lint:typos"]`) + "\n" +
		monoSection(t, "mise.toml", "[tasks.test]") + "\n" +
		monoSection(t, "mise.toml", "[tasks.check]")
}

// monoTypos is the typos pin of templates/mise.toml, which lint:typos needs.
func monoTypos(t *testing.T) string {
	t.Helper()
	m := regexp.MustCompile(`(?m)^typos = "[^"]+"`).FindString(monoTemplate(t, "mise.toml"))
	if m == "" {
		t.Fatal("mise.toml pins no typos")
	}
	return m + "\n"
}

// monoGoService is the Go part of mutation-go.toml (mutation:go and its tool) with the go pin of this
// repository, plus the service tasks of templates/mise.toml.
func monoGoService(t *testing.T) string {
	t.Helper()
	gotestsum := "gotestsum = \"1.13.0\"\n" // the pin of this repository, as the template's go example asks
	_, goPart, found := strings.Cut(mutationFragment(t), "# --- Go ---")
	if !found {
		t.Fatal("mutation-go.toml has no `# --- Go ---` marker")
	}
	return withGo(t, "[tools]\n"+monoTypos(t)+gotestsum) + goPart + "\n" + monoCommented(t, `[tasks."test:go"]`) + "\n" + monoCommon(t)
}

func monoPyService(t *testing.T) string {
	t.Helper()
	return "[tools]\n" + monoTypos(t) + pyTools + "\n" +
		strings.ReplaceAll(monoTemplate(t, "mutation-py.toml"), "<paquet>", "src/demo") + "\n" + monoCommented(t, `[tasks."test:py"]`) + "\n" + monoCommon(t)
}

// monorepoCheck builds a monorepo whose first commit is MUTATION_BASE, adds a Double function to the Go
// service and a double function to the Python service in a second commit, with the given unit tests, and
// runs `mise run check` at the root.
func monorepoCheck(t *testing.T, goUnit, pyUnitTest string) (int, string) {
	t.Helper()
	root := monoTemplate(t, "monorepo.toml")
	roots := regexp.MustCompile(`(?m)^(\s*config_roots\s*=\s*)\[[^\]]*\]`)
	if !roots.MatchString(root) {
		t.Fatal("monorepo.toml declares no config_roots")
	}
	root = roots.ReplaceAllString(root, `${1}["services/a", "services/b"]`)
	goSvc, pySvc := monoGoService(t), monoPyService(t)

	r := newRepo(t)
	r.write("mise.toml", root)
	r.write(".gitignore", pyIgnore)
	r.write("services/a/mise.toml", goSvc)
	r.write("services/a/go.mod", goMod)
	r.write("services/a/calc.go", calcBase)
	r.write("services/a/calc_test.go", unitBase)
	r.write("services/b/mise.toml", pySvc)
	r.write("services/b/pyproject.toml", pyProject)
	r.write("services/b/src/demo/__init__.py", "")
	r.write("services/b/src/demo/calc.py", pyBase)
	r.write("services/b/tests/unit/test_calc.py", pyUnit)
	r.sh("git", "add", "-A")
	r.sh("git", "-c", "user.name=t", "-c", "user.email=t@example.com", "commit", "-q", "-m", "base")
	base := r.sh("git", "rev-parse", "HEAD")
	r.write("services/a/calc.go", calcDbl)
	r.write("services/a/calc_test.go", goUnit)
	r.write("services/b/src/demo/calc.py", pyDouble)
	r.write("services/b/tests/unit/test_calc.py", pyUnitTest)
	r.sh("git", "add", "-A")
	r.sh("git", "-c", "user.name=t", "-c", "user.email=t@example.com", "commit", "-q", "-m", "change")
	c, o, e := runBoth(t, r.dir, []string{"MUTATION_BASE=" + base}, "mise", "run", "check")
	return c, o + e
}

func TestMNR004_CheckedResultsSucceed(t *testing.T) {
	code, out := monorepoCheck(t, monoGoUnit, monoPyUnit)
	if code != 0 {
		t.Errorf("check failed although both services' unit tests check their results (exit %d):\n%s", code, out)
	}
}

func TestMNR004_PythonUncheckedResultFails(t *testing.T) {
	code, out := monorepoCheck(t, monoGoUnit, monoPyBare)
	if code == 0 {
		t.Errorf("check succeeded although the Python unit test does not check the result:\n%s", out)
	}
}

func TestMNR004_GoUncheckedResultFails(t *testing.T) {
	code, out := monorepoCheck(t, monoGoBare, monoPyUnit)
	if code == 0 {
		t.Errorf("check succeeded although the Go unit test does not check the result:\n%s", out)
	}
}
