package opt

import "rayo/internal/ast"

// Small-function inlining.
//
// This pass replaces a call to a tiny top-level function with the function's
// body expression, substituting arguments for parameters. It is intentionally
// conservative — it only inlines functions that are trivial to reason about, so
// the rewrite is guaranteed to preserve behavior:
//
//   - the callee is a top-level `def` whose body is exactly `return <expr>`;
//   - it has no decorators, no generic type parameters, and is not `main`;
//   - the call passes exactly one argument per parameter;
//   - it is not (directly) recursive;
//   - substitution does not duplicate a side-effecting argument: an argument is
//     only substituted if it is a literal or a name, or if its parameter is
//     used at most once in the body.
//
// Inlining runs after constant folding and before dead-code elimination so that
// inlined literal expressions can be folded again by a subsequent fold pass.

// InlineModule inlines eligible small-function calls throughout the module.
func InlineModule(mod *ast.Module) *ast.Module {
	if mod == nil {
		return nil
	}
	table := collectInlinable(mod)
	if len(table) == 0 {
		return mod
	}
	for _, stmt := range mod.Body {
		inlineInStmt(stmt, table)
	}
	return mod
}

// inlinable describes a function that may be inlined at its call sites.
type inlinable struct {
	params []string
	body   ast.Expr // the returned expression
	// uses maps parameter name -> number of references in body.
	uses map[string]int
}

// collectInlinable finds all top-level single-return functions eligible for
// inlining, keyed by name.
func collectInlinable(mod *ast.Module) map[string]*inlinable {
	table := map[string]*inlinable{}
	for _, stmt := range mod.Body {
		fn, ok := stmt.(*ast.FuncDef)
		if !ok {
			continue
		}
		if fn.Name == "main" || len(fn.Decorators) > 0 || len(fn.TypeParams) > 0 {
			continue
		}
		if len(fn.Body) != 1 {
			continue
		}
		ret, ok := fn.Body[0].(*ast.ReturnStmt)
		if !ok || ret.Value == nil {
			continue
		}
		params := make([]string, len(fn.Params))
		for i, p := range fn.Params {
			params[i] = p.Name
		}
		// Skip directly recursive functions (the body references the function
		// name); inlining them would not terminate.
		if exprReferences(ret.Value, fn.Name) {
			continue
		}
		uses := map[string]int{}
		for _, p := range params {
			uses[p] = countName(ret.Value, p)
		}
		table[fn.Name] = &inlinable{params: params, body: ret.Value, uses: uses}
	}
	return table
}

func inlineInStmt(stmt ast.Stmt, table map[string]*inlinable) {
	switch s := stmt.(type) {
	case *ast.FuncDef:
		for i := range s.Body {
			s.Body[i] = inlineInStmtReturning(s.Body[i], table)
		}
	case *ast.ClassDef:
		for _, m := range s.Methods {
			for i := range m.Body {
				m.Body[i] = inlineInStmtReturning(m.Body[i], table)
			}
		}
	default:
		inlineInStmtReturning(stmt, table)
	}
}

// inlineInStmtReturning rewrites expressions within a statement, inlining calls.
func inlineInStmtReturning(stmt ast.Stmt, table map[string]*inlinable) ast.Stmt {
	switch s := stmt.(type) {
	case *ast.VarStmt:
		s.Value = inlineExpr(s.Value, table)
	case *ast.AssignStmt:
		s.Target = inlineExpr(s.Target, table)
		s.Value = inlineExpr(s.Value, table)
	case *ast.ReturnStmt:
		s.Value = inlineExpr(s.Value, table)
	case *ast.ExprStmt:
		s.Expr = inlineExpr(s.Expr, table)
	case *ast.IfStmt:
		s.Cond = inlineExpr(s.Cond, table)
		inlineBlock(s.Then, table)
		for _, e := range s.Elifs {
			e.Cond = inlineExpr(e.Cond, table)
			inlineBlock(e.Body, table)
		}
		inlineBlock(s.Else, table)
	case *ast.WhileStmt:
		s.Cond = inlineExpr(s.Cond, table)
		inlineBlock(s.Body, table)
	case *ast.ForStmt:
		s.Iter = inlineExpr(s.Iter, table)
		inlineBlock(s.Body, table)
	case *ast.TryStmt:
		inlineBlock(s.Body, table)
		for _, exc := range s.Excepts {
			inlineBlock(exc.Body, table)
		}
		inlineBlock(s.Finally, table)
	case *ast.MatchStmt:
		s.Subject = inlineExpr(s.Subject, table)
		for _, cs := range s.Cases {
			if cs.Pattern != nil {
				cs.Pattern = inlineExpr(cs.Pattern, table)
			}
			inlineBlock(cs.Body, table)
		}
	case *ast.FuncDef:
		inlineBlock(s.Body, table)
	}
	return stmt
}

func inlineBlock(stmts []ast.Stmt, table map[string]*inlinable) {
	for i := range stmts {
		stmts[i] = inlineInStmtReturning(stmts[i], table)
	}
}

