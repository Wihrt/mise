package acceptance

import (
	"strings"
	"testing"
)

// Slug mutation-full: MUT-007. Same throwaway repositories as MUT-002/003/004, except that the weakly
// tested code is already in MUTATION_BASE (nothing changed since). Without MUTATION_FULL the task has
// no diff to mutate and succeeds; with MUTATION_FULL=1 it mutates everything and fails.

// mutationFull commits base (a mise.toml and the weak code) then runs task with MUTATION_BASE set to
// that very commit, with MUTATION_FULL=1 in the environment when full is true.
func mutationFull(t *testing.T, base files, task string, full bool) (int, string) {
	t.Helper()
	r := newRepo(t)
	for name, content := range base {
		r.write(name, content)
	}
	r.sh("git", "add", "-A")
	r.sh("git", "-c", "user.name=t", "-c", "user.email=t@example.com", "commit", "-q", "-m", "base")
	env := []string{"MUTATION_BASE=" + r.sh("git", "rev-parse", "HEAD")}
	if full {
		env = append(env, "MUTATION_FULL=1")
	}
	return run(t, r.dir, env, "mise", "run", task)
}

func fullFiles(t *testing.T, lang string) (files, string) {
	t.Helper()
	switch lang {
	case "go":
		return files{
			"mise.toml":    withGo(t, mutationFragment(t)),
			"go.mod":       goMod,
			"calc.go":      calcDbl,
			"calc_test.go": unitBase + "\nfunc TestDouble(t *testing.T) {\n\tDouble(2)\n}\n",
			".gitignore":   ".tdd\n",
		}, "mutation:go"
	case "py":
		return files{
			"mise.toml":               pyMise(t),
			"pyproject.toml":          pyProject,
			"src/demo/__init__.py":    "",
			"src/demo/calc.py":        pyDouble,
			"tests/unit/test_calc.py": strings.Replace(pyUnit, "import name", "import double, name", 1) + "\n\ndef test_double():\n    double(2)\n",
			".gitignore":              pyIgnore,
		}, "mutation:py"
	}
	return files{
		"mise.toml":                      helmMise(t),
		"charts/demo/Chart.yaml":         helmChart,
		"charts/demo/values.yaml":        helmValues,
		"charts/demo/templates/cm.yaml":  helmCMPort,
		"charts/demo/tests/cm_test.yaml": helmSuiteHd,
		".gitignore":                     helmIgnore,
	}, "mutation:helm:demo"
}

func testMUT007(t *testing.T, lang string) {
	t.Run("without MUTATION_FULL succeeds", func(t *testing.T) {
		base, task := fullFiles(t, lang)
		if code, out := mutationFull(t, base, task, false); code != 0 {
			t.Errorf("%s failed without MUTATION_FULL although nothing changed since MUTATION_BASE (exit %d):\n%s", task, code, out)
		}
	})
	t.Run("with MUTATION_FULL=1 fails", func(t *testing.T) {
		base, task := fullFiles(t, lang)
		if code, out := mutationFull(t, base, task, true); code == 0 {
			t.Errorf("%s succeeded with MUTATION_FULL=1 although the code in MUTATION_BASE is weakly tested:\n%s", task, out)
		}
	})
}

func TestMUT007_FullModeGo(t *testing.T)   { testMUT007(t, "go") }
func TestMUT007_FullModePy(t *testing.T)   { testMUT007(t, "py") }
func TestMUT007_FullModeHelm(t *testing.T) { testMUT007(t, "helm") }
