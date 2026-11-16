package gate

import (
	"bytes"
	"strings"
	"testing"
)

type trackedGit struct{ listed string }

func (g trackedGit) Run(_ string, args ...string) (int, string) {
	if args[0] == "ls-files" {
		return 0, g.listed
	}
	return 0, ""
}

type reportTask struct{ files fakeFiles }

func (t reportTask) Run(string, string, int) (TaskResult, error) {
	t.files["/r/py.lcov"] = "SF:app/a.py\nDA:1,0\nend_of_record\nSF:.venv/lib/b.py\nDA:1,0\nend_of_record\n"
	return TaskResult{}, nil
}

func TestFindingsIgnoreUntrackedFiles(t *testing.T) {
	files := fakeFiles{"/r/app/a.py": "x = 1\n"}
	var out bytes.Buffer
	New(trackedGit{listed: "app/a.py\n"}, files, reportTask{files}, noTasks{}, &out).
		Run(Options{Mode: "coverage", Task: "test", Repo: "/r", Coverage: []string{"py.lcov"}, Timeout: 5, Jobs: 1})
	got := out.String()
	if !strings.Contains(got, "UNCOVERED app/a.py:1-1") || strings.Contains(got, ".venv") {
		t.Errorf("output %q: want the tracked file judged, the untracked one ignored", got)
	}
}
