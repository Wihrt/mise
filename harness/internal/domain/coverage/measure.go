// Package coverage reads coverage reports and tells which blocks of the
// production code no report saw executed. It is pure: no file system, no process.
package coverage

import (
	"errors"
	"sort"
	"strings"
)

// Pos is a line and a column of a source file.
type Pos struct{ Line, Col int }

// span is the first and the last position of a block.
type span struct{ start, end Pos }

// Measure is what the reports say, per file of the repository: for each block
// that holds statements, whether a report saw it executed.
type Measure map[string]map[span]bool

// Finding is one thing the gate reports about a file.
type Finding struct {
	Kind, File string
	Start, End int // first and last line
	At         Pos // where the block starts
	Text       string
	Reason     string
}

// Merge joins reports: a block is covered as soon as one report says so.
func Merge(ms ...Measure) Measure {
	out := Measure{}
	for _, m := range ms {
		for file, blocks := range m {
			if out[file] == nil {
				out[file] = map[span]bool{}
			}
			for s, covered := range blocks {
				out[file][s] = out[file][s] || covered
			}
		}
	}
	return out
}

// Files lists the files the reports name, sorted.
func (m Measure) Files() []string {
	files := make([]string, 0, len(m))
	for f := range m {
		files = append(files, f)
	}
	sort.Strings(files)
	return files
}

// Select keeps the files for which keep is true.
func (m Measure) Select(keep func(file string) bool) Measure {
	out := Measure{}
	for f, blocks := range m {
		if keep(f) {
			out[f] = blocks
		}
	}
	return out
}

// Findings lists the blocks no report saw executed, sorted by file then line.
// source returns the content of a file of the repository.
func (m Measure) Findings(source func(file string) string) []Finding {
	var out []Finding
	for _, file := range m.Files() {
		var lines []string
		var found []Finding
		for s, covered := range m[file] {
			if covered {
				continue
			}
			if lines == nil {
				lines = strings.Split(source(file), "\n")
			}
			text := ""
			if s.start.Line >= 1 && s.start.Line <= len(lines) {
				text = strings.TrimSpace(lines[s.start.Line-1])
			}
			found = append(found, Finding{
				Kind: "UNCOVERED", File: file, Start: s.start.Line, End: s.end.Line, At: s.start, Text: text,
			})
		}
		sort.Slice(found, func(i, j int) bool {
			if found[i].At.Line != found[j].At.Line {
				return found[i].At.Line < found[j].At.Line
			}
			return found[i].At.Col < found[j].At.Col
		})
		out = append(out, found...)
	}
	return out
}

// Parse reads a report, its format told by its content. modules maps a module
// path to its directory in the repository.
func Parse(text string, modules map[string]string) (Measure, error) {
	if strings.HasPrefix(text, "mode:") {
		return ParseGoProfile(text, modules)
	}
	return nil, errors.New("unknown coverage report format")
}
