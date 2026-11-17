// Package coverage reads coverage reports and tells which blocks of the
// production code no report saw executed. It is pure: no file system, no process.
package coverage

import (
	"errors"
	"sort"
	"strings"
)

// pos is a line and a column of a source file.
type pos struct{ line, col int }

// kind tells what a span stands for: a Go block, a Python line or a Python branch.
type kind int

const (
	block kind = iota
	line
	branch
)

// span is the first and the last position of a block, a line or a branch. The
// branches of a line are told apart by desc.
type span struct {
	start, end pos
	kind       kind
	desc       string
}

// Measure is what the reports say, per file of the repository: for each block
// that holds statements, whether a report saw it executed.
type Measure map[string]map[span]bool

// Finding is one block no report saw executed.
type Finding struct {
	Kind, File string
	Start, End int    // first and last line
	Col        int    // column where a Go block starts
	Text       string // first line, trimmed: what a written exception names
	Reason     string // why a written exception accepts it
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

// Findings lists what no report saw executed, sorted by file then line: Go
// blocks, groups of Python lines that no executed line separates, and Python
// lines that carry a never-taken branch.
func (m Measure) Findings() []Finding {
	files := make([]string, 0, len(m))
	for f := range m {
		files = append(files, f)
	}
	sort.Strings(files)
	var out []Finding
	for _, file := range files {
		spans := make([]span, 0, len(m[file]))
		executed := map[int]bool{}
		for s, covered := range m[file] {
			spans = append(spans, s)
			if s.kind == line && covered {
				executed[s.start.line] = true
			}
		}
		sort.Slice(spans, func(i, j int) bool {
			if spans[i].start.line != spans[j].start.line {
				return spans[i].start.line < spans[j].start.line
			}
			return spans[i].start.col < spans[j].start.col
		})
		grouping := false
		for _, s := range spans {
			covered := m[file][s]
			switch {
			case s.kind == line && covered:
				grouping = false
			case s.kind == line && grouping:
				out[len(out)-1].End = s.start.line
			case s.kind == line:
				out = append(out, Finding{Kind: "UNCOVERED", File: file, Start: s.start.line, End: s.start.line})
				grouping = true
			case s.kind == branch && !covered && executed[s.start.line]:
				if n := len(out); n == 0 || out[n-1].Kind != "UNCOVERED BRANCH" || out[n-1].File != file || out[n-1].Start != s.start.line {
					out = append(out, Finding{Kind: "UNCOVERED BRANCH", File: file, Start: s.start.line, End: s.start.line})
				}
			case s.kind == block && !covered:
				out = append(out, Finding{Kind: "UNCOVERED", File: file, Start: s.start.line, End: s.end.line, Col: s.start.col})
			}
		}
	}
	return out
}

// Parse reads a report, its format told by its content. modules maps a module
// path to its directory in the repository.
func Parse(text string, modules map[string]string) (Measure, error) {
	if strings.HasPrefix(text, "mode:") {
		return ParseGoProfile(text, modules)
	}
	if strings.HasPrefix(text, "SF:") || strings.Contains(text, "\nSF:") {
		return ParseLCOV(text)
	}
	return nil, errors.New("unknown coverage report format")
}
