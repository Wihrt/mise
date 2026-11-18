package coverage

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
)

// ErrorReturns gives the positions (line, column) of the return statements that
// are the whole body of an if testing an error against nil:
// `if err != nil { return ... }`, with an identifier named err or ending in err
// or Err.
func ErrorReturns(source string) (map[[2]int]bool, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "", source, 0)
	if err != nil {
		return nil, err
	}
	out := map[[2]int]bool{}
	ast.Inspect(f, func(n ast.Node) bool {
		s, ok := n.(*ast.IfStmt)
		if !ok || !comparesErrorToNil(s.Cond) || len(s.Body.List) != 1 {
			return true
		}
		if ret, ok := s.Body.List[0].(*ast.ReturnStmt); ok {
			p := fset.Position(ret.Pos())
			out[[2]int{p.Line, p.Column}] = true
		}
		return true
	})
	return out, nil
}

func comparesErrorToNil(cond ast.Expr) bool {
	b, ok := cond.(*ast.BinaryExpr)
	if !ok || b.Op != token.NEQ {
		return false
	}
	id, ok := b.X.(*ast.Ident)
	nilID, ok2 := b.Y.(*ast.Ident)
	return ok && ok2 && nilID.Name == "nil" &&
		(id.Name == "err" || strings.HasSuffix(id.Name, "err") || strings.HasSuffix(id.Name, "Err"))
}
