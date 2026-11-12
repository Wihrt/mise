package tdd

import (
	"regexp"
	"sort"
	"strings"
)

var (
	symbolPatterns = []*regexp.Regexp{
		regexp.MustCompile(`undefined: ([` + word + `.]+)`),
		regexp.MustCompile(`NameError: name '([` + word + `]+)' is not defined`),
		regexp.MustCompile(`cannot import name '([` + word + `]+)'`),
		regexp.MustCompile(`No module named '([` + word + `.]+)'`),
		regexp.MustCompile(`(?:Reference to undeclared[^:\n]*|Unsupported attribute)[^` + word + `]+.*?"([` + word + `]+)"`),
	}
	errorLine = regexp.MustCompile(`undefined: |NameError|ImportError|ModuleNotFoundError|cannot import name|Reference to undeclared|Unsupported attribute`)
)

// CompileError is an output line that looks like a compile or collect error,
// with the symbols it names.
type CompileError struct {
	Line    string
	Symbols []string
}

// CompileErrors finds the compile and collect errors of a task output.
func CompileErrors(text string) []CompileError {
	var res []CompileError
	for _, line := range Lines(text) {
		if !errorLine.MatchString(line) {
			continue
		}
		set := map[string]bool{}
		for _, rx := range symbolPatterns {
			for _, m := range rx.FindAllStringSubmatch(line, -1) {
				set[m[1]] = true
				set[m[1][strings.LastIndex(m[1], ".")+1:]] = true
			}
		}
		syms := make([]string, 0, len(set))
		for s := range set {
			syms = append(syms, s)
		}
		sort.Strings(syms)
		res = append(res, CompileError{Line: line, Symbols: syms})
	}
	return res
}
