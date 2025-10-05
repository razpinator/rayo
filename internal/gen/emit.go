package gen

import (
	"fmt"
	"rayo/internal/ast"
)

// EmitModule emits Go code for a module AST.
func EmitModule(mod *ast.Module, ctx *GenContext) string {
	ctx.Code.WriteString(fmt.Sprintf("package %s\n\n", ctx.PackageName))
	for _, imp := range mod.Imports {
		ctx.Code.WriteString(fmt.Sprintf("import \"%s\"\n", imp.Path))
	}
	// Check if any top-level statement is a ReturnStmt
	hasReturn := false
	for _, stmt := range mod.Body {
		if _, ok := stmt.(*ast.ReturnStmt); ok {
			hasReturn = true
			break
		}
	}
	if hasReturn {
		ctx.Code.WriteString("func main() {\n")
		for _, stmt := range mod.Body {
			EmitStmt(stmt, ctx)
		}
		ctx.Code.WriteString("}\n")
	} else {
		for _, stmt := range mod.Body {
			EmitStmt(stmt, ctx)
		}
	}
	return ctx.Code.String()
}

func EmitStmt(stmt ast.Stmt, ctx *GenContext) {
	switch s := stmt.(type) {
	case *ast.FuncDef:
		ctx.Code.WriteString(fmt.Sprintf("func %s() {\n", s.Name))
		for _, bodyStmt := range s.Body {
			EmitStmt(bodyStmt, ctx)
		}
		ctx.Code.WriteString("}\n")
	case *ast.VarStmt:
		ctx.Code.WriteString(fmt.Sprintf("var %s = %s\n", s.Name, emitExpr(s.Value, ctx)))
	case *ast.ReturnStmt:
		ctx.Code.WriteString(fmt.Sprintf("return %s\n", emitExpr(s.Value, ctx)))
		// ...extend for other statements...
	}
}

func emitExpr(expr ast.Expr, ctx *GenContext) string {
	switch e := expr.(type) {
	case *ast.Literal:
		return fmt.Sprintf("%v", e.Value)
	case *ast.Name:
		return e.Ident
	// ...extend for other expressions...
	default:
		return "<expr>"
	}
}
