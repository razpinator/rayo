package sem

import (
	"rayo/internal/ast"
	"rayo/internal/diag"
)

// Scope represents a lexical scope and its symbol table. Alongside the inferred
// type of each binding it tracks whether a name has been used (for unused-var
// warnings) and whether it is definitely assigned (for use-before-assignment
// diagnostics).
type Scope struct {
	Parent   *Scope
	Symbols  map[string]Type
	Used     map[string]bool
	Assigned map[string]bool
	DeclSpan map[string]diag.Span
}

func NewScope(parent *Scope) *Scope {
	return &Scope{
		Parent:   parent,
		Symbols:  map[string]Type{},
		Used:     map[string]bool{},
		Assigned: map[string]bool{},
		DeclSpan: map[string]diag.Span{},
	}
}

// lookup resolves name to its inferred type, walking up the scope chain. It
// satisfies the resolver interface used by InferType.
func (s *Scope) lookup(name string) (Type, bool) {
	for sc := s; sc != nil; sc = sc.Parent {
		if t, ok := sc.Symbols[name]; ok {
			return t, true
		}
	}
	return nil, false
}

// markUsed records a use of name in the nearest scope that declares it.
func (s *Scope) markUsed(name string) {
	for sc := s; sc != nil; sc = sc.Parent {
		if _, ok := sc.Symbols[name]; ok {
			sc.Used[name] = true
			return
		}
	}
}

// isAssigned reports whether name is definitely assigned in scope.
func (s *Scope) isAssigned(name string) bool {
	for sc := s; sc != nil; sc = sc.Parent {
		if _, ok := sc.Symbols[name]; ok {
			return sc.Assigned[name]
		}
	}
	return false
}

// define declares name with an inferred type and an assigned flag.
func (s *Scope) define(name string, t Type, assigned bool, span diag.Span) {
	s.Symbols[name] = t
	if _, seen := s.Used[name]; !seen {
		s.Used[name] = false
	}
	s.Assigned[name] = assigned
	s.DeclSpan[name] = span
}

// checker carries the reporter through the walk so we no longer rely on a
// package-level global (which was not safe for concurrent use).
type checker struct {
	rep diag.Reporter
}

// CheckModule performs semantic checks on a module: type inference, null-safety,
// definite assignment, must-return analysis, and unused-variable warnings.
func CheckModule(mod *ast.Module, rep diag.Reporter) {
	c := &checker{rep: rep}
	scope := NewScope(nil)
	for _, stmt := range mod.Body {
		c.checkStmt(stmt, scope)
	}
	c.reportUnused(scope)
}

// reportUnused emits a warning for each declared-but-never-used binding.
func (c *checker) reportUnused(scope *Scope) {
	for name, used := range scope.Used {
		if !used {
			diag.ReportAt(c.rep, scope.DeclSpan[name], diag.SeverityWarning, "unused variable: "+name)
		}
	}
}

