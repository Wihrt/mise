package acceptance

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Slug mutation-py: MUT-003. The fragment claude/skills/repo-tooling/templates/mutation-py.toml is read
// from the repository and merged with the common part of mutation-go.toml into the mise.toml of a
// throwaway Python repository (package src/demo), where `mise run mutation:py` is run against MUTATION_BASE.

const (
	pyProject = `[project]
name = "demo"
version = "0.1.0"
requires-python = ">=3.13"

[build-system]
requires = ["hatchling"]
build-backend = "hatchling.build"

[tool.hatch.build.targets.wheel]
packages = ["src/demo"]

[dependency-groups]
dev = ["pytest", "pytest-gremlins==1.15.0"]

[tool.pytest-gremlins]
paths = ["src/demo"]
`
	pyBase    = "def name():\n    return \"calc\"\n"
	pyDouble  = pyBase + "\n\ndef double(x):\n    return x * 3\n"
	pyExtra   = "def extra():\n    return \"extra\"\n"
	pyUnit    = "from demo.calc import name\n\n\ndef test_name():\n    assert name() == \"calc\"\n"
	pyIgnore  = ".tdd\n.venv\n__pycache__\ncoverage\n.gremlins\n.pytest_cache\n*.egg-info\n"
	pyTools   = "uv = \"0.13.7\"\npython = \"3.13.18\"\n"
	pyDeleted = "<deleted>"
)

// pyMise merges the common part of mutation-go.toml (up to the Go section), uv and python, and the
// mutation-py.toml fragment with `<paquet>` replaced by the package directory.
func pyMise(t *testing.T) string {
	t.Helper()
	goFrag := mutationFragment(t)
	common, _, found := strings.Cut(goFrag, "# --- Go ---")
	if !found {
		t.Fatal("mutation-go.toml has no `# --- Go ---` marker after the common part")
	}
	const endTools = "# --- end common part (tools) ---\n"
	if !strings.Contains(common, endTools) {
		t.Fatal("mutation-go.toml has no end-of-tools marker in the common part")
	}
	common = strings.Replace(common, endTools, endTools+pyTools, 1)
	b, err := os.ReadFile(filepath.Join("..", "..", "claude", "skills", "repo-tooling", "templates", "mutation-py.toml"))
	if err != nil {
		t.Fatalf("fragment mutation-py.toml: %v", err)
	}
	return common + "\n" + strings.ReplaceAll(string(b), "<paquet>", "src/demo")
}

// mutationPy builds a Python repository whose first commit is MUTATION_BASE, applies change in a
// second commit (a value of pyDeleted deletes the file), runs `mise run mutation:py` and returns its
// exit code and output.
func mutationPy(t *testing.T, change files) (int, string) {
	t.Helper()
	mise := pyMise(t)
	r := newRepo(t)
	r.write("mise.toml", mise)
	r.write("pyproject.toml", pyProject)
	r.write("src/demo/__init__.py", "")
	r.write("src/demo/calc.py", pyBase)
	r.write("src/demo/extra.py", pyExtra)
	r.write("tests/unit/test_calc.py", pyUnit)
	r.write(".gitignore", pyIgnore)
	r.sh("git", "add", "-A")
	r.sh("git", "-c", "user.name=t", "-c", "user.email=t@example.com", "commit", "-q", "-m", "base")
	base := r.sh("git", "rev-parse", "HEAD")
	for name, content := range change {
		if content == pyDeleted {
			if err := os.Remove(filepath.Join(r.dir, name)); err != nil {
				t.Fatal(err)
			}
			continue
		}
		r.write(name, content)
	}
	r.sh("git", "add", "-A")
	r.sh("git", "-c", "user.name=t", "-c", "user.email=t@example.com", "commit", "-q", "-m", "change")
	return run(t, r.dir, []string{"MUTATION_BASE=" + base}, "mise", "run", "mutation:py")
}

func TestMUT003_UncheckedResultFails(t *testing.T) {
	code, out := mutationPy(t, files{
		"src/demo/calc.py": pyDouble,
		"tests/unit/test_calc.py": strings.Replace(pyUnit, "import name", "import double, name", 1) +
			"\n\ndef test_double():\n    double(2)\n",
	})
	if code == 0 {
		t.Errorf("mutation:py succeeded although the unit test does not check the result:\n%s", out)
	}
}

func TestMUT003_UntestedFunctionFails(t *testing.T) {
	code, out := mutationPy(t, files{"src/demo/calc.py": pyDouble})
	if code == 0 {
		t.Errorf("mutation:py succeeded although no unit test covers the new function:\n%s", out)
	}
}

func TestMUT003_CheckedResultSucceeds(t *testing.T) {
	code, out := mutationPy(t, files{
		"src/demo/calc.py": pyDouble,
		"tests/unit/test_calc.py": strings.Replace(pyUnit, "import name", "import double, name", 1) +
			"\n\ndef test_double():\n    assert double(2) == 6\n",
	})
	if code != 0 {
		t.Errorf("mutation:py failed although the unit tests check the result (exit %d):\n%s", code, out)
	}
}

func TestMUT003_NoPythonChangeSucceeds(t *testing.T) {
	code, out := mutationPy(t, files{"README.md": "hello\n"})
	if code != 0 {
		t.Errorf("mutation:py failed although no Python file changed (exit %d):\n%s", code, out)
	}
}

func TestMUT003_DeletedFileSucceeds(t *testing.T) {
	code, out := mutationPy(t, files{"src/demo/extra.py": pyDeleted})
	if code != 0 {
		t.Errorf("mutation:py failed although a package file was only deleted (exit %d):\n%s", code, out)
	}
}
