package coverage

import "strings"

// Pragmas lists the lines of a Python source that carry a coverage pragma.
func Pragmas(file, source string) []Finding {
	var out []Finding
	for i, l := range strings.Split(source, "\n") {
		if strings.Contains(l, "# pragma: no cover") || strings.Contains(l, "# pragma: no branch") {
			out = append(out, Finding{Kind: "PRAGMA", File: file, Start: i + 1, End: i + 1})
		}
	}
	return out
}
