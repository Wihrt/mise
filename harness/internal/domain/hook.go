// Package domain holds what a hook decides on, free of any I/O.
package domain

// Request is one tool call Claude Code is about to make.
type Request struct {
	Tool         string // Bash, Edit, Write, NotebookEdit, Agent, Task...
	Command      string // Bash
	Path         string // Edit, Write, NotebookEdit
	Cwd          string
	SubagentType string // Agent, Task
	Model        string // Agent, Task
	Description  string // Agent, Task
}

// EditsFile reports whether the tool writes a file.
func (r Request) EditsFile() bool {
	return r.Tool == "Edit" || r.Tool == "Write" || r.Tool == "NotebookEdit"
}

// Verdict is what a hook answers.
type Verdict int

const (
	// Allow lets the call through.
	Allow Verdict = iota
	// Deny refuses the call.
	Deny
	// Ask hands the call to the user for a go.
	Ask
)

// Decision is a verdict and, unless it allows, its reason.
type Decision struct {
	Verdict Verdict
	Reason  string
}

// Allowed is the decision that lets a call through.
func Allowed() Decision { return Decision{Verdict: Allow} }

// Denied refuses a call for the given reason.
func Denied(reason string) Decision { return Decision{Verdict: Deny, Reason: reason} }

// Asked hands a call to the user with the given reason.
func Asked(reason string) Decision { return Decision{Verdict: Ask, Reason: reason} }