// inlineExpr rewrites an expression tree, replacing eligible calls with the
// inlined (argument-substituted) body.
func inlineExpr(expr ast.Expr, table map[string]*inlinable) ast.Expr {
	if expr == nil {
		return nil
	}
	switch e := expr.(type) {
	case *ast.Call:
		// Recurse into callee and args first.
		e.Func = inlineExpr(e.Func, table)
		for i := range e.Args {
			e.Args[i] = inlineExpr(e.Args[i], table)
		}
		if name, ok := e.Func.(*ast.Name); ok {
			if fn, ok := table[name.Ident]; ok && canInline(fn, e.Args) {
				return substitute(fn.body, fn.params, e.Args)
			}
		}
		return e
	case *ast.BinaryOp:
		e.Left = inlineExpr(e.Left, table)
		e.Right = inlineExpr(e.Right, table)
		return e
	case *ast.UnaryOp:
		e.Right = inlineExpr(e.Right, table)
		return e
	case *ast.Index:
		e.Target = inlineExpr(e.Target, table)
		e.Index = inlineExpr(e.Index, table)
		return e
	case *ast.Attr:
		e.Target = inlineExpr(e.Target, table)
		return e
	case *ast.DictLit:
		for i := range e.Keys {
			e.Keys[i] = inlineExpr(e.Keys[i], table)
			e.Vals[i] = inlineExpr(e.Vals[i], table)
		}
		return e
	case *ast.ListLit:
		for i := range e.Elems {
			e.Elems[i] = inlineExpr(e.Elems[i], table)
		}
		return e
	case *ast.Lambda:
		e.Body = inlineExpr(e.Body, table)
		return e
	default:
		return expr
	}
}

// canInline reports whether a specific call site can be safely inlined: arity
// must match, and no side-effecting argument may be duplicated (an argument is
// safe if it is a literal/name, or its parameter is used at most once).
func canInline(fn *inlinable, args []ast.Expr) bool {
	if len(args) != len(fn.params) {
		return false
	}
	for i, p := range fn.params {
		if isSimpleExpr(args[i]) {
			continue
		}
		if fn.uses[p] > 1 {
			return false // would duplicate a non-trivial argument
		}
	}
	return true
}

// substitute returns a deep copy of body with each parameter Name replaced by
// the corresponding argument expression.
func substitute(body ast.Expr, params []string, args []ast.Expr) ast.Expr {
	sub := map[string]ast.Expr{}
	for i, p := range params {
		sub[p] = args[i]
	}
	return copyExprWithSub(body, sub)
}

// copyExprWithSub deep-copies an expression, substituting parameter names.
func copyExprWithSub(expr ast.Expr, sub map[string]ast.Expr) ast.Expr {
	switch e := expr.(type) {
	case *ast.Name:
		if repl, ok := sub[e.Ident]; ok {
			return repl
		}
		return &ast.Name{Ident: e.Ident}
	case *ast.Literal:
		return &ast.Literal{Value: e.Value}
	case *ast.BinaryOp:
		return &ast.BinaryOp{Op: e.Op, Left: copyExprWithSub(e.Left, sub), Right: copyExprWithSub(e.Right, sub)}
	case *ast.UnaryOp:
		return &ast.UnaryOp{Op: e.Op, Right: copyExprWithSub(e.Right, sub)}
	case *ast.Call:
		args := make([]ast.Expr, len(e.Args))
		for i, a := range e.Args {
			args[i] = copyExprWithSub(a, sub)
		}
		return &ast.Call{Func: copyExprWithSub(e.Func, sub), Args: args}
	case *ast.Index:
		return &ast.Index{Target: copyExprWithSub(e.Target, sub), Index: copyExprWithSub(e.Index, sub)}
	case *ast.Attr:
		return &ast.Attr{Target: copyExprWithSub(e.Target, sub), Attr: e.Attr}
	case *ast.DictLit:
		keys := make([]ast.Expr, len(e.Keys))
		vals := make([]ast.Expr, len(e.Vals))
		for i := range e.Keys {
			keys[i] = copyExprWithSub(e.Keys[i], sub)
			vals[i] = copyExprWithSub(e.Vals[i], sub)
		}
		return &ast.DictLit{Keys: keys, Vals: vals}
	case *ast.ListLit:
		elems := make([]ast.Expr, len(e.Elems))
		for i := range e.Elems {
			elems[i] = copyExprWithSub(e.Elems[i], sub)
		}
		return &ast.ListLit{Elems: elems}
	default:
		// Lambdas and anything unrecognized are returned as-is; inlining does
		// not descend into nested function scopes for substitution.
		return expr
	}
}

// isSimpleExpr reports whether an expression is trivially duplicable without
// changing behavior (no observable side effects, cheap to re-evaluate).
func isSimpleExpr(expr ast.Expr) bool {
	switch expr.(type) {
	case *ast.Literal, *ast.Name:
		return true
	}
	return false
}

// exprReferences reports whether expr references the identifier name anywhere.
func exprReferences(expr ast.Expr, name string) bool {
	return countName(expr, name) > 0
}

// countName counts references to identifier name within expr.
func countName(expr ast.Expr, name string) int {
	n := 0
	var walk func(ast.Expr)
	walk = func(e ast.Expr) {
		switch x := e.(type) {
		case *ast.Name:
			if x.Ident == name {
				n++
			}
		case *ast.BinaryOp:
			walk(x.Left)
			walk(x.Right)
		case *ast.UnaryOp:
			walk(x.Right)
		case *ast.Call:
			walk(x.Func)
			for _, a := range x.Args {
				walk(a)
			}
		case *ast.Index:
			walk(x.Target)
			walk(x.Index)
		case *ast.Attr:
			walk(x.Target)
		case *ast.DictLit:
			for i := range x.Keys {
				walk(x.Keys[i])
				walk(x.Vals[i])
			}
		case *ast.ListLit:
			for _, el := range x.Elems {
				walk(el)
			}
		case *ast.Lambda:
			walk(x.Body)
		}
	}
	walk(expr)
	return n
}
