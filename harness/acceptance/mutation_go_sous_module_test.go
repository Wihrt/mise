package acceptance

import (
	"strings"
	"testing"
)

// Slug mutation-go-sous-module: MUT-006. Same fragment as MUT-002, but the Go module lives in svc/
// and the adopting repository sets dir = "svc" on [tasks."mutation:go"].

func mutationGoSvc(t *testing.T, change files) (int, string) {
	t.Helper()
	fragment := strings.Replace(mutationFragment(t), "[tasks.\"mutation:go\"]\n", "[tasks.\"mutation:go\"]\ndir = \"svc\"\n", 1)
	if !strings.Contains(fragment, "dir = \"svc\"") {
		t.Fatal("fragment has no [tasks.\"mutation:go\"] table")
	}
	r := newRepo(t)
	r.write("mise.toml", withGo(t, fragment))
	r.write("svc/go.mod", goMod)
	r.write("svc/calc.go", calcBase)
	r.write("svc/calc_test.go", unitBase)
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

func TestMUT006_SubModuleUncheckedResultFails(t *testing.T) {
	code, out := mutationGoSvc(t, files{
		"svc/calc.go":      calcDbl,
		"svc/calc_test.go": unitBase + "\nfunc TestDouble(t *testing.T) {\n\tDouble(2)\n}\n",
	})
	if code == 0 {
		t.Errorf("mutation:go succeeded although the unit test does not check the result:\n%s", out)
	}
}

func TestMUT006_SubModuleCheckedResultSucceeds(t *testing.T) {
	code, out := mutationGoSvc(t, files{
		"svc/calc.go":      calcDbl,
		"svc/calc_test.go": unitBase + "\nfunc TestDouble(t *testing.T) {\n\tif Double(2) != 6 {\n\t\tt.Fatal(\"double\")\n\t}\n}\n",
	})
	if code != 0 {
		t.Errorf("mutation:go failed although the unit test checks the result (exit %d):\n%s", code, out)
	}
}
