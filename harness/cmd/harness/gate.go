package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"slices"
	"strings"

	"github.com/Wihrt/mise/harness/internal/adapters/fsys"
	"github.com/Wihrt/mise/harness/internal/adapters/git"
	"github.com/Wihrt/mise/harness/internal/adapters/mise"
	"github.com/Wihrt/mise/harness/internal/app/gate"
	"github.com/Wihrt/mise/harness/internal/app/local"
)

var gateUsage = strings.Join([]string{
	"usage:",
	"  harness gate red   --slug S --test SHA --task T [--repo DIR] [--spec PATH --ac ID,ID] [--allow-compile-red]",
	"  harness gate green --slug S --test SHA --feat SHA --task T [--repo DIR] [--ac-changed ID,ID] [--replaces S,S]",
	"                     [--proofs DIR [--jobs N] [--no-neutral] [--no-minimal] [--since SHA] [--unneeded-ok FILE:START-END]]",
	"                     [--proof-exempt] [--spec PATH]",
	"  harness gate characterize --slug S --test SHA --task T --spec PATH --ac ID,ID --proofs|--mutants DIR [--repo DIR]",
	"                     [--ac-changed ID,ID] [--replaces S,S] [--jobs N]",
	"  harness gate replay --base SHA --head SHA --proofs DIR --task T [--repo DIR] [--accept-lost NAME]",
	"Common: [--format junit|tofu-json] [--report PATH] [--timeout SECONDS]",
	"Last output line is PASS or FAIL. Exit 0 PASS, 1 FAIL, 2 usage or environment error.",
}, "\n")

// csv is a comma list flag; empty items are dropped.
type csv struct {
	values *[]string
	set    *bool
}

func (c csv) String() string { return "" }

func (c csv) Set(v string) error {
	*c.values = nil
	for _, x := range strings.Split(v, ",") {
		if x != "" {
			*c.values = append(*c.values, x)
		}
	}
	if c.set != nil {
		*c.set = true
	}
	return nil
}

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
		return gate.Options{}, errors.New("a mode is required: red, green, characterize or replay")
	}
	o := gate.Options{Mode: args[0]}
	fs := flag.NewFlagSet("harness gate "+o.Mode, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	required := []string{"task"}
	var mutants string
	var acGiven bool
	fs.StringVar(&o.Task, "task", "", "mise task")
	fs.StringVar(&o.Repo, "repo", ".", "repository")
	fs.IntVar(&o.Timeout, "timeout", 540, "seconds per mise run (below the 600 s Bash tool cap)")
	o.Jobs = 1
	switch o.Mode {
	case "replay":
		required = append(required, "base", "head", "proofs")
		fs.StringVar(&o.Base, "base", "", "SHA")
		fs.StringVar(&o.Head, "head", "", "SHA")
		fs.StringVar(&o.Proofs, "proofs", "", "DIR")
		fs.Var(list{&o.AcceptLost}, "accept-lost", "NAME (repeatable)")
	case "red", "green", "characterize":
		required = append(required, "slug", "test")
		fs.StringVar(&o.Slug, "slug", "", "slug")
		fs.StringVar(&o.Test, "test", "", "SHA")
		fs.StringVar(&o.Format, "format", "", "report format")
		fs.StringVar(&o.Report, "report", "", "report path, repo-relative")
		fs.StringVar(&o.Spec, "spec", "", "spec file")
		if o.Mode != "green" {
			fs.Var(csv{&o.AC, &acGiven}, "ac", "comma list of criteria IDs")
		}
		if o.Mode == "red" {
			fs.BoolVar(&o.AllowCompileRed, "allow-compile-red", false, "keep the compile/collect error excuse")
		} else {
			fs.Var(csv{&o.ACChanged, nil}, "ac-changed", "comma list of criteria IDs whose test files may change")
			fs.Var(csv{&o.Replaces, nil}, "replaces", "comma list of slugs that may be changed or removed")
			fs.IntVar(&o.Jobs, "jobs", 1, "judge N proofs and N removed blocks at a time")
			fs.StringVar(&o.Proofs, "proofs", "", "DIR")
		}
		if o.Mode == "characterize" {
			required = append(required, "spec", "ac")
			fs.StringVar(&mutants, "mutants", "", "synonym of --proofs")
		}
		if o.Mode == "green" {
			required = append(required, "feat")
			fs.StringVar(&o.Feat, "feat", "", "SHA")
			fs.Var(list{&o.UnneededOK}, "unneeded-ok", "FILE:START-END (repeatable)")
			fs.StringVar(&o.Since, "since", "", "SHA")
			fs.BoolVar(&o.ProofExempt, "proof-exempt", false, "exempt from proofs")
			fs.BoolVar(&o.NoNeutral, "no-neutral", false, "skip the neutral change")
			fs.BoolVar(&o.NoMinimal, "no-minimal", false, "skip the minimal-code proof")
		}
	default:
		return o, fmt.Errorf("unknown mode %q", o.Mode)
	}
	if err := fs.Parse(args[1:]); err != nil {
		return o, err
	}
	if fs.NArg() > 0 {
		return o, fmt.Errorf("unrecognized arguments: %s", strings.Join(fs.Args(), " "))
	}
	given := map[string]bool{}
	fs.Visit(func(f *flag.Flag) { given[f.Name] = true })
	for _, name := range required {
		if !given[name] {
			return o, fmt.Errorf("--%s is required", name)
		}
	}
	if o.Mode == "characterize" {
		if given["proofs"] == given["mutants"] {
			return o, errors.New("one of --proofs and --mutants is required, not both")
		}
		if given["mutants"] {
			o.Proofs = mutants
		}
	}
	if o.Mode == "red" && (o.Spec != "") != (len(o.AC) > 0) {
		return o, errors.New("--spec and --ac go together")
	}
	if o.Format != "" && !slices.Contains(gate.Formats, o.Format) {
		return o, fmt.Errorf("--format: invalid choice %q", o.Format)
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
	return gate.New(git.NewRunner(), fsys.New(), mise.NewRunner(), mise.TaskFile{}, stdout).Run(o)
}

var localUsage = strings.Join([]string{
	"usage:",
	"  harness local forge --repo DIR   print the forge CLI (gh, glab) for the host of DIR's origin",
	"  harness local cas-reels ID       print the lines of cas-reels.md that name ID",
	"The local directory is $CLAUDE_LOCAL_DIR, default ~/.claude/local.",
}, "\n")

func runLocal(args []string, stdout, stderr io.Writer, getenv func(string) string) int {
	dir := getenv("CLAUDE_LOCAL_DIR")
	if dir == "" {
		dir = getenv("HOME") + "/.claude/local"
	}
	l := local.New(dir, git.NewRunner(), fsys.New(), stdout)
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
