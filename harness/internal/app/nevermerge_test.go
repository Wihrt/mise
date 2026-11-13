package app

import (
	"strings"
	"testing"

	"github.com/Wihrt/mise/harness/internal/domain"
)

func TestNeverMergeAsksForMergeStack(t *testing.T) {
	p := NewNeverMerge(fieldsParser{}, fakeRepo{def: "main", cur: "feat"})
	cases := map[string]domain.Verdict{
		"harness merge-stack 14":                            domain.Ask,
		"/home/u/.local/bin/harness merge-stack feat/top":   domain.Ask,
		"harness merge-stack --validate check 14":           domain.Ask,
		"harness merge-stack --dry-run 14":                  domain.Allow,
		"harness merge-stack --help":                        domain.Allow,
		"harness hook never-merge":                          domain.Allow,
		"gh pr merge 14":                                    domain.Deny,
		"harness merge-stack --dry-run 14 ; gh pr merge 14": domain.Deny,
	}
	for command, want := range cases {
		got := p.Evaluate(domain.Request{Tool: "Bash", Command: command, Cwd: "/r"})
		if got.Verdict != want {
			t.Errorf("%q: verdict %v, want %v", command, got.Verdict, want)
		}
		if want == domain.Ask && !strings.Contains(got.Reason, "this stack only") {
			t.Errorf("%q: reason %q", command, got.Reason)
		}
	}
}
