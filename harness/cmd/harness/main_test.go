package main

import (
	"bytes"
	"strings"
	"testing"
)

func noEnv(string) string { return "" }

func TestRun(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		stdin      string
		wantCode   int
		wantStdout string
	}{
		{"usage", []string{"hook"}, "", 2, ""},
		{"unknown hook", []string{"hook", "nope"}, "{}", 2, ""},
		{"unreadable input allows", []string{"hook", "never-merge"}, "not json", 0, ""},
		{"allowed call writes nothing", []string{"hook", "never-merge"}, `{"tool_name":"Bash","tool_input":{"command":"git status"}}`, 0, ""},
		{"denied call", []string{"hook", "never-merge"}, `{"tool_name":"Bash","tool_input":{"command":"gh pr merge 1"}}`, 0, `"permissionDecision":"deny"`},
		{"asked call", []string{"hook", "cluster-guard"}, `{"tool_name":"Bash","tool_input":{"command":"tofu apply"}}`, 0, `"permissionDecision":"ask"`},
		{"hinted call decides nothing", []string{"hook", "search-hint"}, `{"tool_name":"Grep","tool_input":{"pattern":"how is auth handled"}}`, 0, `{"hookSpecificOutput":{"hookEventName":"PreToolUse","additionalContext":"This search reads like a question`},
		{"exact search writes nothing", []string{"hook", "search-hint"}, `{"tool_name":"Grep","tool_input":{"pattern":"NewMergeStack"}}`, 0, ""},
		{"role argument", []string{"hook", "agent-guard", "implementer"}, `{"tool_name":"Bash","tool_input":{"command":"git push"}}`, 0, "implementer must not push"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			code := run(tt.args, strings.NewReader(tt.stdin), &stdout, &stderr, noEnv)
			if code != tt.wantCode {
				t.Errorf("exit code = %d, want %d (stderr: %s)", code, tt.wantCode, stderr.String())
			}
			if tt.wantStdout == "" && stdout.Len() != 0 {
				t.Errorf("stdout = %q, want nothing", stdout.String())
			}
			if !strings.Contains(stdout.String(), tt.wantStdout) {
				t.Errorf("stdout = %q, want it to contain %q", stdout.String(), tt.wantStdout)
			}
		})
	}
}
