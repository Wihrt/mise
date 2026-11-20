package domain

import "time"

// Timing bounds the waits on the forge.
type Timing struct {
	Interval time.Duration // between two polls
	Grace    time.Duration // how long a commit may stay without any pipeline before it is said to have none
	Timeout  time.Duration // longest wait for pipelines to end
}

// PullRequest is one merge request of a forge, as far as merging a stack needs it.
type PullRequest struct {
	Number int
	URL    string
	Head   string // branch the request comes from
	Base   string // branch it targets
	SHA    string // commit at the tip of Head
	State  string // as the forge says it, lower case
	Open   bool
	Draft  bool
	// Conflicts and MergeStatus are what a forge says of mergeability; MergeStatus is
	// empty when it does not say (GitHub), otherwise "mergeable" or the obstacle.
	Conflicts   bool
	MergeStatus string
}

// Checks is the outcome of the pipelines of a commit.
type Checks int

const (
	// ChecksPass means every pipeline ended well.
	ChecksPass Checks = iota
	// ChecksFail means one pipeline failed or never ended.
	ChecksFail
	// ChecksNone means the commit has no pipeline at all.
	ChecksNone
)

// Stack is a chain of merge requests, bottom first: the first targets the
// target of the stack, each next one targets the head of the one below.
type Stack []PullRequest

// Top is the last request of the stack, the one whose code is the outcome.
func (s Stack) Top() PullRequest { return s[len(s)-1] }
