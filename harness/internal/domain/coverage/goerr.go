package coverage

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
)

// ErrorReturns gives the positions (line, column) of the return statements that
// are the whole body of an if testing an error against nil and one of whose
// results names that error: `if err != nil { return ..., err }`, with an
// identifier named err or ending in err or Err.
func ErrorReturns(source string) (map[[2]int]bool, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "", source, 0)
	if err != nil {
		return nil, err
	}
	out := map[[2]int]bool{}
	ast.Inspect(f, func(n ast.Node) bool {
		s, ok := n.(*ast.IfStmt)
		if !ok || len(s.Body.List) != 1 {
			return true
		}
		name := testedError(s.Cond)
		if ret, ok := s.Body.List[0].(*ast.ReturnStmt); ok && name != "" && returnsIdent(ret, name) {
			p := fset.Position(ret.Pos())
			out[[2]int{p.Line, p.Column}] = true
		}
		return true
	})
	return out, nil
}

// testedError gives the name of the error compared to nil, or "".
func testedError(cond ast.Expr) string {
	b, ok := cond.(*ast.BinaryExpr)
	if !ok || b.Op != token.NEQ {
		return ""
	}
	id, ok := b.X.(*ast.Ident)
	nilID, ok2 := b.Y.(*ast.Ident)
	if ok && ok2 && nilID.Name == "nil" &&
		(id.Name == "err" || strings.HasSuffix(id.Name, "err") || strings.HasSuffix(id.Name, "Err")) {
		return id.Name
	}
	return ""
}

// returnsIdent tells whether one of the results of ret names the identifier.
func returnsIdent(ret *ast.ReturnStmt, name string) bool {
	found := false
	for _, r := range ret.Results {
		ast.Inspect(r, func(n ast.Node) bool {
			if id, ok := n.(*ast.Ident); ok && id.Name == name {
				found = true
			}
			return !found
		})
	}
	return found
}
