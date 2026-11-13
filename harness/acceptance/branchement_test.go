package acceptance

import (
	"strings"
	"testing"
)

// Slugs branchement (FRM-004) and revue-finale (FRM-007): the judged task is wired.

const (
	renderTask    = "test:render"
	renderCommand = "Branche : `feat/div` · Commande : `mise run " + renderTask + "`\n"
	gitlabCI      = "render:\n  script:\n    - mise run " + renderTask + "\n"
)

func withCommand(md string) string {
	return strings.Replace(md, "## div — Division\n", "## div — Division\n"+renderCommand, 1)
}

func renderGreen(r *repo, checkDepends string) (int, string) {
	r.t.Helper()
	test, feat := r.greenSetup(greenOpts{md: withCommand(testsMD), task: renderTask, checkDepends: checkDepends})
	return r.gate("green", "--slug", "div", "--test", test, "--feat", feat, "--task", renderTask, "--proof-exempt")
}

func renderChar(r *repo) (int, string) {
	r.t.Helper()
	sha := r.charSetup(charOpts{md: withCommand(stabMD), miseTask: renderTask})
	r.makePatch("CALC-003", killing)
	return r.char(sha, charRun{task: renderTask})
}

// unwired: the task is cited by no commit hook nor CI job (a hook cites another task only).
func unwired(r *repo) { r.wiring = files{".pre-commit-config.yaml": precommit("check")} }

func TestFRM004_GreenTaskNotWired(t *testing.T) {
	r := newRepo(t)
	unwired(r)
	_, out := renderGreen(r, "")
	with(t, out, "NOT WIRED test:render")
	lastIs(t, out, "FAIL")
}

func TestFRM004_CharacterizeTaskNotWired(t *testing.T) {
	r := newRepo(t)
	unwired(r)
	_, out := renderChar(r)
	with(t, out, "NOT WIRED test:render")
	lastIs(t, out, "FAIL")
}

// Once the task is cited in .pre-commit-config.yaml or .gitlab-ci.yml, the NOT WIRED line is gone.
func TestFRM004_GreenTaskWired(t *testing.T) {
	wirings := map[string]files{
		".pre-commit-config.yaml": {".pre-commit-config.yaml": precommit(renderTask)},
		".gitlab-ci.yml":          {".pre-commit-config.yaml": precommit("check"), ".gitlab-ci.yml": gitlabCI},
	}
	for name, wiring := range wirings {
		t.Run(name, func(t *testing.T) {
			r := newRepo(t)
			r.wiring = wiring
			_, out := renderGreen(r, "")
			with(t, out, "GREEN CALC-003")
			without(t, out, "NOT WIRED")
		})
	}
}

func TestFRM004_CharacterizeTaskWired(t *testing.T) {
	r := newRepo(t)
	r.wiring = files{".pre-commit-config.yaml": precommit(renderTask)}
	_, out := renderChar(r)
	with(t, out, "KILLED CALC-003")
	without(t, out, "NOT WIRED")
}

// The hook cites only `mise run check`, and test:render is a dependency of check.
func TestFRM004_GreenTaskWiredByDependency(t *testing.T) {
	r := newRepo(t)
	unwired(r)
	_, out := renderGreen(r, renderTask)
	with(t, out, "GREEN CALC-003")
	without(t, out, "NOT WIRED")
}

// The hook runs `mise run test` (default wiring); [tasks.test] aggregates test:* by pattern.
func TestFRM007_GreenTaskReachedByDependencyPattern(t *testing.T) {
	r := newRepo(t)
	test, feat := r.greenSetup(greenOpts{md: withCommand(testsMD), task: renderTask, testDepends: true})
	_, out := r.gate("green", "--slug", "div", "--test", test, "--feat", feat, "--task", renderTask, "--proof-exempt")
	with(t, out, "GREEN CALC-003")
	without(t, out, "NOT WIRED")
}

func TestFRM007_CharacterizeTaskReachedByDependencyPattern(t *testing.T) {
	r := newRepo(t)
	sha := r.charSetup(charOpts{md: withCommand(stabMD), mise: miseToml(miseOpts{task: renderTask, testDepends: true})})
	r.makePatch("CALC-003", killing)
	_, out := r.char(sha, charRun{task: renderTask})
	with(t, out, "KILLED CALC-003")
	without(t, out, "NOT WIRED")
}
