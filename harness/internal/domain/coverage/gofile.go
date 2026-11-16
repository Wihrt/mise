package coverage

import (
	"go/ast"
	"go/parser"
	"go/token"
)

// HasFunc reports whether a Go source declares a function or a method. A source
// that does not parse is taken to declare one: it is then judged like the others.
func HasFunc(source string) bool {
	f, err := parser.ParseFile(token.NewFileSet(), "", source, parser.SkipObjectResolution)
	if err != nil {
		return true
	}
	for _, d := range f.Decls {
		if _, ok := d.(*ast.FuncDecl); ok {
			return true
		}
	}
	return false
}
