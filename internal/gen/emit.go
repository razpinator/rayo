package gen

import (
	"fmt"
	"strings"

	"rayo/internal/ast"
)

// EmitModule emits Go code for a module AST. It is a convenience entry point
// used by tests and benchmarks; the multi-file compiler drives EmitStmt
// directly (see internal/compile). The `fmt` import is added only when the
// module actually calls print(), keeping generated code go-vet clean.
func EmitModule(mod *ast.Module, ctx *GenContext) string {
	ctx.Code.WriteString(fmt.Sprintf("package %s\n\n", ctx.PackageName))

	hasPrint := false
	for _, stmt := range mod.Body {
		if ContainsPrint(stmt) {
			hasPrint = true
			break
		}
	}
	imported := map[string]bool{}
	for _, imp := range mod.Imports {
		imported[imp.Path] = true
	}
	if hasPrint && !imported["fmt"] {
		ctx.Code.WriteString("import \"fmt\"\n")
	}
	for _, imp := range mod.Imports {
		ctx.Code.WriteString(fmt.Sprintf("import %q\n", imp.Path))
	}

	// Emit top-level function definitions first, then wrap the remaining
	// statements in main() so `return` at top level is valid Go.
	if ctx.declared == nil {
		ctx.declared = map[string]bool{}
	}
	var funcs, body strings.Builder
	for _, stmt := range mod.Body {
		target := &body
		if _, ok := stmt.(*ast.FuncDef); ok {
			target = &funcs
		}
		sub := &GenContext{PackageName: ctx.PackageName, Code: target, TempVarIdx: ctx.TempVarIdx, declared: ctx.declared}
		EmitStmt(stmt, sub)
		ctx.TempVarIdx = sub.TempVarIdx
	}
	ctx.Code.WriteString(funcs.String())
	if body.Len() > 0 {
		ctx.Code.WriteString("func main() {\n")
		ctx.Code.WriteString(body.String())
		ctx.Code.WriteString("}\n")
	}
	return ctx.Code.String()
}

// EmitStmt emits Go for a single statement.
func EmitStmt(stmt ast.Stmt, ctx *GenContext) {
	switch s := stmt.(type) {
	case *ast.FuncDef:
		// Rayo is expression-oriented; use `any` so `return expr` is valid Go.
		params := make([]string, len(s.Params))
		for i, p := range s.Params {
			params[i] = p.Name + " any"
		}
		ctx.Code.WriteString(fmt.Sprintf("func %s(%s) any {\n", s.Name, strings.Join(params, ", ")))
		for _, bodyStmt := range s.Body {
			EmitStmt(bodyStmt, ctx)
		}
		// Guarantee the function compiles as `func() any` even when the body
		// falls through without an explicit return.
		if !endsWithReturn(s.Body) {
			ctx.Code.WriteString("return nil\n")
		}
		ctx.Code.WriteString("}\n")

	case *ast.VarStmt:
		ctx.markDeclared(s.Name)
		ctx.Code.WriteString(fmt.Sprintf("var %s = %s\n", s.Name, emitExpr(s.Value, ctx)))
		// Reference the binding so an unused declaration does not break the Go
		// build; Rayo tolerates locally-unused bindings (they are warned, not
		// rejected, by the semantic checker).
		ctx.Code.WriteString(fmt.Sprintf("_ = %s\n", s.Name))

	case *ast.AssignStmt:
		// Plain-name targets declare with `:=` on first sight and reassign with
		// `=` afterwards. Non-name targets (index/attr) always use `=`.
		if name, ok := s.Target.(*ast.Name); ok {
			if ctx.markDeclared(name.Ident) {
				ctx.Code.WriteString(fmt.Sprintf("%s := %s\n", name.Ident, emitExpr(s.Value, ctx)))
				ctx.Code.WriteString(fmt.Sprintf("_ = %s\n", name.Ident))
			} else {
				ctx.Code.WriteString(fmt.Sprintf("%s = %s\n", name.Ident, emitExpr(s.Value, ctx)))
			}
		} else {
			ctx.Code.WriteString(fmt.Sprintf("%s = %s\n", emitExpr(s.Target, ctx), emitExpr(s.Value, ctx)))
		}

	case *ast.ExprStmt:
		ctx.Code.WriteString(fmt.Sprintf("%s\n", emitExpr(s.Expr, ctx)))

	case *ast.IfStmt:
		ctx.Code.WriteString(fmt.Sprintf("if %s {\n", emitExpr(s.Cond, ctx)))
		for _, st := range s.Then {
			EmitStmt(st, ctx)
		}
		for _, elif := range s.Elifs {
			ctx.Code.WriteString(fmt.Sprintf("} else if %s {\n", emitExpr(elif.Cond, ctx)))
			for _, st := range elif.Body {
				EmitStmt(st, ctx)
			}
		}
		if len(s.Else) > 0 {
			ctx.Code.WriteString("} else {\n")
			for _, st := range s.Else {
				EmitStmt(st, ctx)
			}
		}
		ctx.Code.WriteString("}\n")

	case *ast.WhileStmt:
		ctx.Code.WriteString(fmt.Sprintf("for %s {\n", emitExpr(s.Cond, ctx)))
		for _, st := range s.Body {
			EmitStmt(st, ctx)
		}
		ctx.Code.WriteString("}\n")

	case *ast.ForStmt:
		// `for x in iter { ... }` lowers to a Go range over the iterable.
		ctx.Code.WriteString(fmt.Sprintf("for _, %s := range %s {\n", s.Var, emitExpr(s.Iter, ctx)))
		ctx.Code.WriteString(fmt.Sprintf("_ = %s\n", s.Var))
		for _, st := range s.Body {
			EmitStmt(st, ctx)
		}
		ctx.Code.WriteString("}\n")

	case *ast.TryStmt:
		emitTry(s, ctx)

	case *ast.ReturnStmt:
		if s.Value != nil {
			ctx.Code.WriteString(fmt.Sprintf("return %s\n", emitExpr(s.Value, ctx)))
		} else {
			ctx.Code.WriteString("return nil\n")
		}
	}
}

