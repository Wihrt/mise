package domain

import "testing"

func TestExploratory(t *testing.T) {
	cases := map[string]bool{
		"how is authentication handled": true,
		"where the stack gets merged":   true,
		"auth|login|session|token":      true,
		"retry | backoff | timeout":     true,
		"TF_PLUGIN_CACHE_DIR":           false,
		"NewMergeStack":                 false,
		"func NewMergeStack":            false, // two words: a declaration to find
		"merge stack":                   false,
		`func \(m \*MergeStack\)`:       false,
		"permissionDecision.*ask":       false,
		"foo|bar":                       false,
		"TODO|FIXME|XXX":                true,  // plain alternatives, by the rule
		"error: cannot read the trees":  false, // punctuation: a message to find
		"status == 200 and body":        false,
		"":                              false,
		"kubectl get pods":              true,
		"harness/internal/app merge go": false,
	}
	for pattern, want := range cases {
		if got := Exploratory(pattern); got != want {
			t.Errorf("Exploratory(%q) = %v, want %v", pattern, got, want)
		}
	}
}
