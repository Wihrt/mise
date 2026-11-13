// Command harness runs the Claude Code hooks, the TDD gates and the local-file
// lookups of the harness, and merges stacks of merge requests.
//
// Usage:
//
//	harness hook <name> [argument]                             hook input on stdin
//	harness gate <red|green|characterize|replay> ...
//	harness local <forge|cas-reels> ...
//	harness merge-stack [--dry-run] [--validate <command>] <top merge request>
//
// Configuration comes from the environment: HOME, KUBECONFIG,
// CLAUDE_CODE_SUBAGENT_MODEL, CLAUDE_AGENTS_DIR (default <HOME>/.claude/agents),
// CLAUDE_LOCAL_DIR (default <HOME>/.claude/local), and for merge-stack
// HARNESS_POLL_INTERVAL (15s), HARNESS_CHECKS_GRACE (60s) and
// HARNESS_CHECKS_TIMEOUT (30m), written as Go durations.
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/Wihrt/mise/harness/internal/adapters/agentfile"
	"github.com/Wihrt/mise/harness/internal/adapters/claudecode"
	"github.com/Wihrt/mise/harness/internal/adapters/git"
	"github.com/Wihrt/mise/harness/internal/adapters/github"
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

const usage = "usage: harness hook <name> [argument]\n" +
	"       harness gate <red|green|characterize|replay> ...\n" +
	"       harness local <forge|cas-reels> ...\n" +
	"       harness merge-stack [--dry-run] [--validate <command>] <top merge request>"

// duration reads a Go duration from the environment, or returns the default.
func duration(getenv func(string) string, name string, def time.Duration) time.Duration {
	if d, err := time.ParseDuration(getenv(name)); err == nil {
		return d
	}
	return def
}

// mergeStack runs the merge-stack command in the current directory.
func mergeStack(args []string, stdout, stderr io.Writer, getenv func(string) string) int {
	flags := flag.NewFlagSet("merge-stack", flag.ContinueOnError)
	flags.SetOutput(stderr)
	dryRun := flags.Bool("dry-run", false, "check that the stack can be merged, merge nothing")
	validate := flags.String("validate", "", "command run on the merged default branch when it has no pipeline")
	if err := flags.Parse(args); err != nil || flags.NArg() != 1 {
		_, _ = fmt.Fprintln(stderr, usage)
		return 2
	}
	dir, err := os.Getwd()
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "harness: %v\n", err)
		return 1
	}
	clone := git.NewClone(dir)
	forge := github.NewForge(dir, github.Timing{
		Interval: duration(getenv, "HARNESS_POLL_INTERVAL", 15*time.Second),
		Grace:    duration(getenv, "HARNESS_CHECKS_GRACE", time.Minute),
		Timeout:  duration(getenv, "HARNESS_CHECKS_TIMEOUT", 30*time.Minute),
	})
	opt := app.MergeOptions{Top: flags.Arg(0), DryRun: *dryRun, Validate: *validate}
	if app.NewMergeStack(forge, clone, clone, stdout).Run(opt) {
		return 0
	}
	return 1
}

func run(args []string, stdin io.Reader, stdout, stderr io.Writer, getenv func(string) string) int {
	if len(args) > 0 && args[0] == "gate" {
		return runGate(args[1:], stdout, stderr)
	}
	if len(args) > 0 && args[0] == "local" {
		return runLocal(args[1:], stdout, stderr, getenv)
	}
	if len(args) > 0 && args[0] == "merge-stack" {
		return mergeStack(args[1:], stdout, stderr, getenv)
	}
	if len(args) < 2 || args[0] != "hook" {
		_, _ = fmt.Fprintln(stderr, usage)
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
