package domain

import (
	"regexp"
	"slices"
	"strings"
)

// Invocation is one simple command of a shell line: a program and its words.
type Invocation struct {
	Words []string
}

var (
	wrappers   = []string{"command", "env", "sudo", "exec"}
	assignment = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*=`)
)

// Unwrap drops the leading wrappers (command, env, sudo, exec) and the
// NAME=value words around them, so that the program really run comes first.
func (i Invocation) Unwrap() Invocation {
	w := i.Words
	for len(w) > 0 && (slices.Contains(wrappers, w[0]) || assignment.MatchString(w[0])) {
		w = w[1:]
	}
	return Invocation{Words: w}
}

// Name is the program, or "" for an empty invocation.
func (i Invocation) Name() string {
	if len(i.Words) == 0 {
		return ""
	}
	return i.Words[0]
}

// Args are the words after the program.
func (i Invocation) Args() []string {
	if len(i.Words) == 0 {
		return nil
	}
	return i.Words[1:]
}

// HasArg reports whether one of the words is among the arguments.
func (i Invocation) HasArg(words ...string) bool {
	return slices.ContainsFunc(i.Args(), func(a string) bool { return slices.Contains(words, a) })
}

// WantsHelp reports whether the invocation only asks for help.
func (i Invocation) WantsHelp() bool { return i.HasArg("--help", "-h") }

// Operands are the arguments that are not flags. A flag listed in valueFlags
// takes the next word as its value; "-x=value" carries its own.
func (i Invocation) Operands(valueFlags ...string) []string {
	var ops []string
	args := i.Args()
	for n := 0; n < len(args); n++ {
		a := args[n]
		switch {
		case !strings.HasPrefix(a, "-"):
			ops = append(ops, a)
		case !strings.Contains(a, "=") && slices.Contains(valueFlags, a):
			n++
		}
	}
	return ops
}

// FlagValue is the value of the first of the named flags, written "--flag value"
// or "--flag=value", or "" when none is set.
func (i Invocation) FlagValue(names ...string) string {
	args := i.Args()
	for n, a := range args {
		for _, name := range names {
			if a == name && n+1 < len(args) {
				return args[n+1]
			}
			if v, ok := strings.CutPrefix(a, name+"="); ok {
				return v
			}
		}
	}
	return ""
}
