// Package opt implements AST-to-AST optimization passes that run between
// semantic analysis and code generation.
//
// The first pass is constant folding: expressions whose operands are all
// literals are evaluated at compile time and replaced with a single literal.
// This shrinks the generated Go and is a safe, purely local rewrite (Rayo
// literals are immutable and side-effect free). Folding is intentionally
// conservative — it only folds operations whose result is unambiguous under
// Rayo's semantics and never changes a program's observable behavior.
package opt

import "rayo/internal/ast"

// FoldModule constant-folds every expression in the module in place and returns
// it for convenience.
func FoldModule(mod *ast.Module) *ast.Module {
	if mod == nil {
		return nil
	}
	mod.Body = foldStmts(mod.Body)
	return mod
}

func foldStmts(stmts []ast.Stmt) []ast.Stmt {
	for i := range stmts {
		stmts[i] = foldStmt(stmts[i])
	}
	return stmts
}

func foldStmt(stmt ast.Stmt) ast.Stmt {
	switch s := stmt.(type) {
	case *ast.FuncDef:
		s.Body = foldStmts(s.Body)
	case *ast.VarStmt:
		s.Value = foldExpr(s.Value)
	case *ast.AssignStmt:
		s.Target = foldExpr(s.Target)
		s.Value = foldExpr(s.Value)
	case *ast.IfStmt:
		s.Cond = foldExpr(s.Cond)
		s.Then = foldStmts(s.Then)
		for _, e := range s.Elifs {
			e.Cond = foldExpr(e.Cond)
			e.Body = foldStmts(e.Body)
		}
		s.Else = foldStmts(s.Else)
	case *ast.WhileStmt:
		s.Cond = foldExpr(s.Cond)
		s.Body = foldStmts(s.Body)
	case *ast.ForStmt:
		s.Iter = foldExpr(s.Iter)
		s.Body = foldStmts(s.Body)
	case *ast.ReturnStmt:
		s.Value = foldExpr(s.Value)
	case *ast.TryStmt:
		s.Body = foldStmts(s.Body)
		for _, exc := range s.Excepts {
			exc.Body = foldStmts(exc.Body)
		}
		s.Finally = foldStmts(s.Finally)
	case *ast.ClassDef:
		for _, m := range s.Methods {
			m.Body = foldStmts(m.Body)
		}
		for _, p := range s.Properties {
			p.Get = foldStmts(p.Get)
			p.Set = foldStmts(p.Set)
		}
	case *ast.MatchStmt:
		s.Subject = foldExpr(s.Subject)
		for _, cs := range s.Cases {
			if cs.Pattern != nil {
				cs.Pattern = foldExpr(cs.Pattern)
			}
			cs.Body = foldStmts(cs.Body)
		}
	case *ast.ExprStmt:
		s.Expr = foldExpr(s.Expr)
	}
	return stmt
}

// foldExpr recursively folds an expression, returning either a new literal (if
// the whole subtree evaluates to a constant) or the original node with its
// children folded.
func foldExpr(expr ast.Expr) ast.Expr {
	if expr == nil {
		return nil
	}
	switch e := expr.(type) {
	case *ast.UnaryOp:
		e.Right = foldExpr(e.Right)
		if lit, ok := e.Right.(*ast.Literal); ok {
			if folded, ok := foldUnary(e.Op, lit.Value); ok {
				return ast.SetSpan(&ast.Literal{Value: folded}, e.Span())
			}
		}
		return e
	case *ast.BinaryOp:
		e.Left = foldExpr(e.Left)
		e.Right = foldExpr(e.Right)
		lhs, lok := e.Left.(*ast.Literal)
		rhs, rok := e.Right.(*ast.Literal)
		if lok && rok {
			if folded, ok := foldBinary(e.Op, lhs.Value, rhs.Value); ok {
				return ast.SetSpan(&ast.Literal{Value: folded}, e.Span())
			}
		}
		return e
	case *ast.Call:
		e.Func = foldExpr(e.Func)
		for i := range e.Args {
			e.Args[i] = foldExpr(e.Args[i])
		}
		return e
	case *ast.Index:
		e.Target = foldExpr(e.Target)
		e.Index = foldExpr(e.Index)
		return e
	case *ast.Attr:
		e.Target = foldExpr(e.Target)
		return e
	case *ast.DictLit:
		for i := range e.Keys {
			e.Keys[i] = foldExpr(e.Keys[i])
			e.Vals[i] = foldExpr(e.Vals[i])
		}
		return e
	case *ast.ListLit:
		for i := range e.Elems {
			e.Elems[i] = foldExpr(e.Elems[i])
		}
		return e
	case *ast.Lambda:
		e.Body = foldExpr(e.Body)
		return e
	default:
		// Literal, Name: nothing to fold.
		return expr
	}
}

