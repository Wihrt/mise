// Package shell reads a shell line with a real shell parser.
package shell

import (
	"strings"

	"mvdan.cc/sh/v3/syntax"

	"github.com/Wihrt/mise/harness/internal/domain"
)

// Parser finds the simple commands of a Bash line, including those in pipes,
// lists, subshells and command substitutions.
type Parser struct{}

// NewParser builds the parser.
func NewParser() *Parser { return &Parser{} }

// Parse returns one invocation per simple command. A line the parser cannot
// read is cut on its separators instead, so that a guard still sees it.
func (p *Parser) Parse(command string) []domain.Invocation {
	file, err := syntax.NewParser(syntax.Variant(syntax.LangBash)).Parse(strings.NewReader(command), "")
	if err != nil {
		return cut(command)
	}
	var invs []domain.Invocation
	printer := syntax.NewPrinter()
	syntax.Walk(file, func(node syntax.Node) bool {
		call, ok := node.(*syntax.CallExpr)
		if !ok || len(call.Args) == 0 {
			return true
		}
		words := make([]string, 0, len(call.Args))
		for _, w := range call.Args {
			words = append(words, text(printer, w))
		}
		invs = append(invs, domain.Invocation{Words: words})
		return true
	})
	return invs
}

// text is a word as the program receives it when it is literal or quoted;
// an expansion keeps its source text.
func text(printer *syntax.Printer, w *syntax.Word) string {
	var b strings.Builder
	for _, part := range w.Parts {
		switch x := part.(type) {
		case *syntax.Lit:
			b.WriteString(x.Value)
		case *syntax.SglQuoted:
			b.WriteString(x.Value)
		case *syntax.DblQuoted:
			for _, q := range x.Parts {
				if lit, ok := q.(*syntax.Lit); ok {
					b.WriteString(lit.Value)
				} else {
					_ = printer.Print(&b, q)
				}
			}
		default:
			_ = printer.Print(&b, part)
		}
	}
	return b.String()
}

// cut splits a line on && || ; | and then on blanks.
func cut(command string) []domain.Invocation {
	var invs []domain.Invocation
	for _, seg := range strings.FieldsFunc(command, func(r rune) bool { return r == '&' || r == '|' || r == ';' || r == '\n' }) {
		if words := strings.Fields(seg); len(words) > 0 {
			invs = append(invs, domain.Invocation{Words: words})
		}
	}
	return invs
}
