package acceptance

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Slug mutation-helm: MUT-004. The fragment claude/skills/repo-tooling/templates/mutation-helm.toml is
// read from the repository and merged with the common part of mutation-go.toml into the mise.toml of a
// throwaway repository holding the chart charts/demo, where `mise run mutation:helm:demo` is run
// against MUTATION_BASE.

const (
	helmChart   = "apiVersion: v2\nname: demo\nversion: 0.1.0\n"
	helmValues  = "{}\n"
	helmCMBase  = "apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: demo\ndata:\n  greeting: hello\n"
	helmCMPort  = helmCMBase + "  port: \"8080\"\n"
	helmExtra   = "apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: extra\ndata:\n  other: value\n"
	helmSuiteHd = "suite: cm\ntemplates:\n  - templates/cm.yaml\ntests:\n  - it: renders the config map\n    asserts:\n      - equal:\n          path: data.greeting\n          value: hello\n"
	helmSuitePt = "      - equal:\n          path: data.port\n          value: \"8080\"\n"
	helmSuiteMd = "      - equal:\n          path: apiVersion\n          value: v1\n      - equal:\n          path: kind\n          value: ConfigMap\n      - equal:\n          path: metadata.name\n          value: demo\n"
	helmIgnore  = ".tdd\n"
)

// helmMise merges the common part of mutation-go.toml (up to the Go section) and the mutation-helm.toml
// fragment with `<chart>` replaced by demo and `<chart-dir>` by charts/demo.
func helmMise(t *testing.T) string {
	t.Helper()
	common, _, found := strings.Cut(mutationFragment(t), "# --- Go ---")
	if !found {
		t.Fatal("mutation-go.toml has no `# --- Go ---` marker after the common part")
	}
	b, err := os.ReadFile(filepath.Join("..", "..", "claude", "skills", "repo-tooling", "templates", "mutation-helm.toml"))
	if err != nil {
		t.Fatalf("fragment mutation-helm.toml: %v", err)
	}
	frag := strings.ReplaceAll(string(b), "<chart-dir>", "charts/demo")
	frag = strings.ReplaceAll(frag, "<chart>", "demo")
	return common + "\n" + frag
}

// mutationHelm builds a repository whose first commit (MUTATION_BASE) holds the chart, applies change
// in a second commit, runs `mise run mutation:helm:demo` and returns its exit code and output.
func mutationHelm(t *testing.T, change files) (int, string) {
	t.Helper()
	mise := helmMise(t)
	r := newRepo(t)
	r.write("mise.toml", mise)
	r.write("charts/demo/Chart.yaml", helmChart)
	r.write("charts/demo/values.yaml", helmValues)
	r.write("charts/demo/templates/cm.yaml", helmCMBase)
	r.write("charts/demo/tests/cm_test.yaml", helmSuiteHd)
	r.write(".gitignore", helmIgnore)
	r.sh("git", "add", "-A")
	r.sh("git", "-c", "user.name=t", "-c", "user.email=t@example.com", "commit", "-q", "-m", "base")
	base := r.sh("git", "rev-parse", "HEAD")
	for name, content := range change {
		r.write(name, content)
	}
	r.sh("git", "add", "-A")
	r.sh("git", "-c", "user.name=t", "-c", "user.email=t@example.com", "commit", "-q", "-m", "change")
	return run(t, r.dir, []string{"MUTATION_BASE=" + base}, "mise", "run", "mutation:helm:demo")
}

func TestMUT004_UncheckedFieldFails(t *testing.T) {
	code, out := mutationHelm(t, files{"charts/demo/templates/cm.yaml": helmCMPort})
	if code == 0 {
		t.Errorf("mutation:helm:demo succeeded although no assertion covers the changed field:\n%s", out)
	}
}

func TestMUT004_UnrenderedTemplateFails(t *testing.T) {
	code, out := mutationHelm(t, files{"charts/demo/templates/extra.yaml": helmExtra})
	if code == 0 {
		t.Errorf("mutation:helm:demo succeeded although no suite renders the changed template:\n%s", out)
	}
}

// helm-mutation-test mutates the whole touched template, so the success suite checks every field of it.
func TestMUT004_CheckedFieldSucceeds(t *testing.T) {
	code, out := mutationHelm(t, files{
		"charts/demo/templates/cm.yaml":  helmCMPort,
		"charts/demo/tests/cm_test.yaml": helmSuiteHd + helmSuiteMd + helmSuitePt,
	})
	if code != 0 {
		t.Errorf("mutation:helm:demo failed although the suite checks every field of the changed template (exit %d):\n%s", code, out)
	}
}

func TestMUT004_NoChartChangeSucceeds(t *testing.T) {
	code, out := mutationHelm(t, files{"README.md": "hello\n"})
	if code != 0 {
		t.Errorf("mutation:helm:demo failed although no chart file changed (exit %d):\n%s", code, out)
	}
}
