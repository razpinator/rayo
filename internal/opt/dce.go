package opt

import "rayo/internal/ast"

// Dead code elimination.
//
// This pass performs two behavior-preserving simplifications:
//
//  1. Unreachable-statement removal: within any statement block, statements
//     that follow a terminator (`return`, `raise`, `break`, `continue`) can
//     never execute, so they are dropped.
//
//  2. Constant-branch pruning: an `if`/`elif` whose condition folds to a
//     constant is simplified — a constantly-true branch replaces the whole
//     conditional with that branch's body, and a constantly-false branch is
//     removed (falling through to the remaining elifs/else). This composes with
//     constant folding, which supplies the literal conditions.
//
// Both are local and preserve observable behavior: removed statements are
// provably unreachable, and pruned branches are provably (not) taken.

// EliminateDeadCode runs dead-code elimination over the module in place.
func EliminateDeadCode(mod *ast.Module) *ast.Module {
	if mod == nil {
		return nil
	}
	mod.Body = dceStmts(mod.Body)
	return mod
}

// dceStmts simplifies a statement list: it recurses into nested blocks, prunes
// constant conditionals, and truncates the list after the first terminator.
func dceStmts(stmts []ast.Stmt) []ast.Stmt {
	out := make([]ast.Stmt, 0, len(stmts))
	for _, stmt := range stmts {
		// Constant-branch pruning may replace a single if-statement with the
		// statements of its taken branch (or drop it entirely).
		if ifStmt, ok := stmt.(*ast.IfStmt); ok {
			replaced, taken := pruneIf(ifStmt)
			if taken != nil {
				out = append(out, taken...)
				if endsInTerminator(taken) {
					return out
				}
				continue
			}
			stmt = replaced
		}

		out = append(out, dceStmt(stmt))

		if isTerminator(stmt) {
			// Everything after a terminator is unreachable.
			break
		}
	}
	return out
}

// dceStmt recurses into a statement's nested blocks.
func dceStmt(stmt ast.Stmt) ast.Stmt {
	switch s := stmt.(type) {
	case *ast.FuncDef:
		s.Body = dceStmts(s.Body)
	case *ast.ClassDef:
		for _, m := range s.Methods {
			m.Body = dceStmts(m.Body)
		}
		for _, p := range s.Properties {
			p.Get = dceStmts(p.Get)
			p.Set = dceStmts(p.Set)
		}
	case *ast.IfStmt:
		s.Then = dceStmts(s.Then)
		for _, e := range s.Elifs {
			e.Body = dceStmts(e.Body)
		}
		s.Else = dceStmts(s.Else)
	case *ast.WhileStmt:
		s.Body = dceStmts(s.Body)
	case *ast.ForStmt:
		s.Body = dceStmts(s.Body)
	case *ast.TryStmt:
		s.Body = dceStmts(s.Body)
		for _, exc := range s.Excepts {
			exc.Body = dceStmts(exc.Body)
		}
		s.Finally = dceStmts(s.Finally)
	case *ast.MatchStmt:
		for _, cs := range s.Cases {
			cs.Body = dceStmts(cs.Body)
		}
	}
	return stmt
}

// pruneIf simplifies an if-statement when its condition (or an elif's) is a
// constant boolean. It returns (replacement, taken):
//   - if a branch is constantly true, taken is that branch's (recursively
//     simplified) body and the whole conditional collapses to it;
//   - otherwise replacement is the if-statement with constantly-false branches
//     removed and remaining blocks recursed into, and taken is nil.
func pruneIf(s *ast.IfStmt) (ast.Stmt, []ast.Stmt) {
	// Primary condition.
	if b, ok := constBool(s.Cond); ok {
		if b {
			return nil, dceStmts(s.Then)
		}
		// Condition is false: promote the first elif to the primary, else fall
		// to else.
		if len(s.Elifs) > 0 {
			next := &ast.IfStmt{Cond: s.Elifs[0].Cond, Then: s.Elifs[0].Body, Elifs: s.Elifs[1:], Else: s.Else}
			next.SetSpan(s.Span())
			return pruneIf(next)
		}
		if len(s.Else) > 0 {
			return nil, dceStmts(s.Else)
		}
		// Nothing left: emit an empty (no-op) block by returning an if with a
		// false literal and empty body is undesirable; instead return a
		// no-effect statement list.
		return nil, nil
	}

	// Non-constant primary condition: keep it, but prune constantly-false
	// elifs and recurse into all bodies.
	s.Then = dceStmts(s.Then)
	var keptElifs []*ast.Elif
	for _, e := range s.Elifs {
		if b, ok := constBool(e.Cond); ok {
			if !b {
				continue // drop constantly-false elif
			}
			// A constantly-true elif becomes the else and stops the chain.
			e.Body = dceStmts(e.Body)
			s.Else = e.Body
			s.Elifs = keptElifs
			return s, nil
		}
		e.Body = dceStmts(e.Body)
		keptElifs = append(keptElifs, e)
	}
	s.Elifs = keptElifs
	s.Else = dceStmts(s.Else)
	return s, nil
}

// constBool reports whether expr is a boolean literal and its value.
func constBool(expr ast.Expr) (bool, bool) {
	if lit, ok := expr.(*ast.Literal); ok {
		if b, ok := lit.Value.(bool); ok {
			return b, true
		}
	}
	return false, false
}

// isTerminator reports whether a statement unconditionally transfers control
// out of the current block (so following statements are unreachable).
func isTerminator(stmt ast.Stmt) bool {
	switch s := stmt.(type) {
	case *ast.ReturnStmt:
		return true
	case *ast.ExprStmt:
		return isRaiseOrJump(s.Expr)
	}
	return false
}

// endsInTerminator reports whether the last statement of a list is a
// terminator.
func endsInTerminator(stmts []ast.Stmt) bool {
	if len(stmts) == 0 {
		return false
	}
	return isTerminator(stmts[len(stmts)-1])
}

// isRaiseOrJump reports whether expr models `raise(...)`, `panic(...)`,
// `break`, or `continue` — all of which terminate normal fall-through. These
// are surfaced by the parser as calls/names (see internal/sem/flow.go).
func isRaiseOrJump(expr ast.Expr) bool {
	switch e := expr.(type) {
	case *ast.Call:
		if name, ok := e.Func.(*ast.Name); ok {
			return name.Ident == "raise" || name.Ident == "panic"
		}
	case *ast.Name:
		return e.Ident == "break" || e.Ident == "continue"
	}
	return false
}
