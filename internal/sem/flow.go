package sem

import "rayo/internal/ast"

// Flow analysis: definite "must return" (a.k.a. missing-return) checking.
//
// MustReturn reports whether every control-flow path through stmts ends in a
// terminating statement (a `return`, or a `raise`/panic call). It is used to
// enforce that a function whose body is expected to produce a value does so on
// all paths, matching the spec's guarantee of statically-typed returns.

// MustReturn reports whether the statement list always terminates the enclosing
// function on every path.
func MustReturn(stmts []ast.Stmt) bool {
	for _, stmt := range stmts {
		if stmtTerminates(stmt) {
			return true
		}
	}
	return false
}

// stmtTerminates reports whether a single statement guarantees the function
// exits (return/raise) on all paths that flow through it.
func stmtTerminates(stmt ast.Stmt) bool {
	switch s := stmt.(type) {
	case *ast.ReturnStmt:
		return true
	case *ast.ExprStmt:
		// `raise ...` and `panic(...)` are terminators.
		return isRaise(s.Expr)
	case *ast.IfStmt:
		// An if terminates only if the then-branch, every elif branch, and a
		// present else branch all terminate. Without an else, control can fall
		// through, so it does not terminate.
		if len(s.Else) == 0 {
			return false
		}
		if !MustReturn(s.Then) || !MustReturn(s.Else) {
			return false
		}
		for _, elif := range s.Elifs {
			if !MustReturn(elif.Body) {
				return false
			}
		}
		return true
	case *ast.TryStmt:
		// A try terminates if the body terminates and every except handler
		// terminates. A finally block that terminates also forces termination.
		if MustReturn(s.Finally) {
			return true
		}
		if !MustReturn(s.Body) {
			return false
		}
		for _, exc := range s.Excepts {
			if !MustReturn(exc.Body) {
				return false
			}
		}
		return true
	case *ast.WhileStmt:
		// `while true { ... }` with no break never exits normally, so it
		// terminates the surrounding flow. Any other loop condition may skip
		// the body entirely and fall through.
		return isTrueLiteral(s.Cond) && !containsBreak(s.Body)
	case *ast.MatchStmt:
		// A match terminates only if it is exhaustive (has a wildcard/capture
		// default arm) and every arm terminates. Without a default arm the
		// subject may match nothing and fall through.
		hasDefault := false
		for _, cs := range s.Cases {
			if cs.IsWildcard || cs.Binding != "" {
				hasDefault = true
			}
			if !MustReturn(cs.Body) {
				return false
			}
		}
		return hasDefault && len(s.Cases) > 0
	default:
		// ForStmt and other statements do not guarantee termination.
		return false
	}
}

// isRaise reports whether expr is a `raise ...` or `panic(...)` expression.
// The parser currently models `raise Foo(...)` as a call whose callee resolves
// to the identifier `raise`; `panic` is the Go primitive users may drop to.
func isRaise(expr ast.Expr) bool {
	call, ok := expr.(*ast.Call)
	if !ok {
		return false
	}
	if name, ok := call.Func.(*ast.Name); ok {
		return name.Ident == "raise" || name.Ident == "panic"
	}
	return false
}

func isTrueLiteral(expr ast.Expr) bool {
	lit, ok := expr.(*ast.Literal)
	if !ok {
		return false
	}
	b, ok := lit.Value.(bool)
	return ok && b
}

// containsBreak reports whether the (non-nested-loop) statement list contains a
// break that would let a `while true` exit. Break is modelled as an ExprStmt of
// the name `break`, matching how the parser surfaces the keyword today.
func containsBreak(stmts []ast.Stmt) bool {
	for _, stmt := range stmts {
		switch s := stmt.(type) {
		case *ast.ExprStmt:
			if name, ok := s.Expr.(*ast.Name); ok && name.Ident == "break" {
				return true
			}
		case *ast.IfStmt:
			if containsBreak(s.Then) || containsBreak(s.Else) {
				return true
			}
			for _, elif := range s.Elifs {
				if containsBreak(elif.Body) {
					return true
				}
			}
		case *ast.TryStmt:
			if containsBreak(s.Body) || containsBreak(s.Finally) {
				return true
			}
			for _, exc := range s.Excepts {
				if containsBreak(exc.Body) {
					return true
				}
			}
		}
	}
	return false
}
