package app

import (
	"errors"
	"strings"
	"testing"

	"github.com/Wihrt/mise/harness/internal/domain"
)

// fakeForge is a forge whose default branch is main. Commits are named after
// their content: "main", "a", "b"...; merging a request moves main to its head.
type fakeForge struct {
	prs        map[int]*domain.PullRequest
	checks     map[int]domain.Checks
	mergeErr   map[int]error
	mainChecks domain.Checks
	main       *string // commit of main, shared with fakeHistory
	drift      string  // when set, the commit main ends on instead of the merged head
	calls      []string
}

func (f *fakeForge) DefaultBranch() (string, error) { return "main", nil }

func (f *fakeForge) Get(ref string) (domain.PullRequest, error) {
	for _, pr := range f.prs {
		if pr.Head == ref {
			return *pr, nil
		}
	}
	return domain.PullRequest{}, errors.New("not found")
}

func (f *fakeForge) ByHead(branch string) (domain.PullRequest, bool, error) {
	pr, err := f.Get(branch)
	return pr, err == nil && pr.Open, nil
}

func (f *fakeForge) Retarget(number int, base string) error {
	f.calls = append(f.calls, "retarget "+f.prs[number].Head)
	f.prs[number].Base = base
	return nil
}

func (f *fakeForge) Checks(number int) (domain.Checks, error) { return f.checks[number], nil }

func (f *fakeForge) Merge(number int, sha string) error {
	if err := f.mergeErr[number]; err != nil {
		return err
	}
	f.calls = append(f.calls, "merge "+f.prs[number].Head)
	f.prs[number].Open = false
	*f.main = sha
	if f.drift != "" {
		*f.main = f.drift
	}
	return nil
}

func (f *fakeForge) CommitChecks(string) (domain.Checks, error) { return f.mainChecks, nil }

// fakeHistory knows which commit contains which, and gives every commit its own tree.
type fakeHistory struct {
	main      *string
	contains  map[string][]string // commit -> the commits it contains
	validated []string
	validErr  error
}

func (h *fakeHistory) Fetch() error                  { return nil }
func (h *fakeHistory) Remote(string) (string, error) { return *h.main, nil }
func (h *fakeHistory) Tree(rev string) (string, error) {
	return "tree-" + rev, nil
}
func (h *fakeHistory) IsAncestor(a, d string) bool {
	if a == d {
		return true
	}
	for _, c := range h.contains[d] {
		if c == a {
			return true
		}
	}
	return false
}
func (h *fakeHistory) ChangedFiles(string, string) ([]string, error) {
	return []string{"drift.txt"}, nil
}
func (h *fakeHistory) Run(sha, command string) error {
	h.validated = append(h.validated, command+"@"+sha)
	return h.validErr
}

// threeHigh is a stack a <- b <- c on main, every pipeline green.
func threeHigh() (*fakeForge, *fakeHistory) {
	main := "main"
	forge := &fakeForge{
		main: &main,
		prs: map[int]*domain.PullRequest{
			1: {Number: 1, Head: "a", Base: "main", SHA: "a", Open: true, URL: "u/1"},
			2: {Number: 2, Head: "b", Base: "a", SHA: "b", Open: true, URL: "u/2"},
			3: {Number: 3, Head: "c", Base: "b", SHA: "c", Open: true, URL: "u/3"},
		},
		checks:   map[int]domain.Checks{},
		mergeErr: map[int]error{},
	}
	history := &fakeHistory{main: &main, contains: map[string][]string{
		"a": {"main"}, "b": {"main", "a"}, "c": {"main", "a", "b"},
	}}
	return forge, history
}

func runStack(forge *fakeForge, history *fakeHistory, opt MergeOptions) (string, bool) {
	var out strings.Builder
	ok := NewMergeStack(forge, history, history, &out).Run(opt)
	return out.String(), ok
}

func wantLines(t *testing.T, out string, lines ...string) {
	t.Helper()
	for _, l := range lines {
		if !strings.Contains(out, l) {
			t.Errorf("output lacks %q:\n%s", l, out)
		}
	}
}

func TestMergeStackMergesBottomFirstAndProves(t *testing.T) {
	forge, history := threeHigh()
	out, ok := runStack(forge, history, MergeOptions{Top: "c"})
	if !ok {
		t.Fatalf("want success:\n%s", out)
	}
	if got := strings.Join(forge.calls, ", "); got != "merge a, retarget b, merge b, retarget c, merge c" {
		t.Errorf("calls = %s", got)
	}
	wantLines(t, out, "STACK 3: #1 a -> #2 b -> #3 c", "MERGED #3 u/3", "TREE IDENTICAL main is the code of #3", "CHECKS PASS main")
	if !strings.HasSuffix(out, "PASS\n") {
		t.Errorf("want PASS last:\n%s", out)
	}
}

