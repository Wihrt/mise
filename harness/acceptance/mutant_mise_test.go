package acceptance

import (
	"fmt"
	"regexp"
	"strings"
	"testing"
)

// Slug mutant-mise: PRV-008. The tested code is the body of
// [tasks."bootstrap:agent"]; the judged task is test:bootstrap. Both bodies
// are long multi-line strings: the mutated line is far below the table header,
// which a 3-line-context patch does not show.

const bootTask = "test:bootstrap"

func taskBody(marker string) string {
	pad := ""
	for i := 1; i <= 8; i++ {
		pad += fmt.Sprintf("# step %d\n", i)
	}
	return pad + marker + "\n# done\n# done\n# done\n"
}

var (
	bootBody  = taskBody("echo pass")
	judgeBody = taskBody("sh run_test.sh")
	bootMise  = "[tasks.\"" + bootTask + "\"]\nrun = \"\"\"\n" + judgeBody + "\"\"\"\n" +
		"[tasks.check]\nrun = \"exit 0\"\n" +
		"[tasks.\"bootstrap:agent\"]\nrun = \"\"\"\n" + bootBody + "\"\"\"\n"
	judged = regexp.MustCompile(`(?m)^(KILLED|SURVIVED) CALC-003`)
)

func judgeMise(r *repo, base, patch files) string {
	r.t.Helper()
	r.wiring = files{".pre-commit-config.yaml": precommit(bootTask)}
	sha := r.charSetup(charOpts{miseTask: bootTask, mise: bootMise, files: base})
	r.makePatch("CALC-003", patch)
	_, out := r.char(sha, charRun{task: bootTask})
	return out
}

func TestPRV008_MutantOfCodeTaskIsJudged(t *testing.T) {
	out := judgeMise(newRepo(t), nil, files{"mise.toml": strings.Replace(bootMise, bootBody, taskBody("echo mutated"), 1)})
	if !judged.MatchString(out) {
		t.Errorf("the mutant was not judged:\n%s", out)
	}
}

func TestPRV008_MutantOfTestTaskIsRefused(t *testing.T) {
	out := judgeMise(newRepo(t), nil, files{"mise.toml": strings.Replace(bootMise, judgeBody, taskBody("sh run_test.sh && true"), 1)})
	if !regexp.MustCompile(`(?m)^MUTANT INVALID CALC-003.*touches test task test:bootstrap`).MatchString(out) {
		t.Errorf("the mutant was not refused:\n%s", out)
	}
}

// Another file whose mutated line number is, in mise.toml, a line of the body of test:bootstrap.
func TestPRV008_MutantOfOtherFileIsJudged(t *testing.T) {
	other := ""
	for i := 1; i <= 30; i++ {
		other += fmt.Sprintf("line %d\n", i)
	}
	first := 3 // 1-based number of the first body line: the table header is line 1
	last := first + strings.Count(judgeBody, "\n") - 1
	n := first + 5
	if n < first || n > last {
		t.Fatalf("line %d is outside the body of the task (%d-%d)", n, first, last)
	}
	mutated := strings.Replace(other, fmt.Sprintf("line %d\n", n), fmt.Sprintf("mutated %d\n", n), 1)
	out := judgeMise(newRepo(t), files{"src/code.txt": other}, files{"src/code.txt": mutated})
	if !judged.MatchString(out) {
		t.Errorf("the mutant was not judged:\n%s", out)
	}
	without(t, out, "MUTANT INVALID")
}
