package coverage

import (
	"regexp"
	"strings"
)

// pragma holds the two default exclusion patterns of coverage.py 7.14.0
// (coverage/config.py): no cover and no branch.
var pragma = regexp.MustCompile(`#\s*(pragma|PRAGMA)[:\s]?\s*(no|NO)\s*(cover|COVER|branch|BRANCH)`)

// Pragmas lists the lines of a Python source that carry a coverage pragma.
func Pragmas(file, source string) []Finding {
	var out []Finding
	for i, l := range strings.Split(source, "\n") {
		if pragma.MatchString(l) {
			out = append(out, Finding{Kind: "PRAGMA", File: file, Start: i + 1, End: i + 1})
		}
	}
	return out
}
