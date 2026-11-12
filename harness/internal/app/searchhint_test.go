package app

import (
	"testing"

	"github.com/Wihrt/mise/harness/internal/domain"
)

func TestSearchHint(t *testing.T) {
	p := NewSearchHint(fieldsParser{})
	cases := []struct {
		name string
		req  domain.Request
		want domain.Verdict
	}{
		{"question in the search tool", domain.Request{Tool: "Grep", Pattern: "where are tokens refreshed"}, domain.Hint},
		{"identifier in the search tool", domain.Request{Tool: "Grep", Pattern: "refreshToken"}, domain.Allow},
		{"alternatives through grep", domain.Request{Tool: "Bash", Command: "grep -rn -e auth|login|session src"}, domain.Hint},
		{"alternatives through rtk rg", domain.Request{Tool: "Bash", Command: "rtk proxy rg -g *.go retry|backoff|timeout"}, domain.Hint},
		{"exact string through grep", domain.Request{Tool: "Bash", Command: "grep -rn TF_PLUGIN_CACHE_DIR ."}, domain.Allow},
		{"context flag value is not the pattern", domain.Request{Tool: "Bash", Command: "grep -A 3 refreshToken src"}, domain.Allow},
		{"another program", domain.Request{Tool: "Bash", Command: "echo how|is|this"}, domain.Allow},
		{"second command of the line", domain.Request{Tool: "Bash", Command: "cd src ; rg cache|index|store"}, domain.Hint},
		{"file edit", domain.Request{Tool: "Edit", Path: "/r/a.go"}, domain.Allow},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := p.Evaluate(c.req); got.Verdict != c.want {
				t.Errorf("verdict %v, want %v", got.Verdict, c.want)
			}
		})
	}
}
