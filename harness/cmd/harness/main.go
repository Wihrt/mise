// Command harness runs the Claude Code hooks of the harness.
//
// Usage: harness hook <name> [argument], with the hook input on stdin.
// Configuration comes from the environment: HOME, KUBECONFIG,
// CLAUDE_CODE_SUBAGENT_MODEL, and CLAUDE_AGENTS_DIR (default <HOME>/.claude/agents).
package main

import (
	"fmt"
	"io"
	"os"

	"github.com/Wihrt/mise/harness/internal/adapters/agentfile"
	"github.com/Wihrt/mise/harness/internal/adapters/claudecode"
	"github.com/Wihrt/mise/harness/internal/adapters/git"
	"github.com/Wihrt/mise/harness/internal/adapters/kubeconfig"
	"github.com/Wihrt/mise/harness/internal/adapters/shell"
	"github.com/Wihrt/mise/harness/internal/app"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr, os.Getenv))
}

// policy builds the named hook; the composition root of the program.
func policy(name, arg string, getenv func(string) string) app.Policy {
	home := getenv("HOME")
	agents := getenv("CLAUDE_AGENTS_DIR")
	if agents == "" {
		agents = home + "/.claude/agents"
	}
	parser, repo := shell.NewParser(), git.NewRepo()
	switch name {
	case "require-worktree":
		return app.NewRequireWorktree(parser, repo)
	case "never-merge":
		return app.NewNeverMerge(parser, repo)
	case "guard-subagent-model":
		return app.NewSubagentModel(agentfile.NewDir(agents), getenv("CLAUDE_CODE_SUBAGENT_MODEL"))
	case "search-hint":
		return app.NewSearchHint(parser)
	case "agent-guard":
		return app.NewAgentGuard(parser, arg)
	case "cluster-guard":
		return app.NewClusterGuard(parser, kubeconfig.NewFile(getenv("KUBECONFIG"), home))
	}
	return nil
}

func run(args []string, stdin io.Reader, stdout, stderr io.Writer, getenv func(string) string) int {
	if len(args) < 2 || args[0] != "hook" {
		_, _ = fmt.Fprintln(stderr, "usage: harness hook <name> [argument]")
		return 2
	}
	arg := ""
	if len(args) > 2 {
		arg = args[2]
	}
	p := policy(args[1], arg, getenv)
	if p == nil {
		_, _ = fmt.Fprintf(stderr, "harness: unknown hook %q\n", args[1])
		return 2
	}
	req, err := claudecode.Read(stdin)
	if err != nil {
		// An input the hook cannot read is not a tool call to judge: let Claude Code decide.
		_, _ = fmt.Fprintf(stderr, "harness: %v\n", err)
		return 0
	}
	if err := claudecode.Write(stdout, p.Evaluate(req)); err != nil {
		_, _ = fmt.Fprintf(stderr, "harness: %v\n", err)
		return 1
	}
	return 0
}