// emitTry lowers try/except/finally to Go's recover-based error handling. The
// try body runs inside a closure whose panic is recovered; except handlers are
// matched in order and a finally block is emitted via defer.
func emitTry(s *ast.TryStmt, ctx *GenContext) {
	if len(s.Finally) > 0 {
		ctx.Code.WriteString("func() {\n")
		ctx.Code.WriteString("defer func() {\n")
		for _, st := range s.Finally {
			EmitStmt(st, ctx)
		}
		ctx.Code.WriteString("}()\n")
	}
	// Recover from raised errors and dispatch to except handlers.
	ctx.Code.WriteString("func() {\n")
	if len(s.Excepts) > 0 {
		errVar := "_exc"
		if s.Excepts[0].Var != "" {
			errVar = s.Excepts[0].Var
		}
		ctx.Code.WriteString("defer func() {\n")
		ctx.Code.WriteString(fmt.Sprintf("if %s := recover(); %s != nil {\n", errVar, errVar))
		ctx.Code.WriteString(fmt.Sprintf("_ = %s\n", errVar))
		for _, exc := range s.Excepts {
			for _, st := range exc.Body {
				EmitStmt(st, ctx)
			}
		}
		ctx.Code.WriteString("}\n")
		ctx.Code.WriteString("}()\n")
	}
	for _, st := range s.Body {
		EmitStmt(st, ctx)
	}
	ctx.Code.WriteString("}()\n")
	if len(s.Finally) > 0 {
		ctx.Code.WriteString("}()\n")
	}
}

// endsWithReturn reports whether the final statement of a body is a return, so
// the codegen can skip appending a synthetic `return nil`.
func endsWithReturn(stmts []ast.Stmt) bool {
	if len(stmts) == 0 {
		return false
	}
	_, ok := stmts[len(stmts)-1].(*ast.ReturnStmt)
	return ok
}

func emitExpr(expr ast.Expr, ctx *GenContext) string {
	switch e := expr.(type) {
	case *ast.Literal:
		return emitLiteral(e.Value)
	case *ast.Name:
		return e.Ident
	case *ast.UnaryOp:
		op := e.Op
		if op == "not" {
			op = "!"
		}
		return fmt.Sprintf("%s%s", op, emitExpr(e.Right, ctx))
	case *ast.BinaryOp:
		return fmt.Sprintf("(%s %s %s)", emitExpr(e.Left, ctx), goBinaryOp(e.Op), emitExpr(e.Right, ctx))
	case *ast.Call:
		return emitCall(e, ctx)
	case *ast.Index:
		return fmt.Sprintf("%s[%s]", emitExpr(e.Target, ctx), emitExpr(e.Index, ctx))
	case *ast.Attr:
		return fmt.Sprintf("%s.%s", emitExpr(e.Target, ctx), e.Attr)
	case *ast.DictLit:
		return LowerDict(e, ctx)
	case *ast.ListLit:
		return LowerList(e, ctx)
	case *ast.Lambda:
		return emitLambda(e, ctx)
	default:
		return "nil"
	}
}

// emitLiteral renders a literal value as valid Go source. Numbers are emitted
// bare; strings are Go-quoted; booleans and nil map directly.
func emitLiteral(v any) string {
	switch x := v.(type) {
	case string:
		return fmt.Sprintf("%q", x)
	case bool:
		return fmt.Sprintf("%v", x)
	case nil:
		return "nil"
	default: // int, int64, float64, etc.
		return fmt.Sprintf("%v", x)
	}
}

// goBinaryOp maps Rayo operators to their Go equivalents.
func goBinaryOp(op string) string {
	switch op {
	case "and":
		return "&&"
	case "or":
		return "||"
	default:
		return op
	}
}

func emitCall(e *ast.Call, ctx *GenContext) string {
	funcName := emitExpr(e.Func, ctx)
	if funcName == "print" {
		funcName = "fmt.Println"
	}
	args := make([]string, len(e.Args))
	for i, arg := range e.Args {
		args[i] = emitExpr(arg, ctx)
	}
	return fmt.Sprintf("%s(%s)", funcName, strings.Join(args, ", "))
}

func emitLambda(e *ast.Lambda, ctx *GenContext) string {
	params := make([]string, len(e.Params))
	for i, p := range e.Params {
		params[i] = p.Name + " any"
	}
	return fmt.Sprintf("func(%s) any { return %s }", strings.Join(params, ", "), emitExpr(e.Body, ctx))
}
