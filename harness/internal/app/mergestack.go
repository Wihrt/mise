package app

import (
	"fmt"
	"io"
	"strings"

	"github.com/Wihrt/mise/harness/internal/domain"
)

// Forge is the merge request side of the repository host.
type Forge interface {
	DefaultBranch() (string, error)
	// Get reads a request by number, URL or head branch.
	Get(ref string) (domain.PullRequest, error)
	// ByHead finds the open request that comes from a branch.
	ByHead(branch string) (domain.PullRequest, bool, error)
	Retarget(number int, base string) error
	// Checks waits for the pipelines of a request to end.
	Checks(number int) (domain.Checks, error)
	// Merge merges a request with a merge commit, only if its head is still sha.
	Merge(number int, sha string) error
	// CommitChecks waits for the pipelines of a commit of the default branch to end.
	CommitChecks(sha string) (domain.Checks, error)
}

// History reads the commits of the local clone.
type History interface {
	Fetch() error
	// Remote is the commit of a branch on the remote, after Fetch.
	Remote(branch string) (string, error)
	Tree(rev string) (string, error)
	IsAncestor(ancestor, descendant string) bool
	// ChangedFiles lists the files that differ between two commits.
	ChangedFiles(from, to string) ([]string, error)
}

// Validator runs a validation command on a commit, outside the working tree.
type Validator interface {
	Run(sha, command string) error
}

// MergeStack merges a stack of merge requests, bottom first, then proves that
// the default branch holds the code of the top of the stack and that it passes.
type MergeStack struct {
	forge     Forge
	history   History
	validator Validator
	out       io.Writer
}

// NewMergeStack builds the use case.
func NewMergeStack(forge Forge, history History, validator Validator, out io.Writer) *MergeStack {
	return &MergeStack{forge: forge, history: history, validator: validator, out: out}
}

// MergeOptions tunes one run.
type MergeOptions struct {
	Top      string // number, URL or branch of the top request
	DryRun   bool   // check, merge nothing
	Validate string // command run on the merged default branch when it has no pipeline
}

func (m *MergeStack) say(format string, args ...any) {
	_, _ = fmt.Fprintf(m.out, format+"\n", args...)
}

// Run reports whether the stack was merged and proven; it stops at the first doubt.
func (m *MergeStack) Run(opt MergeOptions) bool {
	def, stack, err := m.resolve(opt.Top)
	if err != nil {
		m.say("FAIL %v", err)
		return false
	}
	m.say("STACK %d: %s", len(stack), describe(stack))
	if !m.ready(def, stack) {
		m.say("FAIL nothing merged")
		return false
	}
	if opt.DryRun {
		m.say("READY")
		m.say("PASS nothing merged (dry run)")
		return true
	}
	if !m.merge(def, stack) {
		return false
	}
	return m.prove(def, stack.Top(), opt.Validate)
}

// resolve walks down from the top request to the default branch.
func (m *MergeStack) resolve(top string) (string, domain.Stack, error) {
	def, err := m.forge.DefaultBranch()
	if err != nil {
		return "", nil, fmt.Errorf("default branch: %w", err)
	}
	pr, err := m.forge.Get(top)
	if err != nil {
		return "", nil, fmt.Errorf("merge request %s: %w", top, err)
	}
	stack := domain.Stack{pr}
	seen := map[string]bool{pr.Head: true}
	for pr.Base != def {
		below, found, err := m.forge.ByHead(pr.Base)
		if err != nil {
			return "", nil, fmt.Errorf("merge request of %s: %w", pr.Base, err)
		}
		if !found {
			return "", nil, fmt.Errorf("#%d targets %s, which is neither %s nor the branch of an open merge request", pr.Number, pr.Base, def)
		}
		if seen[below.Head] {
			return "", nil, fmt.Errorf("the merge requests form a loop at %s", below.Head)
		}
		seen[below.Head] = true
		stack = append(domain.Stack{below}, stack...)
		pr = below
	}
	return def, stack, nil
}

func describe(stack domain.Stack) string {
	parts := make([]string, len(stack))
	for i, pr := range stack {
		parts[i] = fmt.Sprintf("#%d %s", pr.Number, pr.Head)
	}
	return strings.Join(parts, " -> ")
}

