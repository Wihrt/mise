package shell

import (
	"slices"
	"testing"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name    string
		command string
		want    [][]string
	}{
		{"simple", "git push origin feat", [][]string{{"git", "push", "origin", "feat"}}},
		{"list and pipe", "helm template c | kubectl apply -f - && echo ok",
			[][]string{{"helm", "template", "c"}, {"kubectl", "apply", "-f", "-"}, {"echo", "ok"}}},
		{"quoted argument stays one word", `git commit -m "doc: kubectl delete is guarded"`,
			[][]string{{"git", "commit", "-m", "doc: kubectl delete is guarded"}}},
		{"command substitution is seen", `echo "$(kubectl delete pod x)"`,
			[][]string{{"echo", "$(kubectl delete pod x)"}, {"kubectl", "delete", "pod", "x"}}},
		{"subshell", "(cd /r && git merge feat)", [][]string{{"cd", "/r"}, {"git", "merge", "feat"}}},
		{"assignments are not words", "NO_PROXY=x kubectl get pods", [][]string{{"kubectl", "get", "pods"}}},
		{"unreadable line is cut on its separators", "git push 'origin ; gh pr merge", [][]string{{"git", "push", "'origin"}, {"gh", "pr", "merge"}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewParser().Parse(tt.command)
			if len(got) != len(tt.want) {
				t.Fatalf("Parse(%q) = %v, want %v", tt.command, got, tt.want)
			}
			for i := range got {
				if !slices.Equal(got[i].Words, tt.want[i]) {
					t.Errorf("invocation %d = %q, want %q", i, got[i].Words, tt.want[i])
				}
			}
		})
	}
}