func TestMergeStackOfOne(t *testing.T) {
	forge, history := threeHigh()
	out, ok := runStack(forge, history, MergeOptions{Top: "a"})
	if !ok || strings.Join(forge.calls, ", ") != "merge a" {
		t.Fatalf("ok=%v calls=%v\n%s", ok, forge.calls, out)
	}
}

func TestMergeStackDryRunMergesNothing(t *testing.T) {
	forge, history := threeHigh()
	out, ok := runStack(forge, history, MergeOptions{Top: "c", DryRun: true})
	if !ok || len(forge.calls) != 0 {
		t.Fatalf("ok=%v calls=%v", ok, forge.calls)
	}
	wantLines(t, out, "READY", "PASS nothing merged (dry run)")
}

func TestMergeStackRefusesBeforeMerging(t *testing.T) {
	cases := map[string]struct {
		arrange func(*fakeForge, *fakeHistory)
		line    string
	}{
		"red pipeline":   {func(f *fakeForge, _ *fakeHistory) { f.checks[2] = domain.ChecksFail }, "CHECKS FAIL #2"},
		"draft":          {func(f *fakeForge, _ *fakeHistory) { f.prs[3].Draft = true }, "DRAFT #3"},
		"main has moved": {func(_ *fakeForge, h *fakeHistory) { *h.main = "other" }, "OUTDATED #1"},
		"not stacked":    {func(_ *fakeForge, h *fakeHistory) { h.contains["b"] = []string{"main"} }, "NOT STACKED #2"},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			forge, history := threeHigh()
			c.arrange(forge, history)
			out, ok := runStack(forge, history, MergeOptions{Top: "c"})
			if ok || len(forge.calls) != 0 {
				t.Fatalf("want a refusal with no merge: ok=%v calls=%v", ok, forge.calls)
			}
			wantLines(t, out, c.line, "FAIL nothing merged")
		})
	}
}

func TestMergeStackUnknownBase(t *testing.T) {
	forge, history := threeHigh()
	forge.prs[1].Open = false
	out, ok := runStack(forge, history, MergeOptions{Top: "c"})
	if ok || len(forge.calls) != 0 {
		t.Fatalf("ok=%v calls=%v", ok, forge.calls)
	}
	wantLines(t, out, "FAIL #2 targets a, which is neither main nor the branch of an open merge request")
}

func TestMergeStackStopsWhereItResists(t *testing.T) {
	forge, history := threeHigh()
	forge.mergeErr[2] = errors.New("conflicts with its base")
	out, ok := runStack(forge, history, MergeOptions{Top: "c"})
	if ok {
		t.Fatal("want a failure")
	}
	if got := strings.Join(forge.calls, ", "); got != "merge a, retarget b" {
		t.Errorf("calls = %s", got)
	}
	wantLines(t, out, "STOPPED #2: conflicts with its base", "MERGED: #1", "REMAINING: #2 #3", "FAIL stack partly merged")
}

func TestMergeStackTreeDiffers(t *testing.T) {
	forge, history := threeHigh()
	forge.drift = "someone-else"
	out, ok := runStack(forge, history, MergeOptions{Top: "c"})
	if ok {
		t.Fatal("want a failure")
	}
	wantLines(t, out, "TREE DIFFERS main is not the code of #3: drift.txt", "FAIL stack merged, main differs from the top of the stack")
}

func TestMergeStackRedDefaultBranch(t *testing.T) {
	forge, history := threeHigh()
	forge.mainChecks = domain.ChecksFail
	out, ok := runStack(forge, history, MergeOptions{Top: "c"})
	if ok {
		t.Fatal("want a failure")
	}
	wantLines(t, out, "TREE IDENTICAL", "CHECKS FAIL main", "FAIL stack merged, main is red")
}

func TestMergeStackWithoutPipeline(t *testing.T) {
	t.Run("validation command passes", func(t *testing.T) {
		forge, history := threeHigh()
		forge.mainChecks = domain.ChecksNone
		out, ok := runStack(forge, history, MergeOptions{Top: "c", Validate: "mise run check"})
		if !ok || strings.Join(history.validated, ",") != "mise run check@c" {
			t.Fatalf("ok=%v validated=%v\n%s", ok, history.validated, out)
		}
		wantLines(t, out, "VALIDATION PASS mise run check", "PASS")
	})
	t.Run("validation command fails", func(t *testing.T) {
		forge, history := threeHigh()
		forge.mainChecks, history.validErr = domain.ChecksNone, errors.New("exit status 1")
		out, ok := runStack(forge, history, MergeOptions{Top: "c", Validate: "mise run check"})
		if ok {
			t.Fatal("want a failure")
		}
		wantLines(t, out, "VALIDATION FAIL mise run check", "FAIL stack merged, main is red")
	})
	t.Run("nothing to verify with", func(t *testing.T) {
		forge, history := threeHigh()
		forge.mainChecks = domain.ChecksNone
		out, ok := runStack(forge, history, MergeOptions{Top: "c"})
		if ok {
			t.Fatal("want a failure")
		}
		wantLines(t, out, "CHECKS NONE", "FAIL stack merged, main not verified")
	})
}
