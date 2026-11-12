package app

import (
	"path/filepath"

	"github.com/Wihrt/mise/harness/internal/domain"
)

// Checkout tells where a directory stands in a git repository.
type Checkout interface {
	Exists(dir string) bool
	InMainCheckout(dir string) bool
	Ignored(dir, path string) bool
}

// RequireWorktree refuses edits and commits in the main checkout of a repository.
type RequireWorktree struct {
	parser   CommandParser
	checkout Checkout
}

// NewRequireWorktree builds the policy.
func NewRequireWorktree(parser CommandParser, checkout Checkout) *RequireWorktree {
	return &RequireWorktree{parser: parser, checkout: checkout}
}

// Evaluate implements Policy.
func (p *RequireWorktree) Evaluate(req domain.Request) domain.Decision {
	if req.EditsFile() {
		return p.edit(req.Path)
	}
	for _, inv := range gitCalls(p.parser, req, "commit") {
		dir := gitDir(inv, req.Cwd)
		if p.checkout.Exists(dir) && p.checkout.InMainCheckout(dir) {
			return domain.Denied("Commits from the main checkout are blocked: " + dir +
				". Create or enter a git worktree first, then commit there.")
		}
	}
	return domain.Allowed()
}

func (p *RequireWorktree) edit(path string) domain.Decision {
	if path == "" {
		return domain.Allowed()
	}
	dir := filepath.Dir(path)
	for !p.checkout.Exists(dir) && dir != "/" && dir != "." {
		dir = filepath.Dir(dir)
	}
	if !p.checkout.InMainCheckout(dir) || p.checkout.Ignored(dir, path) {
		return domain.Allowed()
	}
	return domain.Denied("Edits in the main checkout are blocked: " + path +
		". Create or enter a git worktree first, then edit the file there.")
}
