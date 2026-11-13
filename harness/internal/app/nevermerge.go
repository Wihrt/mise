package app

import (
	"path"
	"regexp"
	"strings"

	"github.com/Wihrt/mise/harness/internal/domain"
)

// Branches names the branches of the repository in a directory; "" when unknown.
type Branches interface {
	Exists(dir string) bool
	Default(dir string) string
	Current(dir string) string
}

// NeverMerge refuses merges, pushes to the default branch and force pushes
// without a lease. The one way to merge is `harness merge-stack`, which it
// hands to the user: their go is the authorisation, for that stack only.
type NeverMerge struct {
	parser   CommandParser
	branches Branches
}

// NewNeverMerge builds the policy.
func NewNeverMerge(parser CommandParser, branches Branches) *NeverMerge {
	return &NeverMerge{parser: parser, branches: branches}
}

var (
	forgeFlags = []string{"-R", "--repo"}
	shortForce = regexp.MustCompile(`^-[A-Za-z]*f[A-Za-z]*$`)
)

// Evaluate implements Policy.
func (p *NeverMerge) Evaluate(req domain.Request) domain.Decision {
	if req.Tool != "Bash" {
		return domain.Allowed()
	}
	for _, inv := range p.parser.Parse(req.Command) {
		inv = inv.Unwrap()
		var d domain.Decision
		switch path.Base(inv.Name()) {
		case "harness":
			d = mergeStack(inv)
		case "glab":
			d = forgeMerge(inv, "mr", "glab mr merge is blocked. Stop at a green pipeline and hand over the MR link; when the user asks for the merge, run `harness merge-stack <top MR>`.")
		case "gh":
			d = forgeMerge(inv, "pr", "gh pr merge is blocked. Stop at a green pipeline and hand over the PR link; when the user asks for the merge, run `harness merge-stack <top PR>`.")
		case "git":
			d = p.git(inv, req.Cwd)
		}
		if d.Verdict != domain.Allow {
			return d
		}
	}
	return domain.Allowed()
}

// mergeStack asks the user before a stack is merged; a dry run merges nothing.
func mergeStack(inv domain.Invocation) domain.Decision {
	ops := inv.Operands("--validate", "-validate")
	if len(ops) == 0 || ops[0] != "merge-stack" || inv.WantsHelp() || inv.HasArg("--dry-run", "-dry-run") {
		return domain.Allowed()
	}
	target := "this merge request"
	if len(ops) > 1 {
		target = ops[1]
	}
	return domain.Asked("Merge " + target + " and every merge request below it into the default branch? Your go covers this stack only.")
}

func forgeMerge(inv domain.Invocation, noun, reason string) domain.Decision {
	ops := inv.Operands(forgeFlags...)
	if len(ops) >= 2 && ops[0] == noun && ops[1] == "merge" && !inv.WantsHelp() {
		return domain.Denied(reason)
	}
	return domain.Allowed()
}

func (p *NeverMerge) git(inv domain.Invocation, cwd string) domain.Decision {
	ops := inv.Operands(gitFlags...)
	if len(ops) == 0 {
		return domain.Allowed()
	}
	dir := gitDir(inv, cwd)
	switch ops[0] {
	case "push":
		if forces(inv) {
			return domain.Denied("Force pushes are blocked. Use git push --force-with-lease instead.")
		}
		if !p.branches.Exists(dir) {
			return domain.Allowed()
		}
		def := p.branches.Default(dir)
		if def == "" {
			return domain.Allowed()
		}
		targets := ops[1:]
		onDefault := p.branches.Current(dir) == def && len(targets) <= 1
		if onDefault || pushesTo(targets, def) {
			return domain.Denied("Pushing to the default branch (" + def + ") is blocked. Push a branch and go through a merge request.")
		}
	case "merge":
		if !p.branches.Exists(dir) {
			return domain.Allowed()
		}
		cur, def := p.branches.Current(dir), p.branches.Default(dir)
		if cur != "" && cur == def {
			return domain.Denied("git merge on the default branch (" + def + ") is blocked. Work on a branch and go through a merge request.")
		}
	}
	return domain.Allowed()
}

// forces reports a force push that carries no lease.
func forces(inv domain.Invocation) bool {
	force, lease := false, false
	for _, a := range inv.Args() {
		if a == "--force" || shortForce.MatchString(a) {
			force = true
		}
		if strings.HasPrefix(a, "--force-with-lease") {
			lease = true
		}
	}
	return force && !lease
}

// pushesTo reports whether one of the refspecs writes the branch.
func pushesTo(refspecs []string, branch string) bool {
	for _, r := range refspecs {
		dst := strings.TrimPrefix(r, "+")
		if src, d, ok := strings.Cut(r, ":"); ok {
			if strings.Contains(d, ":") || strings.Contains(src, ":") {
				continue
			}
			dst = d
		}
		if strings.TrimPrefix(dst, "refs/heads/") == branch {
			return true
		}
	}
	return false
}