func (c *checker) checkStmt(stmt ast.Stmt, scope *Scope) {
	switch s := stmt.(type) {
	case *ast.FuncDef:
		c.checkFuncDef(s, scope)

	case *ast.ClassDef:
		c.checkClassDef(s, scope)

	case *ast.VarStmt:
		var typ Type = &AnyType{}
		if s.Value != nil {
			c.checkExpr(s.Value, scope)
			typ = InferTypeIn(s.Value, scope)
		}
		scope.define(s.Name, typ, s.Value != nil, s.Span())

	case *ast.AssignStmt:
		if s.Value != nil {
			c.checkExpr(s.Value, scope)
		}
		if name, ok := s.Target.(*ast.Name); ok {
			// Assignment both declares (if new) and satisfies definite
			// assignment for the target name.
			t := InferTypeIn(s.Value, scope)
			if _, exists := scope.lookup(name.Ident); exists {
				scope.markUsed(name.Ident)
			}
			scope.define(name.Ident, t, true, s.Span())
			scope.Used[name.Ident] = true
		} else {
			// Indexed/attribute targets: the base is being used.
			c.checkExpr(s.Target, scope)
		}

	case *ast.IfStmt:
		c.checkExpr(s.Cond, scope)
		c.checkBlock(s.Then, scope)
		for _, elif := range s.Elifs {
			c.checkExpr(elif.Cond, scope)
			c.checkBlock(elif.Body, scope)
		}
		c.checkBlock(s.Else, scope)

	case *ast.WhileStmt:
		c.checkExpr(s.Cond, scope)
		c.checkBlock(s.Body, scope)

	case *ast.ForStmt:
		child := NewScope(scope)
		child.define(s.Var, &AnyType{}, true, s.Span())
		c.checkExpr(s.Iter, scope)
		c.checkStmts(s.Body, child)
		child.Used[s.Var] = true // loop variable is implicitly consumed
		c.reportUnused(child)

	case *ast.ReturnStmt:
		if s.Value != nil {
			c.checkExpr(s.Value, scope)
		}

	case *ast.TryStmt:
		c.checkBlock(s.Body, scope)
		for _, exc := range s.Excepts {
			child := NewScope(scope)
			if exc.Var != "" {
				child.define(exc.Var, &AnyType{}, true, exc.Span())
				child.Used[exc.Var] = true
			}
			c.checkStmts(exc.Body, child)
			c.reportUnused(child)
		}
		c.checkBlock(s.Finally, scope)

	case *ast.MatchStmt:
		c.checkExpr(s.Subject, scope)
		for _, cs := range s.Cases {
			child := NewScope(scope)
			if cs.Pattern != nil {
				c.checkExpr(cs.Pattern, scope)
			}
			if cs.Binding != "" {
				// A capture binds the subject to a name inside the arm.
				child.define(cs.Binding, &AnyType{}, true, cs.Span())
				child.Used[cs.Binding] = true
			}
			c.checkStmts(cs.Body, child)
			c.reportUnused(child)
		}

	case *ast.ExprStmt:
		c.checkExpr(s.Expr, scope)
	}
}

// checkFuncDef checks a function body in a fresh child scope and enforces
// must-return when the body produces values on some path.
func (c *checker) checkFuncDef(fn *ast.FuncDef, parent *Scope) {
	// Decorator expressions are evaluated in the enclosing scope.
	for _, d := range fn.Decorators {
		c.checkExpr(d, parent)
	}
	scope := NewScope(parent)
	for _, p := range fn.Params {
		scope.define(p.Name, &AnyType{}, true, p.Span())
		scope.Used[p.Name] = true // parameters need not be used
	}
	c.checkStmts(fn.Body, scope)
	c.reportUnused(scope)

	// If any path returns a value, all paths must return.
	if returnsValue(fn.Body) && !MustReturn(fn.Body) {
		diag.ReportAt(c.rep, fn.Span(), diag.SeverityError,
			"missing return: not all paths in '"+fn.Name+"' return a value")
	}
}

// checkClassDef registers the class name as a known symbol and checks each
// method and property body. Methods are checked like functions; the class's
// fields are made visible via `self`, so member access inside methods is not
// flagged. Must-return is intentionally not enforced for property getters
// (they are lowered with a synthetic zero-value return).
func (c *checker) checkClassDef(cls *ast.ClassDef, parent *Scope) {
	// The class name becomes a value in scope (constructor calls reference it).
	parent.define(cls.Name, &AnyType{}, true, cls.Span())
	parent.Used[cls.Name] = true

	for _, m := range cls.Methods {
		c.checkFuncDef(m, parent)
	}
	for _, prop := range cls.Properties {
		getScope := NewScope(parent)
		getScope.define("self", &AnyType{}, true, prop.Span())
		getScope.Used["self"] = true
		c.checkStmts(prop.Get, getScope)
		c.reportUnused(getScope)
		if len(prop.Set) > 0 {
			setScope := NewScope(parent)
			setScope.define("self", &AnyType{}, true, prop.Span())
			setScope.Used["self"] = true
			setParam := prop.SetParam
			if setParam == "" {
				setParam = "value"
			}
			setScope.define(setParam, &AnyType{}, true, prop.Span())
			setScope.Used[setParam] = true
			c.checkStmts(prop.Set, setScope)
			c.reportUnused(setScope)
		}
	}
}

