package tdd

import (
	"regexp"
	"strings"
)

var (
	namePos       = regexp.MustCompile(`^\s*(?:func Test|def test_?|- ?it:\s*"?\[?|it:\s*"?\[?|run "|@test ")`)
	namePosSelect = regexp.MustCompile(`^\s*(func Test|def test|- ?it:|it:|run "|@test ")`)
	// Bruno: test("[ID] what", ...) inside a tests {} block.
	bruNamePos = regexp.MustCompile("^\\s*test\\(\\s*[\"'`]\\[?")
)

// Entry is one test found in a test file.
type Entry struct {
	File string
	Line int
	Cand string // normalized ID of the test name, "" when it carries none
	Body string
}

// Scan finds the tests of a file at name positions only. It returns them with
// the preamble of the file: what precedes its first test.
func Scan(file, content string) ([]Entry, string) {
	lines := Lines(content)
	sel, name := namePosSelect, namePos
	if strings.HasSuffix(file, ".bru") {
		sel, name = bruNamePos, bruNamePos
	}
	var idx []int
	for i, ln := range lines {
		if sel.MatchString(ln) {
			idx = append(idx, i)
		}
	}
	if len(idx) == 0 {
		return nil, strings.Join(lines, "\n")
	}
	var entries []Entry
	for n, i := range idx {
		end := len(lines)
		if n+1 < len(idx) {
			end = idx[n+1]
		}
		for j := i; j < end; j++ { // the next YAML document belongs to the neighbour
			if lines[j] == "---" {
				end = j
			}
		}
		for j := i; j < end; j++ { // a module-level trailer is no test body
			if strings.HasPrefix(lines[j], "if __name__") {
				end = j
			}
		}
		// Decorators above the next test belong to it.
		for end > i+1 && (strings.TrimSpace(lines[end-1]) == "" || strings.HasPrefix(strings.TrimLeftFunc(lines[end-1], isSpace), "@")) {
			end--
		}
		stripped := lines[i]
		if loc := name.FindStringIndex(stripped); loc != nil {
			stripped = stripped[loc[1]:]
		}
		entries = append(entries, Entry{
			File: file,
			Line: i + 1,
			Cand: strings.ReplaceAll(idAt(stripped), "-", ""),
			Body: strings.TrimRightFunc(strings.Join(lines[i:end], "\n"), isSpace),
		})
	}
	return entries, strings.Join(lines[:idx[0]], "\n")
}
