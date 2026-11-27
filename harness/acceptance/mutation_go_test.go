package acceptance

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// Slug mutation-go: MUT-002. The fragment claude/skills/repo-tooling/templates/mutation-go.toml
// is read from the repository (not copied) and used as the mise.toml of a throwaway Go repository,
// where `mise run mutation:go` is run against MUTATION_BASE.

const (
	goMod    = "module example.com/calc\n\ngo 1.22\n"
	calcBase = "package calc\n\nfunc Name() string { return \"calc\" }\n"
	calcDbl  = calcBase + "\nfunc Double(x int) int { return x * 3 }\n"
	unitBase = "package calc\n\nimport \"testing\"\n\nfunc TestName(t *testing.T) {\n\tif Name() != \"calc\" {\n\t\tt.Fatal(\"name\")\n\t}\n}\n"
)

func mutationFragment(t *testing.T) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("..", "..", "claude", "skills", "repo-tooling", "templates", "mutation-go.toml"))
	if err != nil {
		t.Fatalf("fragment mutation-go.toml: %v", err)
	}
	return string(b)
}

// withGo makes the throwaway repository declare its own go, as a real repo adopting the fragment does,
// with the version this repository pins in .config/mise.toml.
func withGo(t *testing.T, fragment string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("..", "..", ".config", "mise.toml"))
	if err != nil {
		t.Fatalf(".config/mise.toml: %v", err)
	}
	m := regexp.MustCompile(`(?m)^go\s*=\s*"([^"]+)"`).FindSubmatch(b)
	if m == nil {
		t.Fatal("no go version pinned in .config/mise.toml")
	}
	if !strings.Contains(fragment, "[tools]\n") {
		t.Fatal("fragment has no [tools] table")
	}
	return strings.Replace(fragment, "[tools]\n", "[tools]\ngo = \""+string(m[1])+"\"\n", 1)
}

// mutationGo builds a Go repository whose first commit is MUTATION_BASE, applies change in a second
// commit, runs `mise run mutation:go` and returns its exit code and output.
func mutationGo(t *testing.T, change files) (int, string) {
	t.Helper()
	fragment := mutationFragment(t)
	r := newRepo(t)
	r.write("mise.toml", withGo(t, fragment))
	r.write("go.mod", goMod)
	r.write("calc.go", calcBase)
	r.write("calc_test.go", unitBase)
	r.write(".gitignore", ".tdd\n")
	r.sh("git", "add", "-A")
	r.sh("git", "-c", "user.name=t", "-c", "user.email=t@example.com", "commit", "-q", "-m", "base")
	base := r.sh("git", "rev-parse", "HEAD")
	for name, content := range change {
		r.write(name, content)
	}
	r.sh("git", "add", "-A")
	r.sh("git", "-c", "user.name=t", "-c", "user.email=t@example.com", "commit", "-q", "-m", "change")
	return run(t, r.dir, []string{"MUTATION_BASE=" + base}, "mise", "run", "mutation:go")
}

func TestMUT002_UncheckedResultFails(t *testing.T) {
	code, out := mutationGo(t, files{
		"calc.go":      calcDbl,
		"calc_test.go": unitBase + "\nfunc TestDouble(t *testing.T) {\n\tDouble(2)\n}\n",
	})
	if code == 0 {
		t.Errorf("mutation:go succeeded although the unit test does not check the result:\n%s", out)
	}
}

func TestMUT002_BinaryOnlyFails(t *testing.T) {
	code, out := mutationGo(t, files{
		"calc.go": calcDbl,
		"cmd/calc/main.go": "package main\n\nimport (\n\t\"fmt\"\n\n\t\"example.com/calc\"\n)\n\n" +
			"func main() { fmt.Println(calc.Double(2)) }\n",
		"e2e/e2e_test.go": "package e2e\n\nimport (\n\t\"os/exec\"\n\t\"path/filepath\"\n\t\"strings\"\n\t\"testing\"\n)\n\n" +
			"func TestBinary(t *testing.T) {\n\tbin := filepath.Join(t.TempDir(), \"calc\")\n" +
			"\tif out, err := exec.Command(\"go\", \"build\", \"-o\", bin, \"../cmd/calc\").CombinedOutput(); err != nil {\n\t\tt.Fatalf(\"build: %v\\n%s\", err, out)\n\t}\n" +
			"\tout, err := exec.Command(bin).Output()\n\tif err != nil {\n\t\tt.Fatal(err)\n\t}\n" +
			"\tif strings.TrimSpace(string(out)) != \"6\" {\n\t\tt.Fatalf(\"got %q\", out)\n\t}\n}\n",
	})
	if code == 0 {
		t.Errorf("mutation:go succeeded although only a binary-running test calls the function:\n%s", out)
	}
}

func TestMUT002_CheckedResultSucceeds(t *testing.T) {
	code, out := mutationGo(t, files{
		"calc.go":      calcDbl,
		"calc_test.go": unitBase + "\nfunc TestDouble(t *testing.T) {\n\tif Double(2) != 6 {\n\t\tt.Fatal(\"double\")\n\t}\n}\n",
	})
	if code != 0 {
		t.Errorf("mutation:go failed although the unit test checks the result (exit %d):\n%s", code, out)
	}
}

func TestMUT002_NoGoChangeSucceeds(t *testing.T) {
	code, out := mutationGo(t, files{"README.md": "hello\n"})
	if code != 0 {
		t.Errorf("mutation:go failed although no Go file changed (exit %d):\n%s", code, out)
	}
}
