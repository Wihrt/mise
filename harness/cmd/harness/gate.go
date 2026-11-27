package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/Wihrt/mise/harness/internal/adapters/fsys"
	"github.com/Wihrt/mise/harness/internal/adapters/git"
	"github.com/Wihrt/mise/harness/internal/adapters/mise"
	"github.com/Wihrt/mise/harness/internal/app/gate"
	"github.com/Wihrt/mise/harness/internal/app/local"
)

var gateUsage = strings.Join([]string{
	"usage:",
	"  harness gate coverage --task T --coverage REPORT [--coverage REPORT ...] [--repo DIR] [--baseline] [--timeout SECONDS]",
	"Last output line is PASS or FAIL. Exit 0 PASS, 1 FAIL, 2 usage or environment error.",
}, "\n")

// list is a repeatable flag.
type list struct{ values *[]string }

func (l list) String() string { return "" }

func (l list) Set(v string) error {
	*l.values = append(*l.values, v)
	return nil
}

// parseGate reads the command line of `harness gate`.
func parseGate(args []string) (gate.Options, error) {
	if len(args) == 0 {
		return gate.Options{}, errors.New("a mode is required: coverage")
	}
	var o gate.Options
	if args[0] != "coverage" {
		return o, fmt.Errorf("unknown mode %q", args[0])
	}
	fs := flag.NewFlagSet("harness gate coverage", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.StringVar(&o.Task, "task", "", "mise task")
	fs.StringVar(&o.Repo, "repo", ".", "repository")
	fs.IntVar(&o.Timeout, "timeout", 540, "seconds per mise run (below the 600 s Bash tool cap)")
	fs.Var(list{&o.Coverage}, "coverage", "REPORT (repeatable)")
	fs.BoolVar(&o.Baseline, "baseline", false, "write the findings to COVERAGE.md as initial debt")
	if err := fs.Parse(args[1:]); err != nil {
		return o, err
	}
	if fs.NArg() > 0 {
		return o, fmt.Errorf("unrecognized arguments: %s", strings.Join(fs.Args(), " "))
	}
	given := map[string]bool{}
	fs.Visit(func(f *flag.Flag) { given[f.Name] = true })
	for _, name := range []string{"task", "coverage"} {
		if !given[name] {
			return o, fmt.Errorf("--%s is required", name)
		}
	}
	return o, nil
}

func runGate(args []string, stdout, stderr io.Writer) int {
	o, err := parseGate(args)
	if errors.Is(err, flag.ErrHelp) {
		_, _ = fmt.Fprintln(stdout, gateUsage)
		return 0
	}
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "%s\nharness gate: %v\n", gateUsage, err)
		return 2
	}
	return gate.New(git.NewRunner(), fsys.New(), mise.NewRunner(), stdout).Run(o)
}

var localUsage = strings.Join([]string{
	"usage:",
	"  harness local forge --repo DIR   print the forge CLI (gh, glab) for the host of DIR's origin",
	"  harness local cas-reels ID       print the lines of cas-reels.md that name ID",
	"The local directory is $CLAUDE_LOCAL_DIR, default ~/.claude/local.",
}, "\n")

func runLocal(args []string, stdout, stderr io.Writer, getenv func(string) string) int {
	l := local.New(localDir(getenv), git.NewRunner(), fsys.New(), stdout)
	command, rest := "", []string(nil)
	if len(args) > 0 {
		command, rest = args[0], args[1:]
	}
	if repo, ok := strings.CutPrefix(strings.Join(rest, "="), "--repo="); ok && command == "forge" && len(rest) <= 2 && repo != "" {
		code, err := l.Forge(repo)
		if err != nil {
			_, _ = fmt.Fprintf(stderr, "harness local: %v\n", err)
		}
		return code
	}
	if command == "cas-reels" && len(rest) == 1 {
		return l.CasReels(rest[0])
	}
	_, _ = fmt.Fprintln(stderr, localUsage)
	return 2
}

// localDir is the directory of the machine-local files of the harness.
func localDir(getenv func(string) string) string {
	if dir := getenv("CLAUDE_LOCAL_DIR"); dir != "" {
		return dir
	}
	return getenv("HOME") + "/.claude/local"
}