// checkStmts checks statements in the given scope (no new scope).
func (c *checker) checkStmts(stmts []ast.Stmt, scope *Scope) {
	for _, stmt := range stmts {
		c.checkStmt(stmt, scope)
	}
}

// checkBlock checks a nested block; bindings introduced here are scoped to it.
func (c *checker) checkBlock(stmts []ast.Stmt, parent *Scope) {
	if len(stmts) == 0 {
		return
	}
	scope := NewScope(parent)
	c.checkStmts(stmts, scope)
	c.reportUnused(scope)
}

// checkExpr walks an expression, marking uses, flagging use-before-assignment,
// and running null-safety checks on every member/index access.
func (c *checker) checkExpr(expr ast.Expr, scope *Scope) {
	switch e := expr.(type) {
	case *ast.Name:
		scope.markUsed(e.Ident)
		if _, declared := scope.lookup(e.Ident); declared && !scope.isAssigned(e.Ident) {
			diag.ReportAt(c.rep, e.Span(), diag.SeverityError,
				"use of unassigned variable: "+e.Ident)
		}
	case *ast.Attr:
		c.checkExpr(e.Target, scope)
		if IsOptional(InferTypeIn(e.Target, scope)) {
			diag.ReportAt(c.rep, e.Span(), diag.SeverityError,
				"unsafe dereference of optional value")
		}
	case *ast.Index:
		c.checkExpr(e.Target, scope)
		c.checkExpr(e.Index, scope)
		if IsOptional(InferTypeIn(e.Target, scope)) {
			diag.ReportAt(c.rep, e.Span(), diag.SeverityError,
				"unsafe index of optional value")
		}
	case *ast.Call:
		c.checkExpr(e.Func, scope)
		for _, arg := range e.Args {
			c.checkExpr(arg, scope)
		}
	case *ast.BinaryOp:
		c.checkExpr(e.Left, scope)
		c.checkExpr(e.Right, scope)
	case *ast.UnaryOp:
		c.checkExpr(e.Right, scope)
	case *ast.DictLit:
		for i := range e.Keys {
			c.checkExpr(e.Keys[i], scope)
			c.checkExpr(e.Vals[i], scope)
		}
	case *ast.ListLit:
		for _, elem := range e.Elems {
			c.checkExpr(elem, scope)
		}
	case *ast.Lambda:
		child := NewScope(scope)
		for _, p := range e.Params {
			child.define(p.Name, &AnyType{}, true, p.Span())
			child.Used[p.Name] = true
		}
		c.checkExpr(e.Body, child)
	}
}

// returnsValue reports whether any path in stmts returns a value (as opposed to
// a bare `return`). Void functions are not subject to must-return.
func returnsValue(stmts []ast.Stmt) bool {
	found := false
	visitor := &returnFinder{}
	for _, s := range stmts {
		ast.Walk(visitor, s)
		if visitor.found {
			found = true
			break
		}
	}
	return found
}

type returnFinder struct{ found bool }

func (r *returnFinder) Visit(n ast.Node) bool {
	if r.found {
		return false
	}
	if ret, ok := n.(*ast.ReturnStmt); ok && ret.Value != nil {
		r.found = true
		return false
	}
	// Do not descend into nested function definitions; their returns belong to
	// them, not the enclosing function.
	if _, ok := n.(*ast.FuncDef); ok {
		return false
	}
	return true
}
