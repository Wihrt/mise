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

// span is the first and the last position of a block.
type span struct{ start, end pos }

// Measure is what the reports say, per file of the repository: for each block
// that holds statements, whether a report saw it executed.
type Measure map[string]map[span]bool

// Finding is one block no report saw executed.
type Finding struct {
	Kind, File string
	Start, End int // first and last line
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

// Findings lists the blocks no report saw executed, sorted by file then position.
func (m Measure) Findings() []Finding {
	files := make([]string, 0, len(m))
	for f := range m {
		files = append(files, f)
	}
	sort.Strings(files)
	var out []Finding
	for _, file := range files {
		var spans []span
		for s, covered := range m[file] {
			if !covered {
				spans = append(spans, s)
			}
		}
		sort.Slice(spans, func(i, j int) bool {
			if spans[i].start.line != spans[j].start.line {
				return spans[i].start.line < spans[j].start.line
			}
			return spans[i].start.col < spans[j].start.col
		})
		for _, s := range spans {
			out = append(out, Finding{Kind: "UNCOVERED", File: file, Start: s.start.line, End: s.end.line})
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
	return nil, errors.New("unknown coverage report format")
}
