// Package app holds the hook policies: one use case per hook.
package app

import "github.com/Wihrt/mise/harness/internal/domain"

// Policy decides on one tool call.
type Policy interface {
	Evaluate(req domain.Request) domain.Decision
}

// CommandParser splits a shell line into the simple commands it runs.
type CommandParser interface {
	Parse(command string) []domain.Invocation
}

// gitFlags are the git options that take the next word as their value.
var gitFlags = []string{"-c", "-C"}

// gitCalls returns, unwrapped, the git invocations of a Bash request whose
// subcommand is one of subs.
func gitCalls(parser CommandParser, req domain.Request, subs ...string) []domain.Invocation {
	if req.Tool != "Bash" {
		return nil
	}
	var calls []domain.Invocation
	for _, inv := range parser.Parse(req.Command) {
		inv = inv.Unwrap()
		if inv.Name() != "git" {
			continue
		}
		ops := inv.Operands(gitFlags...)
		if len(ops) == 0 {
			continue
		}
		for _, sub := range subs {
			if ops[0] == sub {
				calls = append(calls, inv)
			}
		}
	}
	return calls
}

// gitDir is the directory a git invocation works in: its -C, else the call's cwd.
func gitDir(inv domain.Invocation, cwd string) string {
	c := inv.FlagValue("-C")
	switch {
	case c == "":
		return cwd
	case len(c) > 0 && c[0] == '/':
		return c
	default:
		return cwd + "/" + c
	}
}
