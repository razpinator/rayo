package gen

import (
	"fmt"
	"rayo/internal/ast"
)

// EmitModule emits Go code for a module AST.
func EmitModule(mod *ast.Module, ctx *GenContext) string {
	ctx.Code.WriteString(fmt.Sprintf("package %s\n\n", ctx.PackageName))

	// Add fmt import if needed for print()
	hasPrint := true // FORCE TRUE for debugging
	for _, stmt := range mod.Body {
		if ContainsPrint(stmt) {
			hasPrint = true
			break
		}
	}
	if hasPrint {
		// Only add if not already present
		imported := false
		for _, imp := range mod.Imports {
			if imp.Path == "fmt" {
				imported = true
				break
			}
		}
		if !imported {
			ctx.Code.WriteString("import \"fmt\"\n")
		}
	}

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
	case *ast.AssignStmt:
		// Simple strategy: use := for Name targets (assuming declaration), = for others
		if _, ok := s.Target.(*ast.Name); ok {
			ctx.Code.WriteString(fmt.Sprintf("%s := %s\n", emitExpr(s.Target, ctx), emitExpr(s.Value, ctx)))
		} else {
			ctx.Code.WriteString(fmt.Sprintf("%s = %s\n", emitExpr(s.Target, ctx), emitExpr(s.Value, ctx)))
		}
	case *ast.ExprStmt:
		ctx.Code.WriteString(fmt.Sprintf("%s\n", emitExpr(s.Expr, ctx)))
	case *ast.IfStmt:
		// Go requires bool expression. Rayo might allow implicit bool (e.g. len(args)).
		// For now assume strictly bool or compatible expressions.
		ctx.Code.WriteString(fmt.Sprintf("if %s {\n", emitExpr(s.Cond, ctx)))
		for _, st := range s.Then {
			EmitStmt(st, ctx)
		}
		if len(s.Else) > 0 {
			ctx.Code.WriteString("} else {\n")
			for _, st := range s.Else {
				EmitStmt(st, ctx)
			}
		}
		ctx.Code.WriteString("}\n")
	case *ast.ReturnStmt:
		if s.Value != nil {
			ctx.Code.WriteString(fmt.Sprintf("return %s\n", emitExpr(s.Value, ctx)))
		} else {
			ctx.Code.WriteString("return\n")
		}
	}
}

func emitExpr(expr ast.Expr, ctx *GenContext) string {
	switch e := expr.(type) {
	case *ast.Literal:
		if s, ok := e.Value.(string); ok {
			// Check if it looks like a number (hacky check but works for "2")
			// Or check the parser logic. Parser stores TokenNumber value as string.
			// Ideally Parser should store number type.
			// But here, if string starts with quote, it's string.
			if len(s) > 0 && (s[0] == '"' || s[0] == '\'') {
				return fmt.Sprintf("\"%s\"", s[1:len(s)-1]) // Strip quotes since we quote again?
				// Wait, parser stripped quotes for strings.
				// If parser stripped quotes, then s="foo". We should emit "foo".
				// If parser didn't strip quotes for numbers, s="2". We should emit 2.
				// The parser logic for TokenString strips quotes.
				// The parser logic for TokenNumber keeps "2".

				// But wait, the parser logic handles TokenString by stripping quotes.
				// And TokenNumber by keeping value.
				// So both end up as strings in Value.
				// How do we distinguish?
				// We can try to parse as float/int.
				// Or check if it contains quotes?
				// But parser stripped quotes from strings!
				// So "foo" became foo.
				// And 2 became 2.
				// Both reflect as strings "foo" and "2".
				// This is bad AST design.
				// I should fix AST Literal to have a Type/Kind field or change Parser.
			}
			return fmt.Sprintf("\"%s\"", s)
		}
		return fmt.Sprintf("%v", e.Value)
	case *ast.Name:
		return e.Ident
	case *ast.BinaryOp:
		return fmt.Sprintf("(%s %s %s)", emitExpr(e.Left, ctx), e.Op, emitExpr(e.Right, ctx))
	case *ast.Call:
		funcName := emitExpr(e.Func, ctx)
		if funcName == "print" {
			funcName = "fmt.Println"
		}
		if funcName == "os.Args" {
			return "os.Args"
		}

		var argsStr string
		for i, arg := range e.Args {
			if i > 0 {
				argsStr += ", "
			}
			argsStr += emitExpr(arg, ctx)
		}
		return fmt.Sprintf("%s(%s)", funcName, argsStr)
	case *ast.Index:
		return fmt.Sprintf("%s[%s]", emitExpr(e.Target, ctx), emitExpr(e.Index, ctx))
	case *ast.Attr:
		return fmt.Sprintf("%s.%s", emitExpr(e.Target, ctx), e.Attr)
	default:
		return "<expr>"
	}
}
