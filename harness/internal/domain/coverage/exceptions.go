package coverage

import (
	"errors"
	"fmt"
	"strings"
)

const (
	wholeFile = "fichier entier"
	// DebtReason is the reason of the entries the baseline writes.
	DebtReason = "dette initiale"
)

// Entry is one written exception: a first line of source, or the whole file.
type Entry struct {
	Text, Reason string
	Whole        bool
}

// Section holds the exceptions of one file.
type Section struct {
	File    string
	Entries []Entry
}

// Exceptions is the content of COVERAGE.md, in file order.
type Exceptions []Section

// Stale is an exception that accepts no finding.
type Stale struct{ File, Text string }

// LineText is the trimmed text of line n (from 1) of a source.
func LineText(source string, n int) string {
	lines := strings.Split(source, "\n")
	if n < 1 || n > len(lines) {
		return ""
	}
	return strings.TrimSpace(lines[n-1])
}

// ParseExceptions reads COVERAGE.md.
func ParseExceptions(md string) (Exceptions, error) {
	var out Exceptions
	for i, l := range strings.Split(md, "\n") {
		if file, ok := strings.CutPrefix(l, "## "); ok {
			out = append(out, Section{File: strings.TrimSpace(file)})
			continue
		}
		rest, ok := strings.CutPrefix(l, "- ")
		if !ok {
			continue
		}
		if len(out) == 0 {
			return nil, fmt.Errorf("COVERAGE.md line %d: entry outside a section", i+1)
		}
		e, err := parseEntry(rest)
		if err != nil {
			return nil, fmt.Errorf("COVERAGE.md line %d: %w", i+1, err)
		}
		s := &out[len(out)-1]
		s.Entries = append(s.Entries, e)
	}
	return out, nil
}

func parseEntry(rest string) (Entry, error) {
	var e Entry
	var tail string
	if t, ok := strings.CutPrefix(rest, wholeFile); ok {
		e.Whole, tail = true, t
	} else {
		text, t, ok := codeSpan(rest)
		if !ok {
			return e, errors.New("entry is neither `text` nor " + wholeFile)
		}
		e.Text, tail = text, t
	}
	reason, ok := strings.CutPrefix(strings.TrimSpace(tail), ":")
	if e.Reason = strings.TrimSpace(reason); !ok || e.Reason == "" {
		return e, errors.New("entry has no reason")
	}
	return e, nil
}

// codeSpan reads a Markdown code span at the start of s: its text and what follows.
func codeSpan(s string) (text, tail string, ok bool) {
	n := len(s) - len(strings.TrimLeft(s, "`"))
	if n == 0 {
		return "", "", false
	}
	for i := n; i < len(s); {
		if s[i] != '`' {
			i++
			continue
		}
		run := len(s[i:]) - len(strings.TrimLeft(s[i:], "`"))
		if run == n {
			text = s[n:i]
			if len(text) > 1 && text[0] == ' ' && text[len(text)-1] == ' ' && strings.TrimSpace(text) != "" {
				text = text[1 : len(text)-1]
			}
			return text, s[i+run:], true
		}
		i += run
	}
	return "", "", false
}

// Apply splits the findings into those a written exception accepts (their
// Reason set) and the others, and lists the exceptions that accept nothing. A
// PRAGMA is never accepted.
func (e Exceptions) Apply(fs []Finding) (accepted, refused []Finding, stale []Stale) {
	used := map[[2]int]bool{}
	for _, f := range fs {
		reason, ok := "", false
		for si, s := range e {
			if s.File != f.File || f.Kind == "PRAGMA" {
				continue
			}
			for ei, en := range s.Entries {
				if en.Whole || (en.Text == f.Text && f.Kind != "NOT MEASURED") {
					used[[2]int{si, ei}] = true
					if !ok {
						reason, ok = en.Reason, true
					}
				}
			}
		}
		if ok {
			f.Reason = reason
			accepted = append(accepted, f)
		} else {
			refused = append(refused, f)
		}
	}
	for si, s := range e {
		for ei, en := range s.Entries {
			if used[[2]int{si, ei}] {
				continue
			}
			text := en.Text
			if en.Whole {
				text = wholeFile
			}
			stale = append(stale, Stale{File: s.File, Text: text})
		}
	}
	return accepted, refused, stale
}

// WithDebt adds, for the refused findings, one entry per distinct text and per
// file (one whole-file entry for a NOT MEASURED file), with the reason of the
// initial debt. The entries already there stay as they are. PRAGMA is skipped.
func (e Exceptions) WithDebt(refused []Finding) Exceptions {
	out := make(Exceptions, len(e))
	for i, s := range e {
		out[i] = Section{File: s.File, Entries: append([]Entry(nil), s.Entries...)}
	}
	for _, f := range refused {
		if f.Kind == "PRAGMA" {
			continue
		}
		entry := Entry{Text: f.Text, Reason: DebtReason, Whole: f.Kind == "NOT MEASURED"}
		si := -1
		for i, s := range out {
			if s.File == f.File {
				si = i
			}
		}
		if si < 0 {
			out = append(out, Section{File: f.File})
			si = len(out) - 1
		}
		if !contains(out[si].Entries, entry) {
			out[si].Entries = append(out[si].Entries, entry)
		}
	}
	return out
}

func contains(entries []Entry, e Entry) bool {
	for _, x := range entries {
		if x.Whole == e.Whole && x.Text == e.Text {
			return true
		}
	}
	return false
}

// Markdown renders COVERAGE.md.
func (e Exceptions) Markdown() string {
	var b strings.Builder
	for i, s := range e {
		if i > 0 {
			b.WriteString("\n")
		}
		fmt.Fprintf(&b, "## %s\n\n", s.File)
		for _, en := range s.Entries {
			if en.Whole {
				fmt.Fprintf(&b, "- %s : %s\n", wholeFile, en.Reason)
			} else {
				fmt.Fprintf(&b, "- %s : %s\n", quote(en.Text), en.Reason)
			}
		}
	}
	return b.String()
}

// quote is the Markdown code span of a text: delimited by more backticks than
// its longest run, padded by a space when it starts or ends with a backtick.
func quote(text string) string {
	longest, run := 0, 0
	for _, c := range text {
		if c == '`' {
			run++
			longest = max(longest, run)
		} else {
			run = 0
		}
	}
	d := strings.Repeat("`", longest+1)
	if strings.HasPrefix(text, "`") || strings.HasSuffix(text, "`") {
		text = " " + text + " "
	}
	return d + text + d
}