// ready checks everything that can be checked before the first merge, and
// names every obstacle rather than the first one.
func (m *MergeStack) ready(def string, stack domain.Stack) bool {
	if err := m.history.Fetch(); err != nil {
		m.say("FETCH FAILED %v", err)
		return false
	}
	base, err := m.history.Remote(def)
	if err != nil {
		m.say("FETCH FAILED %v", err)
		return false
	}
	ok := true
	below := base
	for _, pr := range stack {
		switch {
		case !pr.Open:
			m.say("NOT OPEN #%d", pr.Number)
			ok = false
		case pr.Draft:
			m.say("DRAFT #%d", pr.Number)
			ok = false
		}
		if !m.history.IsAncestor(below, pr.SHA) {
			if below == base {
				m.say("OUTDATED #%d: %s is not in %s; rebase the stack on %s, then ask again", pr.Number, def, pr.Head, def)
			} else {
				m.say("NOT STACKED #%d: %s does not contain the branch below it", pr.Number, pr.Head)
			}
			ok = false
		}
		below = pr.SHA
	}
	if !ok {
		return false
	}
	for _, pr := range stack {
		checks, err := m.forge.Checks(pr.Number)
		switch {
		case err != nil:
			m.say("CHECKS UNKNOWN #%d: %v", pr.Number, err)
			ok = false
		case checks == domain.ChecksFail:
			m.say("CHECKS FAIL #%d", pr.Number)
			ok = false
		}
	}
	return ok
}

// merge merges the requests bottom first and stops at the first that resists.
func (m *MergeStack) merge(def string, stack domain.Stack) bool {
	for i, pr := range stack {
		err := error(nil)
		if pr.Base != def {
			err = m.forge.Retarget(pr.Number, def)
		}
		if err == nil {
			err = m.forge.Merge(pr.Number, pr.SHA)
		}
		if err != nil {
			m.say("STOPPED #%d: %v", pr.Number, err)
			m.say("MERGED:%s", numbers(stack[:i]))
			m.say("REMAINING:%s", numbers(stack[i:]))
			m.say("FAIL stack partly merged")
			return false
		}
		m.say("MERGED #%d %s", pr.Number, pr.URL)
	}
	return true
}

func numbers(stack domain.Stack) string {
	if len(stack) == 0 {
		return " none"
	}
	var b strings.Builder
	for _, pr := range stack {
		_, _ = fmt.Fprintf(&b, " #%d", pr.Number)
	}
	return b.String()
}

// prove checks that the default branch is the code that was reviewed, and that it passes.
func (m *MergeStack) prove(def string, top domain.PullRequest, validate string) bool {
	if err := m.history.Fetch(); err != nil {
		m.say("FAIL stack merged, not proven: %v", err)
		return false
	}
	tip, err := m.history.Remote(def)
	if err != nil {
		m.say("FAIL stack merged, not proven: %v", err)
		return false
	}
	want, err1 := m.history.Tree(top.SHA)
	got, err2 := m.history.Tree(tip)
	if err1 != nil || err2 != nil {
		m.say("FAIL stack merged, not proven: cannot read the trees")
		return false
	}
	if want != got {
		files, _ := m.history.ChangedFiles(top.SHA, tip)
		m.say("TREE DIFFERS %s is not the code of #%d: %s", def, top.Number, strings.Join(files, " "))
		m.say("FAIL stack merged, %s differs from the top of the stack", def)
		return false
	}
	m.say("TREE IDENTICAL %s is the code of #%d", def, top.Number)

	checks, err := m.forge.CommitChecks(tip)
	switch {
	case err != nil:
		m.say("CHECKS UNKNOWN %s: %v", def, err)
		m.say("FAIL stack merged, %s not verified", def)
		return false
	case checks == domain.ChecksFail:
		m.say("CHECKS FAIL %s", def)
		m.say("FAIL stack merged, %s is red", def)
		return false
	case checks == domain.ChecksPass:
		m.say("CHECKS PASS %s", def)
	case validate == "":
		m.say("CHECKS NONE %s has no pipeline and no --validate command was given", def)
		m.say("FAIL stack merged, %s not verified", def)
		return false
	default:
		if err := m.validator.Run(tip, validate); err != nil {
			m.say("VALIDATION FAIL %s: %v", validate, err)
			m.say("FAIL stack merged, %s is red", def)
			return false
		}
		m.say("VALIDATION PASS %s", validate)
	}
	m.say("PASS")
	return true
}
