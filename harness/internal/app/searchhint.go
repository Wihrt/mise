package app

import (
	"path"

	"github.com/Wihrt/mise/harness/internal/domain"
)

// SearchHint reminds the model of semantic search when a text search reads
// like a question about the code. It never refuses a call.
type SearchHint struct {
	parser CommandParser
}

// NewSearchHint builds the policy.
func NewSearchHint(parser CommandParser) *SearchHint { return &SearchHint{parser: parser} }

const searchNote = "This search reads like a question about the code, not like a string to find. " +
	"For \"how does X work\" or \"where is Y handled\", use `semble search \"<question>\"` (agent `semble-search`); " +
	"when it returns nothing useful, come back to grep. Keep grep for exact strings and patterns."

// searchTools are the text search programs, and their flags that take a value.
var (
	searchTools      = []string{"grep", "egrep", "rg", "ag"}
	searchValueFlags = []string{
		"-A", "-B", "-C", "-m", "-f", "-g", "-t", "-T", "-d", "-D",
		"--include", "--exclude", "--exclude-dir", "--glob", "--type", "--type-not", "--max-count", "--max-depth",
		"--after-context", "--before-context", "--context", "--color", "--colour", "--file",
	}
)

// Evaluate implements Policy.
func (p *SearchHint) Evaluate(req domain.Request) domain.Decision {
	switch req.Tool {
	case "Grep":
		if domain.Exploratory(req.Pattern) {
			return domain.Hinted(searchNote)
		}
	case "Bash":
		for _, inv := range p.parser.Parse(req.Command) {
			if domain.Exploratory(searchPattern(inv.Unwrap())) {
				return domain.Hinted(searchNote)
			}
		}
	}
	return domain.Allowed()
}

// searchPattern is what a grep-like invocation looks for, or "" for any other program.
func searchPattern(inv domain.Invocation) string {
	// rtk wraps the search programs: "rtk grep", "rtk proxy rg".
	for path.Base(inv.Name()) == "rtk" || inv.Name() == "proxy" {
		inv = domain.Invocation{Words: inv.Args()}
	}
	found := false
	for _, tool := range searchTools {
		found = found || path.Base(inv.Name()) == tool
	}
	if !found {
		return ""
	}
	if e := inv.FlagValue("-e", "--regexp"); e != "" {
		return e
	}
	if ops := inv.Operands(searchValueFlags...); len(ops) > 0 {
		return ops[0]
	}
	return ""
}
