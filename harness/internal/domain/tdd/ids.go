package tdd

import (
	"regexp"
	"strings"
)

var (
	testPrefix   = regexp.MustCompile(`^(Test|test_|test)`)
	quoteBracket = regexp.MustCompile(`^"?\[?`)
	prefixDigits = regexp.MustCompile(`^([A-Z]+)_([0-9]{3})`)
)

// idAt returns the criterion ID at the very start of s, "" when there is none.
// An ID is capital letters, an optional dash and three digits or more, not
// followed by a letter or a digit (TestSHA256Sum and TestHTTP2Handler are
// legacy names, not IDs).
func idAt(s string) string {
	i := 0
	for i < len(s) && s[i] >= 'A' && s[i] <= 'Z' {
		i++
	}
	if i == 0 {
		return ""
	}
	j := i
	if j < len(s) && s[j] == '-' {
		j++
	}
	k := j
	for k < len(s) && s[k] >= '0' && s[k] <= '9' {
		k++
	}
	if k-j < 3 {
		return ""
	}
	if k < len(s) {
		c := s[k]
		if (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') {
			return ""
		}
	}
	return s[:k]
}

// Norm strips a leading Test, test_ or test, then every dash.
func Norm(s string) string {
	return strings.ReplaceAll(testPrefix.ReplaceAllString(s, ""), "-", "")
}

// Candidate is the normalized ID at the very start of a reported test name, "" when none.
func Candidate(name string) string {
	s := testPrefix.ReplaceAllString(name, "")
	s = quoteBracket.ReplaceAllString(s, "")
	s = prefixDigits.ReplaceAllString(s, "$1$2")
	return strings.ReplaceAll(idAt(s), "-", "")
}

// headingID is the ID of a `### ID` heading, "" for any other line.
func headingID(line string) string {
	if !strings.HasPrefix(line, "### ") {
		return ""
	}
	return idAt(line[4:])
}
