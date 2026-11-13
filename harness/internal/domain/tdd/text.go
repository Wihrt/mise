// Package tdd holds the rules of the red and green gates: reading TESTS.md and
// a spec, naming tests, reading test reports and judging proofs. It is pure:
// it reads no file and runs no process.
package tdd

import (
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Character classes that follow Unicode, as the criteria are written in French
// (a no-break space before a colon is white space).
const (
	space = `[\s\x{1c}-\x{1f}\x{85}\x{a0}\x{1680}\x{2000}-\x{200a}\x{2028}\x{2029}\x{202f}\x{205f}\x{3000}]`
	word  = `\p{L}\p{N}_`
)

// Lines splits a text at every line boundary and drops the terminators.
func Lines(text string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(text); {
		r, size := utf8.DecodeRuneInString(text[i:])
		switch r {
		case '\r':
			lines = append(lines, text[start:i])
			if i+1 < len(text) && text[i+1] == '\n' {
				size = 2
			}
			start = i + size
		case '\n', '\v', '\f', 0x1c, 0x1d, 0x1e, 0x85, 0x2028, 0x2029:
			lines = append(lines, text[start:i])
			start = i + size
		}
		i += size
	}
	if start < len(text) {
		lines = append(lines, text[start:])
	}
	return lines
}

func isWord(r rune) bool {
	return r == '_' || unicode.IsLetter(r) || unicode.IsNumber(r)
}

// ContainsWord reports whether s occurs in text between word boundaries.
func ContainsWord(text, s string) bool {
	if s == "" {
		return false
	}
	first, _ := utf8.DecodeRuneInString(s)
	last, _ := utf8.DecodeLastRuneInString(s)
	for from := 0; ; {
		i := strings.Index(text[from:], s)
		if i < 0 {
			return false
		}
		i += from
		before, after := false, false
		if i > 0 {
			r, _ := utf8.DecodeLastRuneInString(text[:i])
			before = isWord(r)
		}
		if end := i + len(s); end < len(text) {
			r, _ := utf8.DecodeRuneInString(text[end:])
			after = isWord(r)
		}
		if before != isWord(first) && after != isWord(last) {
			return true
		}
		_, size := utf8.DecodeRuneInString(text[i:])
		from = i + size
	}
}

// field returns the n-th white-space separated field of a line, "" when absent.
func field(line string, n int) string {
	f := strings.Fields(line)
	if n < len(f) {
		return f[n]
	}
	return ""
}

// Fnmatch matches a name against a shell pattern whose * also crosses slashes.
func Fnmatch(name, pattern string) bool {
	var b strings.Builder
	b.WriteString(`^(?s:`)
	for i := 0; i < len(pattern); i++ {
		switch c := pattern[i]; c {
		case '*':
			b.WriteString(`.*`)
		case '?':
			b.WriteString(`.`)
		case '[':
			end := strings.IndexByte(pattern[i+1:], ']')
			if end <= 0 {
				b.WriteString(`\[`)
				continue
			}
			class := pattern[i+1 : i+1+end]
			if class[0] == '!' {
				class = "^" + class[1:]
			}
			b.WriteString("[" + strings.ReplaceAll(class, `\`, `\\`) + "]")
			i += end + 1
		default:
			b.WriteString(regexp.QuoteMeta(string(c)))
		}
	}
	b.WriteString(`)$`)
	re, err := regexp.Compile(b.String())
	return err == nil && re.MatchString(name)
}

func isSpace(r rune) bool {
	return unicode.IsSpace(r) || (r >= 0x1c && r <= 0x1f)
}