// foldUnary evaluates a unary operator over a literal operand. It returns
// ok=false when the operation is not foldable for the given operand type.
func foldUnary(op string, v any) (any, bool) {
	switch op {
	case "-":
		switch x := v.(type) {
		case int:
			return -x, true
		case int64:
			return -x, true
		case float64:
			return -x, true
		}
	case "+":
		switch v.(type) {
		case int, int64, float64:
			return v, true
		}
	case "not":
		return !truthy(v), true
	}
	return nil, false
}

// foldBinary evaluates a binary operator over two literal operands.
func foldBinary(op string, a, b any) (any, bool) {
	// String concatenation and comparison.
	if as, ok := a.(string); ok {
		bs, ok := b.(string)
		if !ok {
			return nil, false
		}
		switch op {
		case "+":
			return as + bs, true
		case "==":
			return as == bs, true
		case "!=":
			return as != bs, true
		case "<":
			return as < bs, true
		case "<=":
			return as <= bs, true
		case ">":
			return as > bs, true
		case ">=":
			return as >= bs, true
		}
		return nil, false
	}

	// Boolean logic.
	if ab, ok := a.(bool); ok {
		bb, ok := b.(bool)
		if !ok {
			return nil, false
		}
		switch op {
		case "and":
			return ab && bb, true
		case "or":
			return ab || bb, true
		case "==":
			return ab == bb, true
		case "!=":
			return ab != bb, true
		}
		return nil, false
	}

	// Numeric arithmetic and comparison. If either operand is a float, compute
	// in float; otherwise stay in integer space.
	af, aok := toFloat(a)
	bf, bok := toFloat(b)
	if !aok || !bok {
		return nil, false
	}
	_, aIsInt := toInt(a)
	_, bIsInt := toInt(b)
	bothInt := aIsInt && bIsInt

	switch op {
	case "+", "-", "*":
		if bothInt {
			ai, _ := toInt(a)
			bi, _ := toInt(b)
			switch op {
			case "+":
				return ai + bi, true
			case "-":
				return ai - bi, true
			case "*":
				return ai * bi, true
			}
		}
		switch op {
		case "+":
			return af + bf, true
		case "-":
			return af - bf, true
		case "*":
			return af * bf, true
		}
	case "/":
		// Division by zero is left unfolded so the runtime raises it, matching
		// unoptimized behavior.
		if bf == 0 {
			return nil, false
		}
		if bothInt {
			ai, _ := toInt(a)
			bi, _ := toInt(b)
			return ai / bi, true
		}
		return af / bf, true
	case "%":
		if bothInt {
			bi, _ := toInt(b)
			if bi == 0 {
				return nil, false
			}
			ai, _ := toInt(a)
			return ai % bi, true
		}
		return nil, false
	case "==":
		return af == bf, true
	case "!=":
		return af != bf, true
	case "<":
		return af < bf, true
	case "<=":
		return af <= bf, true
	case ">":
		return af > bf, true
	case ">=":
		return af >= bf, true
	}
	return nil, false
}

// truthy implements Rayo's truthiness for the literal values folding can see.
func truthy(v any) bool {
	switch x := v.(type) {
	case nil:
		return false
	case bool:
		return x
	case int:
		return x != 0
	case int64:
		return x != 0
	case float64:
		return x != 0
	case string:
		return x != ""
	}
	return true
}

func toFloat(v any) (float64, bool) {
	switch x := v.(type) {
	case int:
		return float64(x), true
	case int64:
		return float64(x), true
	case float64:
		return x, true
	}
	return 0, false
}

func toInt(v any) (int, bool) {
	switch x := v.(type) {
	case int:
		return x, true
	case int64:
		return int(x), true
	}
	return 0, false
}
